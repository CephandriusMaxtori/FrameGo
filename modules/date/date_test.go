package date

import (
	"image"
	"testing"
	"time"

	"framego/render"
)

func TestConfigureDefaults(t *testing.T) {
	d := New().(*Date)
	if err := d.Configure(nil); err != nil {
		t.Fatal(err)
	}
	if d.format != "Jan 2, 2006" || d.weekday != "Monday" {
		t.Errorf("formats = %q / %q", d.format, d.weekday)
	}
	if d.tz != time.Local {
		t.Error("default tz should be local")
	}
}

func TestConfigureBadTimezone(t *testing.T) {
	d := New().(*Date)
	if err := d.Configure(map[string]any{"timezone": "Mars/Olympus"}); err == nil {
		t.Error("expected error for invalid timezone")
	}
}

func TestDrawProducesPixels(t *testing.T) {
	d := New().(*Date)
	if err := d.Configure(map[string]any{"format": "Jan 2, 2006", "weekdayFormat": "Monday"}); err != nil {
		t.Fatal(err)
	}
	cv := render.NewCanvas(300, 120)
	now := time.Date(2026, 8, 14, 12, 0, 0, 0, time.UTC)
	if err := d.Draw(cv, image.Rect(0, 0, 300, 120), now); err != nil {
		t.Fatal(err)
	}
	lit := 0
	for y := 0; y < 120; y += 2 {
		for x := 0; x < 300; x += 2 {
			if _, _, _, a := cv.Img.At(x, y).RGBA(); a > 0 {
				lit++
			}
		}
	}
	if lit == 0 {
		t.Error("no text pixels rendered")
	}
}
