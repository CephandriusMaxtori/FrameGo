package clock

import (
	"image"
	"testing"
	"time"

	"framego/render"
)

func TestConfigureDefaults(t *testing.T) {
	c := New().(*Clock)
	if err := c.Configure(nil); err != nil {
		t.Fatal(err)
	}
	if c.format != "15:04" || c.dateFormat != "Mon, Jan 2" {
		t.Errorf("formats = %q / %q", c.format, c.dateFormat)
	}
}

func TestConfigureOptions(t *testing.T) {
	c := New().(*Clock)
	err := c.Configure(map[string]any{
		"format":     "03:04 PM",
		"dateFormat": "Jan _2",
		"timezone":   "UTC",
	})
	if err != nil {
		t.Fatal(err)
	}
	if c.format != "03:04 PM" {
		t.Errorf("format = %q", c.format)
	}
}

func TestConfigureBadTimezone(t *testing.T) {
	c := New().(*Clock)
	if err := c.Configure(map[string]any{"timezone": "Mars/Olympus"}); err == nil {
		t.Error("expected error for invalid timezone")
	}
}

func TestDrawProducesTextPixels(t *testing.T) {
	c := New().(*Clock)
	if err := c.Configure(map[string]any{"format": "15:04", "dateFormat": "Mon, Jan 2"}); err != nil {
		t.Fatal(err)
	}
	cv := render.NewCanvas(300, 160)
	bounds := image.Rect(0, 0, 300, 160)
	now := time.Date(2026, 8, 14, 17, 30, 0, 0, time.UTC)
	if err := c.Draw(cv, bounds, now); err != nil {
		t.Fatal(err)
	}
	lit := 0
	for y := 0; y < 160; y += 2 {
		for x := 0; x < 300; x += 2 {
			r, g, b, _ := cv.Img.At(x, y).RGBA()
			if r>>8 > 80 && g>>8 > 80 && b>>8 > 80 {
				lit++
			}
		}
	}
	if lit == 0 {
		t.Error("no text pixels rendered")
	}
}
