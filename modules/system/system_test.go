package system

import (
	"image"
	"path/filepath"
	"testing"
	"time"

	"framego/render"
)

func TestConfigureDefaults(t *testing.T) {
	s := New().(*System)
	if err := s.Configure(nil); err != nil {
		t.Fatal(err)
	}
	if !s.showCPU || !s.showMem || !s.showDisk {
		t.Error("defaults should show all rows")
	}
	if s.interval <= 0 {
		t.Error("interval must be positive")
	}
}

func TestRefreshSamples(t *testing.T) {
	s := New().(*System)
	if err := s.Configure(map[string]any{"diskPath": filepath.ToSlash(t.TempDir())}); err != nil {
		t.Fatal(err)
	}
	if err := s.refresh(); err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if s.memTotal == 0 {
		t.Error("memTotal is zero")
	}
}

func TestDrawBeforeRefresh(t *testing.T) {
	s := New().(*System)
	if err := s.Configure(map[string]any{"diskPath": filepath.ToSlash(t.TempDir())}); err != nil {
		t.Fatal(err)
	}
	cv := renderCanvas()
	if err := s.Draw(cv, image.Rect(0, 0, 200, 100), time.Now()); err != nil {
		t.Fatal(err)
	}
}

func TestDrawAfterRefresh(t *testing.T) {
	s := New().(*System)
	if err := s.Configure(map[string]any{"diskPath": filepath.ToSlash(t.TempDir())}); err != nil {
		t.Fatal(err)
	}
	if err := s.refresh(); err != nil {
		t.Fatalf("refresh: %v", err)
	}
	cv := renderCanvas()
	if err := s.Draw(cv, image.Rect(0, 0, 260, 120), time.Now()); err != nil {
		t.Fatal(err)
	}
	lit := 0
	for y := 0; y < 120; y++ {
		for x := 0; x < 260; x++ {
			if _, _, _, a := cv.Img.At(x, y).RGBA(); a > 0 {
				lit++
			}
		}
	}
	if lit == 0 {
		t.Error("no pixels rendered")
	}
}

func renderCanvas() *render.Canvas {
	return render.NewCanvas(300, 160)
}
