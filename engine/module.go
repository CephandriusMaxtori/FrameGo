package engine

import (
	"image"
	"time"

	"framego/render"
)

// State describes the lifecycle stage of a module.
type State int

const (
	// StateUninitialized: configuration loaded, background work inactive.
	StateUninitialized State = iota
	// StateActive: module processing events and producing frames.
	StateActive
	// StateSuspended: module throttled or hidden by visibility/power rules.
	StateSuspended
	// StateFaulted: module panicked; isolated and rendering a degraded state.
	StateFaulted
)

// String renders the state name for logs and the status API.
func (s State) String() string {
	switch s {
	case StateUninitialized:
		return "uninitialized"
	case StateActive:
		return "active"
	case StateSuspended:
		return "suspended"
	case StateFaulted:
		return "faulted"
	default:
		return "unknown"
	}
}

// Module is the unified widget contract every FrameGo module implements.
// Modules own their timing and data fetching; the supervisor isolates faults.
type Module interface {
	// Name returns the unique module identifier.
	Name() string
	// Configure applies module options; called at start and on hot reload.
	Configure(opts map[string]any) error
	// Start begins background work (goroutines, subscriptions).
	Start(bus *Bus, log *Logger) error
	// Stop halts background work.
	Stop() error
	// Draw renders the module's frame into the given zone bounds at time now.
	Draw(c *render.Canvas, bounds image.Rectangle, now time.Time) error
}

// CreateFunc instantiates a module by registered name.
type CreateFunc func(name string) (Module, bool)
