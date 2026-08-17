package layout

import (
	"image"

	"framego/config"
)

// Solver maps display dimensions to zone bounding boxes.
type Solver struct {
	cfg config.Display
}

// NewSolver creates a zone solver for the given display configuration.
func NewSolver(cfg config.Display) *Solver {
	return &Solver{cfg: cfg}
}

// Resolve returns the bounding rectangle of every zone. Adjacent zones never
// overlap: the usable area is inset by the display margin, split into five
// equal-height bands separated by gaps, and each band is split into three
// equal-width columns separated by gaps. Bar zones span the full usable width.
func (s *Solver) Resolve() map[ZoneID]image.Rectangle {
	rects := make(map[ZoneID]image.Rectangle, len(AllZones))
	m := s.cfg.Margin
	g := s.cfg.Gap
	usable := image.Rect(m, m, s.cfg.Width-m, s.cfg.Height-m)

	nBands := len(bands)
	nGaps := nBands - 1
	bandH := (usable.Dy() - nGaps*g) / nBands
	colW := (usable.Dx() - 2*g) / 3

	for i, band := range bands {
		by := usable.Min.Y + i*(bandH+g)
		bottom := by + bandH
		for _, zone := range band.zones {
			var r image.Rectangle
			switch zone {
			case ZoneTopBar, ZoneBottomBar:
				r = image.Rect(usable.Min.X, by, usable.Max.X, bottom)
			default:
				col := s.column(zone)
				cx := usable.Min.X + col*(colW+g)
				r = image.Rect(cx, by, cx+colW, bottom)
			}
			rects[zone] = r
		}
	}
	return rects
}

// ZoneRect returns the resolved bounds of a single zone.
func (s *Solver) ZoneRect(id ZoneID) (image.Rectangle, bool) {
	r, ok := s.Resolve()[id]
	return r, ok
}

// column maps a zone to its horizontal column index (0 left, 1 center, 2 right).
func (s *Solver) column(id ZoneID) int {
	switch id {
	case ZoneTopLeft, ZoneUpperLeft, ZoneMiddleLeft, ZoneLowerLeft, ZoneBottomLeft:
		return 0
	case ZoneTopCenter, ZoneUpperCenter, ZoneMiddleCenter, ZoneLowerCenter, ZoneBottomCenter:
		return 1
	default:
		return 2
	}
}
