// Package moon implements the Moon Phase module: an astronomical phase readout
// computed entirely locally (no network) and drawn as a crescent icon.
package moon

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"time"

	"framego/engine"
	"framego/fonts"
	"framego/modules"
	"framego/modules/opt"
	"framego/render"
)

// synodicMonth is the mean lunar cycle length in days.
const synodicMonth = 29.53058867

// newMoonEpoch is a known new moon instant in UTC.
var newMoonEpoch = time.Date(2000, 1, 6, 18, 14, 0, 0, time.UTC)

// Moon renders the current phase as a crescent with a text label.
type Moon struct {
	moonColor   color.RGBA
	shadowColor color.RGBA
	labelColor  color.RGBA
	showPercent bool
	tz          *time.Location
}

// New constructs a moon phase module.
func New() engine.Module { return &Moon{} }

func init() { modules.Register("moon", New) }

// Name identifies the module.
func (m *Moon) Name() string { return "moon" }

// Configure applies module options.
func (m *Moon) Configure(opts map[string]any) error {
	m.moonColor = opt.Color(opts, "moonColor", color.RGBA{R: 240, G: 244, B: 250, A: 255})
	m.shadowColor = opt.Color(opts, "shadowColor", color.RGBA{R: 26, G: 34, B: 48, A: 255})
	m.labelColor = opt.Color(opts, "labelColor", color.RGBA{R: 154, G: 167, B: 184, A: 255})
	m.showPercent = opt.Bool(opts, "showPercent", true)
	tzName := opt.Str(opts, "timezone", "")
	loc := time.Local
	if tzName != "" {
		l, err := time.LoadLocation(tzName)
		if err != nil {
			return fmt.Errorf("moon: bad timezone %q: %w", tzName, err)
		}
		loc = l
	}
	m.tz = loc
	return nil
}

// Start has no background work.
func (m *Moon) Start(_ *engine.Bus, _ *engine.Logger) error { return nil }

// Stop is a no-op.
func (m *Moon) Stop() error { return nil }

// Draw renders the moon crescent and phase label centered in bounds.
func (m *Moon) Draw(cv *render.Canvas, bounds image.Rectangle, now time.Time) error {
	f, name := phaseOf(now.In(m.tz))

	lf := fonts.Face(16, fonts.Regular)
	label := name
	if m.showPercent {
		label = fmt.Sprintf("%s  %d%%", name, int(math.Round(f*100)))
	}
	lw, lh := cv.TextSize(lf, label)
	radius := 30
	iconD := 2 * radius
	spacing := 8

	blockH := iconD + spacing + lh
	blockTop := bounds.Min.Y + (bounds.Dy()-blockH)/2
	center := image.Pt(bounds.Min.X+bounds.Dx()/2, blockTop+radius)
	xIcon := bounds.Min.X + (bounds.Dx()-iconD)/2
	drawCrescent(cv, image.Pt(xIcon+radius, blockTop+radius), radius, f, m.moonColor, m.shadowColor)

	ascent, _, _ := cv.FaceMetrics(lf)
	cv.DrawText(image.Pt(center.X-lw/2, blockTop+iconD+spacing+ascent), label, lf, m.labelColor)
	return nil
}

// phaseOf returns the illuminated fraction in [0,1] and the phase name for now.
func phaseOf(now time.Time) (float64, string) {
	days := now.Sub(newMoonEpoch).Hours() / 24
	age := math.Mod(days, synodicMonth)
	if age < 0 {
		age += synodicMonth
	}
	f := (1 - math.Cos(2*math.Pi*age/synodicMonth)) / 2
	return f, phaseName(age / synodicMonth)
}

// phaseName maps a phase fraction in [0,1) to its conventional name.
func phaseName(f float64) string {
	switch {
	case f < 0.02 || f >= 0.98:
		return "New Moon"
	case f < 0.23:
		return "Waxing Crescent"
	case f < 0.27:
		return "First Quarter"
	case f < 0.48:
		return "Waxing Gibbous"
	case f < 0.52:
		return "Full Moon"
	case f < 0.73:
		return "Waning Gibbous"
	case f < 0.77:
		return "Last Quarter"
	default:
		return "Waning Crescent"
	}
}

// drawCrescent paints a moon disc with the phase's terminator approximated by
// a second circle sliding across the disc. Waxing phases light the right side;
// waning phases light the left side.
func drawCrescent(cv *render.Canvas, center image.Point, radius int, frac float64, lit, shadow color.RGBA) {
	cv.FillCircle(center, radius, lit)
	theta := 2 * math.Pi * frac
	off := 2 * float64(radius)
	if theta <= math.Pi {
		off = -off * (theta / math.Pi)
	} else {
		off = off * ((2*math.Pi - theta) / math.Pi)
	}
	occluder := image.Pt(center.X+int(math.Round(off)), center.Y)
	cv.FillCircle(occluder, radius, shadow)
}

func (m *Moon) Schema() *engine.Schema {
	return &engine.Schema{
		Name:        "moon",
		Description: "Current moon phase with crescent icon",
		Fields: []engine.Field{
			{Key: "moonColor", Label: "Moon Color", Kind: engine.FieldColor, Default: "#f0f4fa"},
			{Key: "shadowColor", Label: "Shadow Color", Kind: engine.FieldColor, Default: "#1a2230"},
			{Key: "labelColor", Label: "Label Color", Kind: engine.FieldColor, Default: "#9aa7b8"},
			{Key: "showPercent", Label: "Show Percentage", Kind: engine.FieldBoolean, Default: "true"},
			{Key: "timezone", Label: "Timezone", Kind: engine.FieldText, Default: "", Placeholder: "America/New_York", Hint: "IANA timezone name (blank = local)"},
		},
	}
}
