// Package date implements the standalone Date module: a large, prominent
// day/date readout that can be placed independently of the Clock module.
package date

import (
	"fmt"
	"image"
	"image/color"
	"time"

	"framego/engine"
	"framego/fonts"
	"framego/modules"
	"framego/modules/opt"
	"framego/render"
)

// Date renders the current day and date centered in its zone.
type Date struct {
	format    string
	weekday   string
	tz        *time.Location
	dayColor  color.RGBA
	dateColor color.RGBA
}

// New constructs a date module.
func New() engine.Module { return &Date{} }

func init() { modules.Register("date", New) }

// Name identifies the module.
func (d *Date) Name() string { return "date" }

// Configure applies module options.
func (d *Date) Configure(opts map[string]any) error {
	d.format = opt.Str(opts, "format", "Jan 2, 2006")
	d.weekday = opt.Str(opts, "weekdayFormat", "Monday")
	tzName := opt.Str(opts, "timezone", "")
	loc := time.Local
	if tzName != "" {
		l, err := time.LoadLocation(tzName)
		if err != nil {
			return fmt.Errorf("date: bad timezone %q: %w", tzName, err)
		}
		loc = l
	}
	d.tz = loc
	d.dayColor = opt.Color(opts, "dayColor", color.RGBA{R: 245, G: 247, B: 250, A: 255})
	d.dateColor = opt.Color(opts, "dateColor", color.RGBA{R: 154, G: 167, B: 184, A: 255})
	return nil
}

// Start has no background work; the date renders on demand.
func (d *Date) Start(_ *engine.Bus, _ *engine.Logger) error { return nil }

// Stop is a no-op.
func (d *Date) Stop() error { return nil }

// Draw renders the weekday above the full date, centered in bounds.
func (d *Date) Draw(cv *render.Canvas, bounds image.Rectangle, now time.Time) error {
	t := now.In(d.tz)

	df := fonts.Face(34, fonts.Medium)
	wf := fonts.Face(20, fonts.Regular)
	ds := t.Format(d.format)
	ws := t.Format(d.weekday)

	dw, dh := cv.TextSize(df, ds)
	ww, wh := cv.TextSize(wf, ws)
	spacing := 8
	blockH := wh + spacing + dh

	ascentD, _, heightD := cv.FaceMetrics(df)
	ascentW, _, heightW := cv.FaceMetrics(wf)

	blockTop := bounds.Min.Y + (bounds.Dy()-blockH)/2
	baseWeekday := blockTop + (heightW-wh)/2 + ascentW
	baseDate := blockTop + wh + spacing + (heightD-dh)/2 + ascentD

	cv.DrawText(image.Pt(bounds.Min.X+(bounds.Dx()-ww)/2, baseWeekday), ws, wf, d.dayColor)
	cv.DrawText(image.Pt(bounds.Min.X+(bounds.Dx()-dw)/2, baseDate), ds, df, d.dateColor)
	return nil
}

func (d *Date) Schema() *engine.Schema {
	return &engine.Schema{
		Name:        "date",
		Description: "Standalone day and date readout",
		Fields: []engine.Field{
			{Key: "format", Label: "Date Format", Kind: engine.FieldText, Default: "Jan 2, 2006", Placeholder: "Go date format"},
			{Key: "weekdayFormat", Label: "Weekday Format", Kind: engine.FieldText, Default: "Monday", Placeholder: "Go weekday format"},
			{Key: "timezone", Label: "Timezone", Kind: engine.FieldText, Default: "", Placeholder: "America/New_York", Hint: "IANA timezone name (blank = local)"},
			{Key: "dayColor", Label: "Weekday Color", Kind: engine.FieldColor, Default: "#f5f7fa"},
			{Key: "dateColor", Label: "Date Color", Kind: engine.FieldColor, Default: "#9aa7b8"},
		},
	}
}
