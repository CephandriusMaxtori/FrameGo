//go:build !(linux && (amd64 || arm64))

package input

import "framego/engine"

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

// Evdev is a no-op on non-Linux platforms.
type Evdev struct{}

// NewEvdev returns an error on non-Linux platforms.
func NewEvdev(_ string, _ *engine.Bus, _ *engine.Logger, _, _ int) (*Evdev, error) {
	return nil, ErrNotSupported
}

// ErrNotSupported is returned when evdev input is not available.
var ErrNotSupported = errNotSupported{}

type errNotSupported struct{}

func (errNotSupported) Error() string { return "evdev input is only supported on Linux (amd64/arm64)" }

// ErrNoTouchDevice is returned when no touch-capable device is found.
var ErrNoTouchDevice = errNoTouchDevice{}

type errNoTouchDevice struct{}

func (errNoTouchDevice) Error() string { return "no touch-capable input device found" }

func (e *Evdev) Start() {}
func (e *Evdev) Stop()  {}

// AutoTouchDevice always returns ErrNotSupported on non-Linux platforms.
func AutoTouchDevice(_ *engine.Logger) (string, error) {
	return "", ErrNotSupported
}

// ListTouchDevices always returns nil on non-Linux platforms.
func ListTouchDevices() []string { return nil }

// DeviceName always returns empty string on non-Linux platforms.
func DeviceName(_ string) string { return "" }
