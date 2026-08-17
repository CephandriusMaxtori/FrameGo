package calendar

import (
	"context"
	"image"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"framego/fonts"
	"framego/render"
)

const icsFeed = `BEGIN:VCALENDAR
VERSION:2.0
PRODID:-//test//EN
BEGIN:VEVENT
UID:1
DTSTAMP:20260101T000000Z
DTSTART:20260820T140000Z
DTEND:20260820T150000Z
SUMMARY:Standup sync
END:VEVENT
BEGIN:VEVENT
UID:2
DTSTART;VALUE=DATE:20260821
SUMMARY:All day offsite
END:VEVENT
BEGIN:VEVENT
UID:3
DTSTAMP:20260101T000000Z
DTSTART:20200801T090000Z
SUMMARY:Ancient event
END:VEVENT
END:VCALENDAR
`

func testServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/calendar")
		_, _ = w.Write([]byte(icsFeed))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestConfigureRequiresURL(t *testing.T) {
	c := New().(*Calendar)
	if err := c.Configure(nil); err == nil {
		t.Error("expected error when url missing")
	}
}

func TestFetchParsesEvents(t *testing.T) {
	srv := testServer(t)
	c := New().(*Calendar)
	if err := c.Configure(map[string]any{"url": srv.URL, "days": 30, "maxEvents": 5}); err != nil {
		t.Fatal(err)
	}
	events, err := c.fetchEvents(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	// Ancient event filtered; two remain, sorted by start.
	if len(events) != 2 {
		t.Fatalf("events = %+v", events)
	}
	if events[0].summary != "Standup sync" || events[0].allDay {
		t.Errorf("first event = %+v", events[0])
	}
	if events[1].summary != "All day offsite" || !events[1].allDay {
		t.Errorf("second event = %+v", events[1])
	}
}

func TestDayLabels(t *testing.T) {
	now := time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)
	cases := []struct {
		start time.Time
		want  string
	}{
		{time.Date(2026, 8, 15, 9, 0, 0, 0, time.UTC), "Today"},
		{time.Date(2026, 8, 16, 9, 0, 0, 0, time.UTC), "Tomorrow"},
		{time.Date(2026, 8, 18, 9, 0, 0, 0, time.UTC), "Tue 8/18"},
	}
	for _, tc := range cases {
		if got := dayLabel(event{start: tc.start}, now); got != tc.want {
			t.Errorf("dayLabel(%v) = %q, want %q", tc.start, got, tc.want)
		}
	}
}

func TestDrawEvents(t *testing.T) {
	srv := testServer(t)
	c := New().(*Calendar)
	if err := c.Configure(map[string]any{"url": srv.URL, "days": 30}); err != nil {
		t.Fatal(err)
	}
	if err := c.fetch(); err != nil {
		t.Fatal(err)
	}
	cv := render.NewCanvas(300, 120)
	if err := c.Draw(cv, image.Rect(0, 0, 300, 120), time.Now()); err != nil {
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

func TestDrawBeforeFetch(t *testing.T) {
	c := New().(*Calendar)
	if err := c.Configure(map[string]any{"url": "http://example.invalid/cal.ics"}); err != nil {
		t.Fatal(err)
	}
	cv := render.NewCanvas(200, 60)
	if err := c.Draw(cv, image.Rect(0, 0, 200, 60), time.Now()); err != nil {
		t.Fatal(err)
	}
}

func TestTruncate(t *testing.T) {
	cv := render.NewCanvas(300, 40)
	long := "a very long event summary that definitely exceeds the available space for a single line"
	short := truncateToFit(cv, fonts.Face(13, fonts.Regular), long, 150)
	if len(short) >= len(long) {
		t.Errorf("truncate failed: %q", short)
	}
}
