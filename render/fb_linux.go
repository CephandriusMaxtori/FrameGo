//go:build linux && (amd64 || arm64)

package render

import (
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"sync"
	"unsafe"

	"golang.org/x/sys/unix"
)

// Framebuffer ioctl request codes from include/uapi/linux/fb.h. They are plain
// constants (not _IOC-encoded), so they are stable across Linux architectures.
const (
	fbIoctlGetVScreenInfo = 0x4600
	fbIoctlGetFScreenInfo = 0x4602
)

// FbBitfield describes a color channel's layout within a pixel.
type FbBitfield struct {
	Offset   uint32
	Length   uint32
	MsbRight uint32
}

// FbVarScreeninfo mirrors struct fb_var_screeninfo for LP64 Linux targets.
type FbVarScreeninfo struct {
	Xres, Yres, XresVirtual, YresVirtual uint32
	Xoffset, Yoffset                     uint32
	BitsPerPixel                         uint32
	Grayscale                            uint32
	Red, Green, Blue, Transp             FbBitfield
	Nonstd, Activate                     uint32
	Height, Width                        uint32
	AccelFlags                           uint32
	Pixclock, LeftMargin, RightMargin    uint32
	UpperMargin, LowerMargin             uint32
	HsyncLen, VsyncLen, Sync, Vmode      uint32
	Rotate, Colorspace                   uint32
	Reserved                             [4]uint32
}

// FbFixScreeninfo mirrors struct fb_fix_screeninfo for LP64 Linux targets.
type FbFixScreeninfo struct {
	ID           [16]byte
	SmemStart    uint64
	SmemLen      uint32
	Type         uint32
	TypeAux      uint32
	Visual       uint32
	XPanstep     uint16
	YPanstep     uint16
	Ywrapstep    uint16
	LineLength   uint32
	MmioStart    uint64
	MmioLen      uint32
	Accel        uint32
	Capabilities uint16
	Reserved     [2]uint16
}

// Framebuffer is a direct-to-kernel Linux framebuffer display target. It opens
// the display device (typically /dev/fb0), reads its geometry via ioctl, and
// maps the screen memory for zero-copy presentation. Pure Go — no cgo.
type Framebuffer struct {
	path string
	fd   int
	fb   []byte
	vi   FbVarScreeninfo
	fi   FbFixScreeninfo
	mu   sync.Mutex
}

// NewFramebuffer opens and maps the Linux framebuffer device.
func NewFramebuffer(path string) (*Framebuffer, error) {
	if path == "" {
		path = "/dev/fb0"
	}
	fd, err := unix.Open(path, unix.O_RDWR, 0)
	if err != nil {
		return nil, fmt.Errorf("framebuffer %s: %w", path, err)
	}
	f := &Framebuffer{path: path, fd: fd}
	if err := f.probe(); err != nil {
		_ = unix.Close(fd)
		return nil, err
	}
	return f, nil
}

// probe queries the variable/fixed screen info and maps the framebuffer.
func (f *Framebuffer) probe() error {
	if _, _, errno := unix.Syscall(unix.SYS_IOCTL, uintptr(f.fd), fbIoctlGetVScreenInfo, uintptr(unsafe.Pointer(&f.vi))); errno != 0 {
		return fmt.Errorf("FBIOGET_VSCREENINFO: %w", errno)
	}
	switch f.vi.BitsPerPixel {
	case 16, 24, 32:
	default:
		return fmt.Errorf("unsupported framebuffer depth %d bpp", f.vi.BitsPerPixel)
	}
	if _, _, errno := unix.Syscall(unix.SYS_IOCTL, uintptr(f.fd), fbIoctlGetFScreenInfo, uintptr(unsafe.Pointer(&f.fi))); errno != 0 {
		return fmt.Errorf("FBIOGET_FSCREENINFO: %w", errno)
	}
	fb, err := unix.Mmap(f.fd, 0, int(f.fi.SmemLen), unix.PROT_READ|unix.PROT_WRITE, unix.MAP_SHARED)
	if err != nil {
		return fmt.Errorf("mmap framebuffer: %w", err)
	}
	f.fb = fb
	return nil
}

// Present blits an RGBA frame into the mapped screen memory. The image is
// clamped to the display resolution and colors are packed per the hardware
// bitfield layout (supports 16/24/32 bpp).
func (f *Framebuffer) Present(img *image.RGBA) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.fb) == 0 {
		return errors.New("framebuffer closed")
	}
	w, h := img.Bounds().Dx(), img.Bounds().Dy()
	if vw := int(f.vi.Xres); w > vw {
		w = vw
	}
	if vh := int(f.vi.Yres); h > vh {
		h = vh
	}
	if w <= 0 || h <= 0 {
		return nil
	}
	stride := int(f.fi.LineLength)
	if stride == 0 {
		stride = int(f.vi.Xres) * int(f.vi.BitsPerPixel/8)
	}
	bpp := int(f.vi.BitsPerPixel)
	lr, lg, lb := f.vi.Red.Length, f.vi.Green.Length, f.vi.Blue.Length
	or, og, ob := f.vi.Red.Offset, f.vi.Green.Offset, f.vi.Blue.Offset

	for y := 0; y < h; y++ {
		src := img.Pix[y*img.Stride:]
		dst := f.fb[y*stride:]
		for x := 0; x < w; x++ {
			r, g, b := src[x*4], src[x*4+1], src[x*4+2]
			v := scale8(r, lr)<<or | scale8(g, lg)<<og | scale8(b, lb)<<ob
			switch bpp {
			case 32:
				binary.LittleEndian.PutUint32(dst[x*4:], v)
			case 24:
				dst[x*3] = byte(v)
				dst[x*3+1] = byte(v >> 8)
				dst[x*3+2] = byte(v >> 16)
			case 16:
				binary.LittleEndian.PutUint16(dst[x*2:], uint16(v))
			}
		}
	}
	return nil
}

// Close unmaps and closes the framebuffer device.
func (f *Framebuffer) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.fb) > 0 {
		_ = unix.Munmap(f.fb)
		f.fb = nil
	}
	if f.fd >= 0 {
		_ = unix.Close(f.fd)
		f.fd = -1
	}
	return nil
}

// scale8 maps an 8-bit channel value into a bitfield of the given length.
func scale8(v uint8, bits uint32) uint32 {
	if bits >= 8 {
		return uint32(v) << (bits - 8)
	}
	return uint32(v) >> (8 - bits)
}
