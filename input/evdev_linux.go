//go:build linux && (amd64 || arm64)

// Package input provides input device readers for the FrameGo kiosk engine.
// On Linux it reads touch events from evdev (typically /dev/input/event0)
// and publishes them on the engine event bus.
package input

import (
	"encoding/binary"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"framego/engine"
)

const (
	evSyn     = 0x00
	evKey     = 0x01
	evRel     = 0x02
	evAbs     = 0x03
	synReport = 0x00

	btnTouch = 0x14a
	btnLeft  = 0x110

	absX = 0x00
	absY = 0x01
	absMtPosX = 0x35
	absMtPosY = 0x36

	relX = 0x00
	relY = 0x01
)

// ioctl constants for evdev capability querying.
const (
	eviocgnameLen = 128
	eviocgphysLen = 128
	eviocgbitLen  = (1024 + 7) / 8
)

type inputEvent struct {
	Time  [16]byte
	Type  uint16
	Code  uint16
	Value int32
}

// TouchEvent is published on engine.TopicTouch.
type TouchEvent struct {
	X, Y  int
	Phase TouchPhase
}

// TouchPhase describes the touch lifecycle.
type TouchPhase int

const (
	TouchDown TouchPhase = iota
	TouchMove
	TouchUp
)

// Evdev reads multitouch events from a Linux evdev device and publishes
// them on the engine bus. It supports multitouch, single-touch, and
// relative mouse input for virtual framebuffer environments.
type Evdev struct {
	path   string
	bus    *engine.Bus
	log    *engine.Logger
	fd     int
	width  int
	height int

	mu      sync.Mutex
	lastX   int
	lastY   int
	curX    int
	curY    int
	pressed bool
	haveX   bool
	done    chan struct{}
	wg      sync.WaitGroup
}

// NewEvdev opens the given evdev device for touch input.
func NewEvdev(path string, bus *engine.Bus, log *engine.Logger, width, height int) (*Evdev, error) {
	if path == "" {
		path = "/dev/input/event0"
	}
	fd, err := syscall.Open(path, syscall.O_RDONLY, 0)
	if err != nil {
		return nil, fmt.Errorf("evdev %s: %w", path, err)
	}
	e := &Evdev{
		path:   path,
		bus:    bus,
		log:    log,
		fd:     fd,
		width:  width,
		height: height,
		done:   make(chan struct{}),
	}
	if width > 0 && height > 0 {
		e.curX = width / 2
		e.curY = height / 2
	}
	return e, nil
}

// Start begins reading touch events in a background goroutine.
func (e *Evdev) Start() {
	e.wg.Add(1)
	go e.readLoop()
}

// Stop halts the read loop and closes the device.
func (e *Evdev) Stop() {
	close(e.done)
	e.wg.Wait()
	_ = syscall.Close(e.fd)
}

func (e *Evdev) readLoop() {
	defer e.wg.Done()
	buf := make([]byte, 24)

	for {
		select {
		case <-e.done:
			return
		default:
		}

		n, err := syscall.Read(e.fd, buf)
		if err != nil {
			if err == syscall.EINTR {
				continue
			}
			select {
			case <-e.done:
				return
			default:
			}
			e.log.Errorf("evdev read: %v", err)
			time.Sleep(100 * time.Millisecond)
			continue
		}
		if n < 24 {
			continue
		}

		ev := inputEvent{
			Type:  binary.LittleEndian.Uint16(buf[16:18]),
			Code:  binary.LittleEndian.Uint16(buf[18:20]),
			Value: int32(binary.LittleEndian.Uint32(buf[20:24])),
		}
		e.handleEvent(ev)
	}
}

func (e *Evdev) handleEvent(ev inputEvent) {
	switch ev.Type {
	case evAbs:
		switch ev.Code {
		case absX, absMtPosX:
			e.mu.Lock()
			e.lastX = int(ev.Value)
			e.haveX = true
			e.mu.Unlock()
		case absY, absMtPosY:
			e.mu.Lock()
			x := e.lastX
			e.lastY = int(ev.Value)
			phase := TouchMove
			if !e.pressed {
				phase = TouchDown
				e.pressed = true
			}
			e.mu.Unlock()

			sx, sy := x, int(ev.Value)
			if e.width > 0 && e.height > 0 {
				sx = x * e.width / 4096
				sy = int(ev.Value) * e.height / 4096
			}
			e.bus.Publish(engine.TopicTouch, TouchEvent{X: sx, Y: sy, Phase: phase})
		}

	case evRel:
		e.mu.Lock()
		switch ev.Code {
		case relX:
			e.curX += int(ev.Value)
		case relY:
			e.curY += int(ev.Value)
		}
		if e.width > 0 {
			if e.curX < 0 {
				e.curX = 0
			} else if e.curX >= e.width {
				e.curX = e.width - 1
			}
		}
		if e.height > 0 {
			if e.curY < 0 {
				e.curY = 0
			} else if e.curY >= e.height {
				e.curY = e.height - 1
			}
		}
		sx, sy := e.curX, e.curY
		phase := TouchMove
		if !e.pressed {
			phase = TouchDown
			e.pressed = true
		}
		e.mu.Unlock()
		e.bus.Publish(engine.TopicTouch, TouchEvent{X: sx, Y: sy, Phase: phase})

	case evKey:
		if (ev.Code == btnTouch || ev.Code == btnLeft) && ev.Value == 0 {
			e.mu.Lock()
			e.pressed = false
			x, y := e.lastX, e.lastY
			e.mu.Unlock()
			sx, sy := x, y
			if e.width > 0 && e.height > 0 {
				sx = x * e.width / 4096
				sy = y * e.height / 4096
			}
			if ev.Code == btnLeft {
				e.mu.Lock()
				sx, sy = e.curX, e.curY
				e.mu.Unlock()
			}
			e.bus.Publish(engine.TopicTouch, TouchEvent{X: sx, Y: sy, Phase: TouchUp})
		}
	}
}

// ioctlQueryBits reads the event type capability bitmap for an evdev fd.
func ioctlQueryBits(fd int, evType int) ([eviocgbitLen]byte, error) {
	var bits [eviocgbitLen]byte
	eviocgbitReq := (0x45 << 24) | (0x20 + uintptr(evType)) | (uintptr(eviocgbitLen) << 16)
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), eviocgbitReq, uintptr(unsafe.Pointer(&bits)))
	if errno != 0 {
		return bits, fmt.Errorf("ioctl EVIOCGBIT(%d): %w", evType, errno)
	}
	return bits, nil
}

// hasBit reports whether bit n is set in the bitmap.
func hasBit(bits [eviocgbitLen]byte, n int) bool {
	return bits[n/8]&(1<<(n%8)) != 0
}

// AutoTouchDevice scans /dev/input/event* and returns the path of the first
// device that has BTN_TOUCH or BTN_LEFT in its key capabilities. Returns
// ErrNoTouchDevice if none is found.
func AutoTouchDevice(log *engine.Logger) (string, error) {
	matches, err := filepath.Glob("/dev/input/event*")
	if err != nil {
		return "", fmt.Errorf("glob /dev/input/event*: %w", err)
	}
	for _, path := range matches {
		fd, err := syscall.Open(path, syscall.O_RDONLY|syscall.O_NONBLOCK, 0)
		if err != nil {
			continue
		}
		bits, err := ioctlQueryBits(fd, evKey)
		_ = syscall.Close(fd)
		if err != nil {
			if log != nil {
				log.Printf("input: skip %s: %v", filepath.Base(path), err)
			}
			continue
		}
		if hasBit(bits, btnTouch) || hasBit(bits, btnLeft) {
			return path, nil
		}
	}
	return "", ErrNoTouchDevice
}

// ErrNoTouchDevice is returned when no touch-capable device is found.
var ErrNoTouchDevice = errNoTouchDevice{}

type errNoTouchDevice struct{}

func (errNoTouchDevice) Error() string { return "no touch-capable input device found" }

// ListTouchDevices returns all detected touch-capable device paths for diagnostics.
func ListTouchDevices() []string {
	var paths []string
	matches, _ := filepath.Glob("/dev/input/event*")
	for _, path := range matches {
		fd, err := syscall.Open(path, syscall.O_RDONLY|syscall.O_NONBLOCK, 0)
		if err != nil {
			continue
		}
		bits, err := ioctlQueryBits(fd, evKey)
		_ = syscall.Close(fd)
		if err != nil {
			continue
		}
		if hasBit(bits, btnTouch) || hasBit(bits, btnLeft) {
			paths = append(paths, path)
		}
	}
	return paths
}

// DeviceName returns the human-readable name for an evdev device, or empty string.
func DeviceName(path string) string {
	fd, err := syscall.Open(path, syscall.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		return ""
	}
	defer syscall.Close(fd)
	var name [eviocgnameLen]byte
	req := (0x45 << 24) | 0x06 | (uintptr(eviocgnameLen) << 16)
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), req, uintptr(unsafe.Pointer(&name)))
	if errno != 0 {
		return ""
	}
	return strings.TrimRight(string(name[:]), "\x00")
}

