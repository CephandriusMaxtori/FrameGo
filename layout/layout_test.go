package layout

import (
	"image"
	"testing"

	"framego/config"
)

func TestZonesResolveNoOverlap(t *testing.T) {
	s := NewSolver(config.Display{Width: 800, Height: 480, Margin: 16, Gap: 8, FPS: 1})
	rects := s.Resolve()
	if len(rects) != len(AllZones) {
		t.Fatalf("resolved %d zones, want %d", len(rects), len(AllZones))
	}
	for _, z := range AllZones {
		r, ok := rects[z]
		if !ok {
			t.Fatalf("zone %s missing", z)
		}
		if r.Dx() <= 0 || r.Dy() <= 0 {
			t.Errorf("zone %s has non-positive bounds %v", z, r)
		}
		if r.Min.X < 0 || r.Min.Y < 0 || r.Max.X > 800 || r.Max.Y > 480 {
			t.Errorf("zone %s out of bounds %v", z, r)
		}
	}
	// Column zones within the same band must not overlap horizontally.
	for i, band := range bands {
		cols := []ZoneID{}
		for _, z := range band.zones {
			if z != ZoneTopBar && z != ZoneBottomBar {
				cols = append(cols, z)
			}
		}
		for a := 0; a < len(cols); a++ {
			for b := a + 1; b < len(cols); b++ {
				if rects[cols[a]].Overlaps(rects[cols[b]]) {
					t.Errorf("band %d: zones %s and %s overlap", i, cols[a], cols[b])
				}
			}
		}
	}
}

func TestMarginRespected(t *testing.T) {
	s := NewSolver(config.Display{Width: 800, Height: 480, Margin: 32, Gap: 0, FPS: 1})
	rects := s.Resolve()
	// Leftmost column zone must start at exactly the margin.
	r := rects[ZoneTopLeft]
	if r.Min.X != 32 {
		t.Errorf("top-left min.X = %d, want 32", r.Min.X)
	}
	// Top band must start at the margin.
	if r.Min.Y != 32 {
		t.Errorf("top band min.Y = %d, want 32", r.Min.Y)
	}
}

func TestBarSpansFullWidth(t *testing.T) {
	s := NewSolver(config.Display{Width: 800, Height: 480, Margin: 16, Gap: 8, FPS: 1})
	rects := s.Resolve()
	bar := rects[ZoneTopBar]
	if bar.Min.X != 16 || bar.Max.X != 800-16 {
		t.Errorf("top-bar width = [%d,%d], want [16,784]", bar.Min.X, bar.Max.X)
	}
}

func TestStack(t *testing.T) {
	bounds := image.Rect(0, 0, 100, 90)
	slots := Stack(bounds, 3, 10)
	if len(slots) != 3 {
		t.Fatalf("got %d slots", len(slots))
	}
	// (90 - 2*10) / 3 = 23px per slot, 10px gaps between, all within bounds.
	for i, s := range slots {
		if s.Dy() != 23 {
			t.Errorf("slot %d height = %d, want 23", i, s.Dy())
		}
	}
	if slots[1].Min.Y != slots[0].Max.Y+10 || slots[2].Min.Y != slots[1].Max.Y+10 {
		t.Errorf("slots not gapped: %v", slots)
	}
	if slots[2].Max.Y > bounds.Max.Y {
		t.Errorf("stack overflows bounds: %v", slots[2])
	}
}

func TestStackSingle(t *testing.T) {
	bounds := image.Rect(0, 0, 100, 90)
	slots := Stack(bounds, 1, 10)
	if len(slots) != 1 || slots[0] != bounds {
		t.Errorf("single slot = %v", slots)
	}
}
