package engine

import (
	"fmt"
	"image"
	"image/color"
	"runtime/debug"
	"sort"
	"sync"
	"time"

	"framego/fonts"
	"framego/render"
)

// ModuleStatus is the externally visible snapshot of a managed module.
type ModuleStatus struct {
	Name      string    `json:"name"`
	State     string    `json:"state"`
	StartedAt time.Time `json:"started_at,omitempty"`
	Error     string    `json:"error,omitempty"`
}

type managed struct {
	mod       Module
	mu        sync.Mutex
	state     State
	startedAt time.Time
	err       error
}

func (m *managed) setState(s State) {
	m.mu.Lock()
	m.state = s
	m.mu.Unlock()
}

func (m *managed) setFaulted(err error) {
	m.mu.Lock()
	m.state = StateFaulted
	m.err = err
	m.mu.Unlock()
}

func (m *managed) stateOf() State {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.state
}

func (m *managed) errorOf() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err == nil {
		return ""
	}
	return m.err.Error()
}

// Supervisor owns module lifecycle, fault isolation, and panic recovery.
// Individual module faults are contained: a panic marks the module Faulted and
// the primary render loop continues running.
type Supervisor struct {
	bus     *Bus
	log     *Logger
	mu      sync.RWMutex
	modules map[string]*managed
}

// NewSupervisor creates a supervisor bound to the given bus and logger.
func NewSupervisor(bus *Bus, log *Logger) *Supervisor {
	return &Supervisor{
		bus:     bus,
		log:     log,
		modules: make(map[string]*managed),
	}
}

// Add configures and registers a module without starting it.
func (s *Supervisor) Add(m Module, opts map[string]any) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	name := m.Name()
	if _, ok := s.modules[name]; ok {
		return fmt.Errorf("module %q already registered", name)
	}
	if err := m.Configure(opts); err != nil {
		return fmt.Errorf("module %q configure: %w", name, err)
	}
	s.modules[name] = &managed{mod: m, state: StateUninitialized}
	return nil
}

// Start activates a registered module.
func (s *Supervisor) Start(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	m, ok := s.modules[name]
	if !ok {
		return fmt.Errorf("module %q not registered", name)
	}
	s.guarded(name, m, func() { _ = m.mod.Start(s.bus, s.log) })
	if m.stateOf() != StateFaulted {
		m.mu.Lock()
		m.state = StateActive
		m.startedAt = time.Now()
		m.err = nil
		m.mu.Unlock()
		s.bus.Publish(TopicModuleState, map[string]any{"module": name, "state": StateActive.String()})
	}
	return nil
}

// StartAll activates every registered module.
func (s *Supervisor) StartAll() {
	for _, name := range s.Names() {
		if err := s.Start(name); err != nil {
			s.log.Errorf("start %s: %v", name, err)
		}
	}
}

// Stop suspends a running module.
func (s *Supervisor) Stop(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	m, ok := s.modules[name]
	if !ok {
		return fmt.Errorf("module %q not registered", name)
	}
	if m.stateOf() == StateActive {
		s.guarded(name, m, func() { _ = m.mod.Stop() })
	}
	m.setState(StateSuspended)
	s.bus.Publish(TopicModuleState, map[string]any{"module": name, "state": StateSuspended.String()})
	return nil
}

// Remove stops (if active) and unregisters a module.
func (s *Supervisor) Remove(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	m, ok := s.modules[name]
	if !ok {
		return nil
	}
	if m.stateOf() == StateActive {
		s.guarded(name, m, func() { _ = m.mod.Stop() })
	}
	delete(s.modules, name)
	s.bus.Publish(TopicModuleState, map[string]any{"module": name, "state": "removed"})
	return nil
}

// Reconfigure reapplies options to a live module.
func (s *Supervisor) Reconfigure(name string, opts map[string]any) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	m, ok := s.modules[name]
	if !ok {
		return fmt.Errorf("module %q not registered", name)
	}
	if err := m.mod.Configure(opts); err != nil {
		return fmt.Errorf("module %q reconfigure: %w", name, err)
	}
	return nil
}

// State returns the current state of a module.
func (s *Supervisor) State(name string) State {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if m, ok := s.modules[name]; ok {
		return m.stateOf()
	}
	return StateUninitialized
}

// Names lists registered module names in sorted order.
func (s *Supervisor) Names() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	names := make([]string, 0, len(s.modules))
	for n := range s.modules {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// Has reports whether name is registered.
func (s *Supervisor) Has(name string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.modules[name]
	return ok
}

// StopAll suspends every running module.
func (s *Supervisor) StopAll() {
	for _, name := range s.Names() {
		_ = s.Stop(name)
	}
}

// Status returns externally visible state for every module.
func (s *Supervisor) Status() []ModuleStatus {
	s.mu.RLock()
	mods := make([]*managed, 0, len(s.modules))
	for _, m := range s.modules {
		mods = append(mods, m)
	}
	s.mu.RUnlock()
	out := make([]ModuleStatus, 0, len(mods))
	for _, m := range mods {
		m.mu.Lock()
		st := ModuleStatus{Name: m.mod.Name(), State: m.state.String(), StartedAt: m.startedAt}
		if m.err != nil {
			st.Error = m.err.Error()
		}
		m.mu.Unlock()
		out = append(out, st)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// DrawAll renders every module mapped to a rect. Faulted modules draw a
// degraded placeholder instead of their content.
func (s *Supervisor) DrawAll(c *render.Canvas, rects map[string]image.Rectangle, now time.Time) {
	s.mu.RLock()
	type item struct {
		name  string
		man   *managed
		state State
	}
	items := make([]item, 0, len(rects))
	for name := range rects {
		m, ok := s.modules[name]
		if !ok || m.stateOf() == StateSuspended {
			continue
		}
		items = append(items, item{name: name, man: m, state: m.stateOf()})
	}
	s.mu.RUnlock()

	for _, it := range items {
		rect := rects[it.name]
		if it.state == StateFaulted {
			drawFaulted(c, rect, it.name)
			continue
		}
		if it.state != StateActive {
			continue
		}
		mod := it.man.mod
		s.guarded(it.name, it.man, func() { _ = mod.Draw(c, rect, now) })
	}
}

// guarded runs fn and isolates any panic into a Faulted state on name. If m is
// non-nil it is marked faulted on panic; callers holding the supervisor lock
// must pass the managed module. Panic marking uses the per-module mutex, so
// guarded never re-acquires the supervisor lock (avoiding RW-lock deadlock).
func (s *Supervisor) guarded(name string, m *managed, fn func()) {
	defer func() {
		if r := recover(); r != nil {
			if m != nil {
				m.setFaulted(fmt.Errorf("panic: %v\n%s", r, debug.Stack()))
			}
			s.log.Errorf("module %s faulted: %v", name, r)
			s.bus.Publish(TopicModuleState, map[string]any{"module": name, "state": StateFaulted.String()})
		}
	}()
	fn()
}

// drawFaulted paints a degraded placeholder for a faulted module.
func drawFaulted(c *render.Canvas, rect image.Rectangle, name string) {
	c.FillRect(rect, render.ParseHexColor("#2a1418", color.RGBA{R: 42, G: 20, B: 24, A: 255}))
	dim := render.ParseHexColor("#e28b94", color.RGBA{R: 226, G: 139, B: 148, A: 255})
	df := fonts.Face(18, fonts.Regular)
	ascent, _, height := c.FaceMetrics(df)
	label := name + " [FAULTED]"
	w, _ := c.TextSize(df, label)
	x := rect.Min.X + (rect.Dx()-w)/2
	y := rect.Min.Y + (rect.Dy()-height)/2 + ascent
	c.DrawText(image.Pt(x, y), label, df, dim)
}
