package engine

import (
	"image"
	"image/color"
	"sync"
	"testing"
	"time"

	"framego/config"
	"framego/render"
)

type testModule struct {
	name     string
	opts     map[string]any
	panicsOn int // render calls before panicking; -1 = never
	calls    int
	mu       sync.Mutex
}

func (m *testModule) Name() string { return m.name }
func (m *testModule) Configure(opts map[string]any) error {
	m.opts = opts
	return nil
}
func (m *testModule) Start(*Bus, *Logger) error { return nil }
func (m *testModule) Stop() error               { return nil }
func (m *testModule) Draw(c *render.Canvas, bounds image.Rectangle, now time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls++
	if m.panicsOn >= 0 && m.calls > m.panicsOn {
		panic("boom")
	}
	c.FillRect(bounds, color.RGBA{R: 0, G: 255, B: 0, A: 255})
	return nil
}

func newTest(name string, panicsOn int) CreateFunc {
	return func(n string) (Module, bool) {
		if n == name {
			return &testModule{name: n, panicsOn: panicsOn}, true
		}
		return nil, false
	}
}

func TestEngineRenderAndReload(t *testing.T) {
	cfg := config.Default()
	cfg.Display.Width, cfg.Display.Height = 200, 120
	cfg.Modules = []config.Module{{Name: "t", Zone: "top-left", Visible: true}}

	e, err := New(cfg, nil, NewLogger(nil), newTest("t", -1))
	if err != nil {
		t.Fatal(err)
	}
	e.Start()
	defer e.Stop()

	img := e.RenderFrame()
	if img.Bounds().Dx() != 200 || img.Bounds().Dy() != 120 {
		t.Errorf("frame size = %v", img.Bounds())
	}
	// The module drew a green rect somewhere.
	if !hasGreen(img) {
		t.Error("no module pixels rendered")
	}
	if st := e.Status(); len(st) != 1 || st[0].State != "active" {
		t.Errorf("status = %+v", st)
	}
}

func TestEnginePanicIsolation(t *testing.T) {
	cfg := config.Default()
	cfg.Display.Width, cfg.Display.Height = 200, 120
	cfg.Modules = []config.Module{
		{Name: "bad", Zone: "top-left", Visible: true},
		{Name: "good", Zone: "top-right", Visible: true},
	}
	create := func(n string) (Module, bool) {
		switch n {
		case "bad":
			return &testModule{name: n, panicsOn: 0}, true
		case "good":
			return &testModule{name: n, panicsOn: -1}, true
		}
		return nil, false
	}
	e, err := New(cfg, nil, NewLogger(nil), create)
	if err != nil {
		t.Fatal(err)
	}
	e.Start()
	defer e.Stop()

	for i := 0; i < 3; i++ {
		_ = e.RenderFrame() // bad module panics each frame but must not crash the engine
	}
	byName := map[string]State{}
	for _, st := range e.Status() {
		byName[st.Name] = StateFor(st.State)
	}
	if byName["bad"] != StateFaulted {
		t.Errorf("bad module state = %v, want faulted", byName["bad"])
	}
	if byName["good"] != StateActive {
		t.Errorf("good module state = %v, want active", byName["good"])
	}
}

func TestEngineReloadDiffsModules(t *testing.T) {
	cfg := config.Default()
	cfg.Display.Width, cfg.Display.Height = 200, 120
	cfg.Modules = []config.Module{{Name: "a", Zone: "top-left", Visible: true}}

	create := func(n string) (Module, bool) {
		switch n {
		case "a", "b":
			return &testModule{name: n, panicsOn: -1}, true
		}
		return nil, false
	}
	e, err := New(cfg, nil, NewLogger(nil), create)
	if err != nil {
		t.Fatal(err)
	}
	e.Start()
	defer e.Stop()

	next := config.Default()
	next.Display.Width, next.Display.Height = 200, 120
	next.Modules = []config.Module{
		{Name: "a", Zone: "bottom-left", Visible: true, Options: map[string]any{"k": "v"}},
		{Name: "b", Zone: "middle-center", Visible: true},
	}
	if err := e.Reload(next); err != nil {
		t.Fatal(err)
	}
	got := map[string]bool{}
	for _, st := range e.Status() {
		got[st.Name] = true
	}
	if !got["a"] || !got["b"] {
		t.Errorf("modules after reload = %v", got)
	}
	if e.Config().Modules[1].Name != "b" {
		t.Errorf("config not updated: %+v", e.Config().Modules)
	}
}

func TestEngineReloadRejectsUnknownModule(t *testing.T) {
	cfg := config.Default()
	cfg.Modules = []config.Module{{Name: "a", Zone: "top-left", Visible: true}}
	e, err := New(cfg, nil, NewLogger(nil), newTest("a", -1))
	if err != nil {
		t.Fatal(err)
	}
	bad := config.Default()
	bad.Modules = []config.Module{{Name: "ghost", Zone: "top-left", Visible: true}}
	if err := e.Reload(bad); err == nil {
		t.Error("expected error reloading unknown module")
	}
}

func hasGreen(img *image.RGBA) bool {
	for y := 0; y < img.Bounds().Dy(); y += 4 {
		for x := 0; x < img.Bounds().Dx(); x += 4 {
			r, g, bl, _ := img.At(x, y).RGBA()
			if g > r && g > bl {
				return true
			}
		}
	}
	return false
}

// StateFor maps a status string back to a State for assertions.
func StateFor(s string) State {
	switch s {
	case "active":
		return StateActive
	case "faulted":
		return StateFaulted
	case "suspended":
		return StateSuspended
	default:
		return StateUninitialized
	}
}
