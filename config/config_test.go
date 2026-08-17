package config

import (
	"os"
	"path/filepath"
	"testing"
)

const sampleJSON = `{
  "display": {"width": 1280, "height": 720, "margin": 20, "gap": 10, "fps": 2},
  "modules": [
    {"name": "clock", "zone": "middle-center", "visible": true}
  ]
}`

func TestLoadJSON(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "cfg.json")
	if err := os.WriteFile(p, []byte(sampleJSON), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Display.Width != 1280 || cfg.Display.Height != 720 {
		t.Errorf("display = %+v", cfg.Display)
	}
	if cfg.Display.FPS != 2 {
		t.Errorf("fps = %d", cfg.Display.FPS)
	}
	if cfg.Modules[0].Zone != "middle-center" {
		t.Errorf("zone = %q", cfg.Modules[0].Zone)
	}
}

func TestLoadYAML(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "cfg.yaml")
	yaml := "display:\n  width: 800\n  height: 480\nmodules:\n  - name: clock\n    zone: top-left\n"
	if err := os.WriteFile(p, []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Modules[0].Name != "clock" || cfg.Modules[0].Zone != "top-left" {
		t.Errorf("module = %+v", cfg.Modules[0])
	}
}

func TestDefaultsApplied(t *testing.T) {
	cfg, err := Parse([]byte(`{"modules":[{"name":"clock","zone":"top-center"}]}`), ".json")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Display.Width == 0 || cfg.Display.FPS == 0 || cfg.Display.Background == "" {
		t.Errorf("defaults not applied: %+v", cfg.Display)
	}
}

func TestValidate(t *testing.T) {
	cfg := Default()
	cfg.Display.Width = 0
	cfg.Modules = []Module{{Name: "clock", Zone: "middle-center"}}
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for zero width")
	}

	cfg.Display.Width = 800
	cfg.Modules = []Module{{Name: "", Zone: "top-left"}}
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for empty module name")
	}

	cfg.Modules = []Module{{Name: "a", Zone: ""}}
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for empty zone")
	}

	cfg.Modules = nil
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for no modules")
	}
}

func TestSaveRoundTrip(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "cfg.json")
	cfg := Default()
	cfg.Display.Width = 1024
	cfg.Modules = []Module{{Name: "clock", Zone: "top-right", Visible: true}}
	if err := cfg.Save(p); err != nil {
		t.Fatal(err)
	}
	reloaded, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.Display.Width != 1024 {
		t.Errorf("round trip width = %d", reloaded.Display.Width)
	}
}
