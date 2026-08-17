package slideshow

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"framego/render"
)

func writePNG(t *testing.T, path string) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			img.Set(x, y, color.RGBA{R: 90, G: 160, B: 230, A: 255})
		}
	}
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
}

func TestConfigureRequiresDir(t *testing.T) {
	s := New().(*Slideshow)
	if err := s.Configure(nil); err == nil {
		t.Error("expected error when dir missing")
	}
}

func TestConfigureScan(t *testing.T) {
	dir := t.TempDir()
	writePNG(t, filepath.Join(dir, "b.png"))
	writePNG(t, filepath.Join(dir, "a.jpg"))
	os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("x"), 0o644)

	s := New().(*Slideshow)
	if err := s.Configure(map[string]any{"dir": dir}); err != nil {
		t.Fatal(err)
	}
	if len(s.files) != 2 {
		t.Fatalf("files = %v", s.files)
	}
	if filepath.Base(s.files[0]) != "a.jpg" {
		t.Errorf("files not sorted: %v", s.files)
	}
}

func TestDrawRendersImage(t *testing.T) {
	dir := t.TempDir()
	writePNG(t, filepath.Join(dir, "a.png"))

	s := New().(*Slideshow)
	if err := s.Configure(map[string]any{"dir": dir, "interval": 60}); err != nil {
		t.Fatal(err)
	}
	cv := render.NewCanvas(80, 80)
	now := time.Unix(1000, 0)
	if err := s.Draw(cv, image.Rect(0, 0, 80, 80), now); err != nil {
		t.Fatal(err)
	}
	r, g, b, _ := cv.Img.At(40, 40).RGBA()
	if r>>8 < 50 || g>>8 < 100 || b>>8 < 150 {
		t.Errorf("scaled image color = %d,%d,%d", r>>8, g>>8, b>>8)
	}
}

func TestDrawEmptyDir(t *testing.T) {
	dir := t.TempDir()
	s := New().(*Slideshow)
	if err := s.Configure(map[string]any{"dir": dir}); err != nil {
		t.Fatal(err)
	}
	cv := render.NewCanvas(80, 40)
	if err := s.Draw(cv, image.Rect(0, 0, 80, 40), time.Now()); err != nil {
		t.Fatal(err)
	}
}

func TestImageAtSkipsCorrupt(t *testing.T) {
	dir := t.TempDir()
	bad := filepath.Join(dir, "bad.png")
	os.WriteFile(bad, []byte("not an image"), 0o644)
	good := filepath.Join(dir, "good.png")
	writePNG(t, good)

	s := New().(*Slideshow)
	if err := s.Configure(map[string]any{"dir": dir}); err != nil {
		t.Fatal(err)
	}
	if len(s.files) != 2 {
		t.Fatalf("files = %v", s.files)
	}
	img := s.imageAt(0)
	if img == nil {
		t.Fatal("expected to fall back to good image")
	}
}
