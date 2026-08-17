// Package clock implements the Clock & Date module.
package clock

import (
	"fmt"
	"image"
	"image/color"
	"time"

	"framego/engine"
	"framego/fonts"
	"framego/modules"
	"framego/render"
)

// Clock renders the current time prominently with the date below it.
type Clock struct {
	format     string
	dateFormat string
	tz         *time.Location
	timeColor  color.RGBA
	dateColor  color.RGBA
}

// New constructs a clock module.
func New() engine.Module {
	return &Clock{}
}

func init() {
	modules.Register("clock", New)
}

// Name identifies the module.
func (c *Clock) Name() string { return "clock" }

// Configure applies module options.
func (c *Clock) Configure(opts map[string]any) error {
	c.format = str(opts, "format", "15:04")
	c.dateFormat = str(opts, "dateFormat", "Mon, Jan 2")
	tzName := str(opts, "timezone", "")
	loc := time.Local
	if tzName != "" {
		l, err := time.LoadLocation(tzName)
		if err != nil {
			return fmt.Errorf("clock: bad timezone %q: %w", tzName, err)
		}
		loc = l
	}
	c.tz = loc
	c.timeColor = render.ParseHexColor(str(opts, "timeColor", "#f5f7fa"), white)
	c.dateColor = render.ParseHexColor(str(opts, "dateColor", "#9aa7b8"), white)
	return nil
}

// Start has no background work; the clock renders on demand.
func (c *Clock) Start(_ *engine.Bus, _ *engine.Logger) error { return nil }

// Stop is a no-op.
func (c *Clock) Stop() error { return nil }

// Draw renders time and date centered in bounds.
func (c *Clock) Draw(cv *render.Canvas, bounds image.Rectangle, now time.Time) error {
	t := now.In(c.tz)
	timeStr := t.Format(c.format)
	dateStr := t.Format(c.dateFormat)

	tf := fonts.Scaled(bounds, 72, fonts.Medium)
	df := fonts.Scaled(bounds, 26, fonts.Regular)
	tw, th := cv.TextSize(tf, timeStr)
	dw, dh := cv.TextSize(df, dateStr)
	spacing := 10

	blockH := th + spacing + dh
	xTime := bounds.Min.X + (bounds.Dx()-tw)/2
	xDate := bounds.Min.X + (bounds.Dx()-dw)/2

	ascent, _, height := cv.FaceMetrics(tf)
	blockTop := bounds.Min.Y + (bounds.Dy()-blockH)/2
	baselineTime := blockTop + (height-th)/2 + ascent
	cv.DrawText(image.Pt(xTime, baselineTime), timeStr, tf, c.timeColor)

	ascentD, _, heightD := cv.FaceMetrics(df)
	baselineDate := blockTop + th + spacing + (heightD-dh)/2 + ascentD
	cv.DrawText(image.Pt(xDate, baselineDate), dateStr, df, c.dateColor)
	return nil
}

func (c *Clock) Schema() *engine.Schema {
	return &engine.Schema{
		Name:        "clock",
		Description: "Current time and date display",
		Fields: []engine.Field{
			{Key: "format", Label: "Time Format", Kind: engine.FieldText, Default: "15:04", Placeholder: "Go time format", Hint: "Go reference time format string"},
			{Key: "dateFormat", Label: "Date Format", Kind: engine.FieldText, Default: "Mon, Jan 2", Placeholder: "Go date format", Hint: "Go reference time format string"},
			{Key: "timezone", Label: "Timezone", Kind: engine.FieldText, Default: "", Placeholder: "America/New_York", Hint: "IANA timezone name (blank = local)"},
			{Key: "timeColor", Label: "Time Color", Kind: engine.FieldColor, Default: "#f5f7fa"},
			{Key: "dateColor", Label: "Date Color", Kind: engine.FieldColor, Default: "#9aa7b8"},
		},
	}
}

var white = color.RGBA{R: 245, G: 247, B: 250, A: 255}

func str(opts map[string]any, key, def string) string {
	if v, ok := opts[key].(string); ok && v != "" {
		return v
	}
	return def
}
