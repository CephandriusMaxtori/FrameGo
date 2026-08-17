package moon

import (
	"image"
	"math"
	"testing"
	"time"

	"framego/render"
)

func TestPhaseOfKnownNewMoon(t *testing.T) {
	f, name := phaseOf(newMoonEpoch)
	if f > 0.02 {
		t.Errorf("new moon illumination = %v, want ~0", f)
	}
	if name != "New Moon" {
		t.Errorf("name = %q", name)
	}
}

func TestPhaseOfFullMoon(t *testing.T) {
	full := newMoonEpoch.Add(time.Duration(synodicMonth/2 * 24 * float64(time.Hour)))
	f, name := phaseOf(full)
	if math.Abs(f-1) > 0.02 {
		t.Errorf("full moon illumination = %v, want ~1", f)
	}
	if name != "Full Moon" {
		t.Errorf("name = %q", name)
	}
}

func TestPhaseNames(t *testing.T) {
	cases := map[float64]string{
		0.0:   "New Moon",
		0.10:  "Waxing Crescent",
		0.25:  "First Quarter",
		0.40:  "Waxing Gibbous",
		0.50:  "Full Moon",
		0.60:  "Waning Gibbous",
		0.75:  "Last Quarter",
		0.90:  "Waning Crescent",
		0.999: "New Moon",
	}
	for f, want := range cases {
		if got := phaseName(f); got != want {
			t.Errorf("phaseName(%v) = %q, want %q", f, got, want)
		}
	}
}

func TestDrawProducesPixels(t *testing.T) {
	m := New().(*Moon)
	if err := m.Configure(nil); err != nil {
		t.Fatal(err)
	}
	cv := render.NewCanvas(200, 120)
	now := time.Date(2026, 8, 14, 12, 0, 0, 0, time.UTC)
	if err := m.Draw(cv, image.Rect(0, 0, 200, 120), now); err != nil {
		t.Fatal(err)
	}
	lit := 0
	for y := 0; y < 120; y += 2 {
		for x := 0; x < 200; x += 2 {
			if _, _, _, a := cv.Img.At(x, y).RGBA(); a > 0 {
				lit++
			}
		}
	}
	if lit == 0 {
		t.Error("no pixels rendered")
	}
}
