// Package quote implements the Quote module: a rotating quotation that can be
// sourced from a local file or the embedded default list. No network access.
package quote

import (
	"bufio"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"os"
	"strings"
	"time"

	"framego/engine"
	"framego/fonts"
	"framego/modules"
	"framego/modules/opt"
	"framego/render"
)

// Quote renders a single quotation, rotating on a configurable cadence.
type Quote struct {
	quotes    []string
	interval  time.Duration
	textColor color.RGBA
	prefix    string
}

// New constructs a quote module.
func New() engine.Module { return &Quote{} }

func init() { modules.Register("quote", New) }

// Name identifies the module.
func (q *Quote) Name() string { return "quote" }

// Configure applies module options and (re)loads the quotes source.
func (q *Quote) Configure(opts map[string]any) error {
	q.interval = opt.Duration(opts, "interval", 24*60*60)
	q.textColor = opt.Color(opts, "textColor", color.RGBA{R: 154, G: 167, B: 184, A: 255})
	q.prefix = opt.Str(opts, "prefix", "")

	file := opt.Str(opts, "file", "")
	if file == "" {
		q.quotes = defaultQuotes()
		return nil
	}
	quotes, err := loadQuotes(file)
	if err != nil {
		return fmt.Errorf("quote: %w", err)
	}
	if len(quotes) == 0 {
		return fmt.Errorf("quote: no quotes found in %q", file)
	}
	q.quotes = quotes
	return nil
}

// Start has no background work; the index is derived from the clock.
func (q *Quote) Start(_ *engine.Bus, _ *engine.Logger) error { return nil }

// Stop is a no-op.
func (q *Quote) Stop() error { return nil }

// Draw renders the current quote, word-wrapped and centered in bounds.
func (q *Quote) Draw(cv *render.Canvas, bounds image.Rectangle, now time.Time) error {
	if len(q.quotes) == 0 {
		q.quotes = defaultQuotes()
	}
	idx := 0
	if q.interval > 0 && len(q.quotes) > 1 {
		idx = int(now.Unix()/int64(q.interval.Seconds())) % len(q.quotes)
	}
	if idx < 0 {
		idx = -idx % len(q.quotes)
	}
	text := q.quotes[idx]
	if q.prefix != "" {
		text = q.prefix + " " + text
	}

	f := fonts.Face(20, fonts.Regular)
	lines := render.WrapText(f, text, bounds.Dx())
	_, _, lineHeight := cv.FaceMetrics(f)
	blockH := len(lines) * lineHeight
	ascent := cv.Ascent(f)
	y := bounds.Min.Y + (bounds.Dy()-blockH)/2
	for _, line := range lines {
		w, _ := cv.TextSize(f, line)
		cv.DrawText(image.Pt(bounds.Min.X+(bounds.Dx()-w)/2, y+ascent), line, f, q.textColor)
		y += lineHeight
	}
	return nil
}

// loadQuotes parses a quotes file: a JSON array of strings, or plain text with
// one quote per line.
func loadQuotes(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %q: %w", path, err)
	}
	trimmed := strings.TrimSpace(string(data))
	if strings.HasPrefix(trimmed, "[") {
		var arr []string
		if err := json.Unmarshal([]byte(trimmed), &arr); err != nil {
			return nil, fmt.Errorf("parse %q as JSON array: %w", path, err)
		}
		var out []string
		for _, s := range arr {
			if s = strings.TrimSpace(s); s != "" {
				out = append(out, s)
			}
		}
		return out, nil
	}
	var out []string
	sc := bufio.NewScanner(strings.NewReader(trimmed))
	for sc.Scan() {
		if s := strings.TrimSpace(sc.Text()); s != "" {
			out = append(out, s)
		}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (q *Quote) Schema() *engine.Schema {
	return &engine.Schema{
		Name:        "quote",
		Description: "Rotating quotation display",
		Fields: []engine.Field{
			{Key: "interval", Label: "Rotate Interval (s)", Kind: engine.FieldDuration, Default: "86400", Hint: "Seconds between quote rotations"},
			{Key: "textColor", Label: "Text Color", Kind: engine.FieldColor, Default: "#9aa7b8"},
			{Key: "prefix", Label: "Prefix", Kind: engine.FieldText, Default: "", Placeholder: "e.g. Quote of the day:"},
			{Key: "file", Label: "Quotes File", Kind: engine.FieldFile, Default: "", Placeholder: "/path/to/quotes.txt", Hint: "JSON array or one-per-line text file (blank = built-in quotes)"},
		},
	}
}

// defaultQuotes returns the embedded fallback quotation list so the module
// works with zero configuration.
func defaultQuotes() []string {
	return []string{
		"The only way to do great work is to love what you do. — Steve Jobs",
		"In the middle of difficulty lies opportunity. — Albert Einstein",
		"Simplicity is the ultimate sophistication. — Leonardo da Vinci",
		"The best time to plant a tree was 20 years ago. The second best time is now. — Chinese Proverb",
		"Whether you think you can or you think you can't, you're right. — Henry Ford",
		"It does not matter how slowly you go as long as you do not stop. — Confucius",
		"Everything should be made as simple as possible, but not simpler. — Albert Einstein",
		"Quality is not an act, it is a habit. — Aristotle",
	}
}
