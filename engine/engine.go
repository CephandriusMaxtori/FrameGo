package engine

import (
	"fmt"
	"image"
	"image/color"
	"sync"
	"time"

	"framego/config"
	"framego/layout"
	"framego/render"
)

// Engine wires the bus, supervisor, layout solver, and canvas into a single
// display loop, and exposes hot reload plus on-demand frame rendering.
type Engine struct {
	cfg     *config.Config
	bus     *Bus
	sup     *Supervisor
	solver  *layout.Solver
	canvas  *render.Canvas
	backend render.Backend
	log     *Logger
	create  CreateFunc

	mu   sync.Mutex
	stop chan struct{}
	once sync.Once
	wg   sync.WaitGroup
}

// New constructs an engine from the given configuration. The create function
// instantiates modules by name; backend may be nil (headless rendering).
func New(cfg *config.Config, backend render.Backend, log *Logger, create CreateFunc) (*Engine, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	e := &Engine{
		cfg:     cfg,
		bus:     NewBus(),
		log:     log,
		backend: backend,
		create:  create,
		solver:  layout.NewSolver(cfg.Display),
		canvas:  render.NewCanvas(cfg.Display.Width, cfg.Display.Height),
		stop:    make(chan struct{}),
	}
	e.sup = NewSupervisor(e.bus, log)
	if err := e.buildModules(); err != nil {
		return nil, err
	}
	return e, nil
}

// buildModules registers every configured module with the supervisor.
func (e *Engine) buildModules() error {
	for _, m := range e.cfg.Modules {
		if e.create == nil {
			return fmt.Errorf("no module factory registered")
		}
		mod, ok := e.create(m.Name)
		if !ok {
			return fmt.Errorf("unknown module %q", m.Name)
		}
		if err := e.sup.Add(mod, m.Options); err != nil {
			return err
		}
	}
	return nil
}

// Start activates modules and launches the frame loop.
func (e *Engine) Start() {
	e.sup.StartAll()
	e.wg.Add(1)
	go e.frameLoop()
}

// Stop halts the frame loop and suspends all modules.
func (e *Engine) Stop() {
	e.once.Do(func() { close(e.stop) })
	e.wg.Wait()
	e.sup.StopAll()
}

// Bus exposes the event bus for external subscribers (e.g. web handlers).
func (e *Engine) Bus() *Bus { return e.bus }

// Logger returns the engine logger.
func (e *Engine) Logger() *Logger { return e.log }

// Config returns a defensive copy of the active configuration.
func (e *Engine) Config() *config.Config {
	e.mu.Lock()
	defer e.mu.Unlock()
	cp := *e.cfg
	cp.Modules = make([]config.Module, len(e.cfg.Modules))
	copy(cp.Modules, e.cfg.Modules)
	return &cp
}

// Status returns the supervisor's module status snapshot.
func (e *Engine) Status() []ModuleStatus {
	return e.sup.Status()
}

// Reload applies a new configuration at runtime: it diffs the module set,
// removes dropped modules, reconfigures existing ones, starts new ones, and
// recomputes layout geometry. The bus survives so unchanged modules keep state.
func (e *Engine) Reload(cfg *config.Config) error {
	if err := cfg.Validate(); err != nil {
		return err
	}
	e.mu.Lock()
	defer e.mu.Unlock()

	newSet := make(map[string]bool, len(cfg.Modules))
	for _, m := range cfg.Modules {
		newSet[m.Name] = true
	}
	for _, name := range e.sup.Names() {
		if !newSet[name] {
			_ = e.sup.Remove(name)
		}
	}
	for _, m := range cfg.Modules {
		switch {
		case e.sup.Has(m.Name):
			if err := e.sup.Reconfigure(m.Name, m.Options); err != nil {
				return err
			}
		case e.create != nil:
			mod, ok := e.create(m.Name)
			if !ok {
				return fmt.Errorf("unknown module %q", m.Name)
			}
			if err := e.sup.Add(mod, m.Options); err != nil {
				return err
			}
			if err := e.sup.Start(m.Name); err != nil {
				return err
			}
		}
	}

	e.cfg = cfg
	e.solver = layout.NewSolver(cfg.Display)
	e.canvas = render.NewCanvas(cfg.Display.Width, cfg.Display.Height)
	return nil
}

// RenderFrame synchronously renders the current layout and returns the frame.
// It is safe to call concurrently with the frame loop.
func (e *Engine) RenderFrame() *image.RGBA {
	e.mu.Lock()
	defer e.mu.Unlock()
	bg := render.ParseHexColor(e.cfg.Display.Background, color.RGBA{R: 11, G: 15, B: 20, A: 255})
	c := e.canvas
	c.Fill(bg)
	e.sup.DrawAll(c, e.moduleRects(), time.Now())
	return c.Img
}

// moduleRects computes the draw bounds of every visible module. Multiple
// modules in one zone are vertically stacked and clipped to the zone.
// Callers must hold e.mu.
func (e *Engine) moduleRects() map[string]image.Rectangle {
	zones := e.solver.Resolve()
	byZone := make(map[string][]string)
	for _, m := range e.cfg.Modules {
		if !m.Visible {
			continue
		}
		byZone[m.Zone] = append(byZone[m.Zone], m.Name)
	}
	rects := make(map[string]image.Rectangle)
	for zone, names := range byZone {
		bounds, ok := zones[layout.ZoneID(zone)]
		if !ok || len(names) == 0 {
			continue
		}
		slots := layout.Stack(bounds, len(names), e.cfg.Display.Gap)
		for i, name := range names {
			rects[name] = slots[i]
		}
	}
	return rects
}

// frameLoop ticks at the configured frame rate, broadcasts the clock tick, and
// presents rendered frames to the backend.
func (e *Engine) frameLoop() {
	defer e.wg.Done()
	for {
		select {
		case <-time.After(e.frameInterval()):
			e.bus.Publish(TopicClockTick, nil)
			img := e.RenderFrame()
			if e.backend != nil {
				if err := e.backend.Present(img); err != nil {
					e.log.Errorf("present frame: %v", err)
				}
			}
		case <-e.stop:
			return
		}
	}
}

func (e *Engine) frameInterval() time.Duration {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.cfg.Display.FPS <= 0 {
		return time.Second
	}
	return time.Second / time.Duration(e.cfg.Display.FPS)
}
