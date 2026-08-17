package quote

import (
	"image"
	"os"
	"path/filepath"
	"testing"
	"time"

	"framego/render"
)

func TestConfigureDefaults(t *testing.T) {
	q := New().(*Quote)
	if err := q.Configure(nil); err != nil {
		t.Fatal(err)
	}
	if len(q.quotes) == 0 {
		t.Fatal("expected embedded quotes")
	}
}

func TestLoadQuotesPlainText(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "quotes.txt")
	content := "First quote.\n\nSecond quote.\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	q := New().(*Quote)
	if err := q.Configure(map[string]any{"file": path}); err != nil {
		t.Fatal(err)
	}
	if len(q.quotes) != 2 || q.quotes[0] != "First quote." || q.quotes[1] != "Second quote." {
		t.Errorf("quotes = %v", q.quotes)
	}
}

func TestLoadQuotesJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "quotes.json")
	if err := os.WriteFile(path, []byte(`["a quote", "another", ""]`), 0o644); err != nil {
		t.Fatal(err)
	}
	q := New().(*Quote)
	if err := q.Configure(map[string]any{"file": path}); err != nil {
		t.Fatal(err)
	}
	if len(q.quotes) != 2 || q.quotes[0] != "a quote" {
		t.Errorf("quotes = %v", q.quotes)
	}
}

func TestConfigureMissingFile(t *testing.T) {
	q := New().(*Quote)
	if err := q.Configure(map[string]any{"file": filepath.Join(t.TempDir(), "nope.txt")}); err == nil {
		t.Error("expected error for missing file")
	}
}

func TestDrawRotatesByInterval(t *testing.T) {
	q := New().(*Quote)
	if err := q.Configure(map[string]any{"quotes": nil, "interval": 60}); err != nil {
		t.Fatal(err)
	}
	q.quotes = []string{"zero", "one"}
	base := time.Unix(1000, 0)
	cv := render.NewCanvas(300, 100)
	bounds := image.Rect(0, 0, 300, 100)
	if err := q.Draw(cv, bounds, base); err != nil {
		t.Fatal(err)
	}
	if idx := int(base.Unix()/60) % 2; idx != 0 {
		t.Errorf("index = %d", idx)
	}
}
