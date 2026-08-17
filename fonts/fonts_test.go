package fonts

import (
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/image/font/gofont/goregular"
)

func TestFace(t *testing.T) {
	f := Face(14, Regular)
	m := f.Metrics()
	if m.Height.Ceil() <= 0 {
		t.Error("empty metrics")
	}
}

func TestFaceCache(t *testing.T) {
	a := Face(20, Bold)
	b := Face(20, Bold)
	if a != b {
		t.Error("faces not cached")
	}
}

func TestLoadFile(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "font.ttf")
	if err := os.WriteFile(p, goregular.TTF, 0o644); err != nil {
		t.Fatal(err)
	}
	f, err := LoadFile(p, 16)
	if err != nil {
		t.Fatal(err)
	}
	if f.Metrics().Height.Ceil() <= 0 {
		t.Error("loaded face has no metrics")
	}
}

func TestLoadFileMissing(t *testing.T) {
	if _, err := LoadFile(filepath.Join(t.TempDir(), "nope.ttf"), 16); err == nil {
		t.Error("expected error for missing font file")
	}
}

func TestLoadFileInvalidData(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "bad.ttf")
	if err := os.WriteFile(p, []byte("not a font"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadFile(p, 16); err == nil {
		t.Error("expected error for invalid font data")
	}
}
