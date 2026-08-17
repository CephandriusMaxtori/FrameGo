//go:build linux && (amd64 || arm64)

// Package input provides input device readers for the FrameGo kiosk engine.
// On Linux it reads touch events from evdev (typically /dev/input/event0)
// and publishes them on the engine event bus.
package input

import (
	"encoding/binary"
	"fmt"
	"sync"
	"syscall"
	"time"

	"framego/engine"
)

const (
	evSyn      = 0x00
	evKey      = 0x01
	evAbs      = 0x03
	synReport  = 0x00
	btnTouch   = 0x14a
	absMtPosX = 0x35
	absMtPosY = 0x36
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
// them on the engine bus.
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
	return &Evdev{
		path:   path,
		bus:    bus,
		log:    log,
		fd:     fd,
		width:  width,
		height: height,
		done:   make(chan struct{}),
	}, nil
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
		if ev.Code == absMtPosX {
			e.mu.Lock()
			e.lastX = int(ev.Value)
			e.haveX = true
			e.mu.Unlock()
		} else if ev.Code == absMtPosY {
			e.mu.Lock()
			x, _ := e.lastX, e.lastY
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

	case evKey:
		if ev.Code == btnTouch && ev.Value == 0 {
			e.mu.Lock()
			e.pressed = false
			x, y := e.lastX, e.lastY
			e.mu.Unlock()
			sx, sy := x, y
			if e.width > 0 && e.height > 0 {
				sx = x * e.width / 4096
				sy = y * e.height / 4096
			}
			e.bus.Publish(engine.TopicTouch, TouchEvent{X: sx, Y: sy, Phase: TouchUp})
		}
	}
}
