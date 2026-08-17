// Package calendar implements the Calendar module: upcoming events pulled from
// a public ICS/Google-calendar feed and refreshed on a background ticker.
package calendar

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"io"
	"net/http"
	"sort"
	"sync"
	"time"

	ical "github.com/arran4/golang-ical"
	"golang.org/x/image/font"

	"framego/engine"
	"framego/fonts"
	"framego/modules"
	"framego/modules/opt"
	"framego/render"
)

// Calendar renders the next few upcoming events from an ICS feed.
type Calendar struct {
	url       string
	username  string
	password  string
	days      int
	maxEvents int
	interval  time.Duration
	client    *http.Client

	done chan struct{}
	wg   sync.WaitGroup

	mu      sync.Mutex
	events  []event
	fetched bool
	lastErr error

	dateColor color.RGBA
	evtColor  color.RGBA
}
type event struct {
	summary string
	start   time.Time
	allDay  bool
}

// New constructs a calendar module.
func New() engine.Module { return &Calendar{} }

func init() { modules.Register("calendar", New) }

// Name identifies the module.
func (c *Calendar) Name() string { return "calendar" }

// Configure applies module options.
func (c *Calendar) Configure(opts map[string]any) error {
	c.url = opt.Str(opts, "url", "")
	if c.url == "" {
		return fmt.Errorf("calendar: option \"url\" is required")
	}
	c.username = opt.Str(opts, "username", "")
	c.password = opt.Str(opts, "password", "")
	c.days = opt.Int(opts, "days", 3)
	if c.days < 1 {
		c.days = 1
	}
	c.maxEvents = opt.Int(opts, "maxEvents", 5)
	if c.maxEvents < 1 {
		c.maxEvents = 5
	}
	c.interval = opt.Duration(opts, "update", 3600)
	if c.interval <= 0 {
		c.interval = time.Hour
	}
	if c.client == nil {
		c.client = &http.Client{Timeout: 20 * time.Second}
	}

	c.dateColor = opt.Color(opts, "dateColor", color.RGBA{R: 154, G: 167, B: 184, A: 255})
	c.evtColor = opt.Color(opts, "eventColor", color.RGBA{R: 154, G: 167, B: 184, A: 255})
	return nil
}

func (c *Calendar) Schema() *engine.Schema {
	return &engine.Schema{
		Name:        "calendar",
		Description: "Upcoming events from a public ICS calendar feed",
		Fields: []engine.Field{
			{Key: "url", Label: "ICS URL", Kind: engine.FieldText, Required: true, Placeholder: "https://calendar.google.com/calendar/ical/...", Hint: "Public ICS feed URL"},
			{Key: "username", Label: "Username", Kind: engine.FieldText, Default: "", Hint: "Basic auth username (if required)"},
			{Key: "password", Label: "Password", Kind: engine.FieldPassword, Default: "", Hint: "Basic auth password (if required)"},
			{Key: "days", Label: "Days Ahead", Kind: engine.FieldNumber, Default: "3", Min: 1, Max: 14},
			{Key: "maxEvents", Label: "Max Events", Kind: engine.FieldNumber, Default: "5", Min: 1, Max: 20},
			{Key: "update", Label: "Update Interval (s)", Kind: engine.FieldDuration, Default: "3600", Hint: "Seconds between feed refreshes"},
			{Key: "dateColor", Label: "Date Color", Kind: engine.FieldColor, Default: "#9aa7b8"},
			{Key: "eventColor", Label: "Event Color", Kind: engine.FieldColor, Default: "#9aa7b8"},
		},
	}
}

// Start launches the background fetch loop.
func (c *Calendar) Start(_ *engine.Bus, log *engine.Logger) error {
	c.done = make(chan struct{})
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		if err := c.fetch(); err != nil {
			c.setErr(err)
			log.Errorf("calendar: %v", err)
		}
		t := time.NewTicker(c.interval)
		defer t.Stop()
		for {
			select {
			case <-t.C:
				if err := c.fetch(); err != nil {
					c.setErr(err)
					log.Errorf("calendar: %v", err)
				}
			case <-c.done:
				return
			}
		}
	}()
	return nil
}

// Stop halts the background fetch loop.
func (c *Calendar) Stop() error {
	if c.done != nil {
		close(c.done)
		c.wg.Wait()
	}
	return nil
}

func (c *Calendar) setErr(err error) {
	c.mu.Lock()
	c.lastErr = err
	c.mu.Unlock()
}

// fetch downloads and parses the feed, storing the upcoming events.
func (c *Calendar) fetch() error {
	events, err := c.fetchEvents(context.Background())
	if err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.events = events
	c.fetched = true
	c.lastErr = nil
	return nil
}

// fetchEvents performs the network/parse work for fetch.
func (c *Calendar) fetchEvents(ctx context.Context) ([]event, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.url, nil)
	if err != nil {
		return nil, err
	}
	if c.username != "" {
		req.SetBasicAuth(c.username, c.password)
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("feed %s: %s", c.url, resp.Status)
	}
	cal, err := ical.ParseCalendar(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, fmt.Errorf("parse ICS: %w", err)
	}

	now := time.Now()
	lo := now.Add(-24 * time.Hour)
	hi := now.Add(time.Duration(c.days) * 24 * time.Hour)

	var events []event
	for _, ev := range cal.Events() {
		e, ok := parseEvent(ev, lo, hi)
		if ok {
			events = append(events, e)
		}
	}
	sort.Slice(events, func(i, j int) bool { return events[i].start.Before(events[j].start) })
	if len(events) > c.maxEvents {
		events = events[:c.maxEvents]
	}
	return events, nil
}

// parseEvent extracts a single event if it falls within the window.
func parseEvent(ev *ical.VEvent, lo, hi time.Time) (event, bool) {
	summary := ""
	if p := ev.GetProperty(ical.ComponentPropertySummary); p != nil {
		summary = p.Value
	}
	startProp := ev.GetProperty(ical.ComponentPropertyDtStart)
	allDay := false
	if startProp != nil {
		vals := startProp.ICalParameters["VALUE"]
		allDay = len(vals) == 1 && vals[0] == "DATE"
	}

	var (
		start time.Time
		err   error
	)
	if allDay {
		start, err = ev.GetAllDayStartAt()
	} else {
		start, err = ev.GetStartAt()
	}
	if err != nil {
		return event{}, false
	}
	if start.Before(lo) || start.After(hi) {
		return event{}, false
	}
	return event{summary: summary, start: start, allDay: allDay}, true
}

// Draw renders the event list, or a placeholder before the first fetch.
func (c *Calendar) Draw(cv *render.Canvas, bounds image.Rectangle, now time.Time) error {
	c.mu.Lock()
	events := c.events
	fetched := c.fetched
	lastErr := c.lastErr
	c.mu.Unlock()

	if !fetched {
		msg := "calendar: collecting…"
		if lastErr != nil {
			msg = "calendar: unavailable"
		}
		return drawCenter(cv, bounds, msg, c.dateColor)
	}
	if len(events) == 0 {
		return drawCenter(cv, bounds, "no upcoming events", c.dateColor)
	}

	tf := fonts.Scaled(bounds, 15, fonts.Medium)
	ef := fonts.Scaled(bounds, 15, fonts.Regular)
	rowH := 24
	dateCol := 112
	summaryW := bounds.Dx() - dateCol

	ascentT := cv.Ascent(tf)
	_, _, tfH := cv.FaceMetrics(tf)
	_, _, efH := cv.FaceMetrics(ef)
	textH := tfH
	if efH > textH {
		textH = efH
	}

	startY := bounds.Min.Y + (bounds.Dy()-rowH*len(events))/2
	y := startY
	for _, ev := range events {
		dateStr := dayLabel(ev, now)
		if !ev.allDay {
			dateStr += " " + ev.start.Format("15:04")
		}
		cv.DrawText(image.Pt(bounds.Min.X, y+ascentT), dateStr, tf, c.dateColor)

		summary := truncateToFit(cv, ef, ev.summary, summaryW)
		cv.DrawText(image.Pt(bounds.Min.X+dateCol, y+ascentT+(tfH-efH)/2), summary, ef, c.evtColor)
		y += rowH
	}
	return nil
}

// dayLabel renders a friendly day marker for an event relative to now.
func dayLabel(ev event, now time.Time) string {
	s := ev.start
	switch {
	case sameDay(s, now):
		return "Today"
	case sameDay(s, now.Add(24*time.Hour)):
		return "Tomorrow"
	default:
		return s.Format("Mon 1/2")
	}
}

func sameDay(a, b time.Time) bool {
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	return ay == by && am == bm && ad == bd
}

// truncateToFit shortens s with an ellipsis to fit within maxW pixels.
func truncateToFit(cv *render.Canvas, f font.Face, s string, maxW int) string {
	if maxW <= 0 {
		return ""
	}
	w, _ := cv.TextSize(f, s)
	if w <= maxW {
		return s
	}
	runes := []rune(s)
	lo, hi := 0, len(runes)
	cut := hi
	for lo <= hi {
		mid := (lo + hi) / 2
		cand := string(runes[:mid]) + "…"
		if cw, _ := cv.TextSize(f, cand); cw <= maxW {
			cut = mid
			lo = mid + 1
		} else {
			hi = mid - 1
		}
	}
	if cut <= 0 {
		return "…"
	}
	return string(runes[:cut]) + "…"
}

// drawCenter renders a single centered status line.
func drawCenter(cv *render.Canvas, bounds image.Rectangle, msg string, col color.RGBA) error {
	f := fonts.Scaled(bounds, 16, fonts.Regular)
	lines := render.WrapText(f, msg, bounds.Dx())
	_, _, lh := cv.FaceMetrics(f)
	ascent := cv.Ascent(f)
	y := bounds.Min.Y + (bounds.Dy()-len(lines)*lh)/2
	for _, line := range lines {
		w, _ := cv.TextSize(f, line)
		cv.DrawText(image.Pt(bounds.Min.X+(bounds.Dx()-w)/2, y+ascent), line, f, col)
		y += lh
	}
	return nil
}
