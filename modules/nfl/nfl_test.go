package nfl

import (
	"image"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"framego/render"
)

const fixture = `{"events":[
  {"date":"2026-09-13T17:00:00Z","competitions":[{"status":{"type":{"state":"post","detail":"Final","shortDetail":"Final"}},"competitors":[
    {"homeAway":"away","score":"21","team":{"abbreviation":"KC","displayName":"Chiefs"}},
    {"homeAway":"home","score":"14","team":{"abbreviation":"DET","displayName":"Lions"}}]}]},
  {"date":"2026-09-13T17:00:00Z","competitions":[{"status":{"type":{"state":"in","detail":"Quarter 2","shortDetail":"Q2 5:32"}},"competitors":[
    {"homeAway":"away","score":"10","team":{"abbreviation":"BUF","displayName":"Bills"}},
    {"homeAway":"home","score":"7","team":{"abbreviation":"MIA","displayName":"Dolphins"}}]}]},
  {"date":"2026-09-17T17:15:00Z","competitions":[{"status":{"type":{"state":"pre","detail":"Sun, Sep 13","shortDetail":"1:15 PM"}},"competitors":[
    {"homeAway":"away","score":"","team":{"abbreviation":"PHI","displayName":"Eagles"}},
    {"homeAway":"home","score":"","team":{"abbreviation":"NYG","displayName":"Giants"}}]}]}
]}`

func testServer(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/scoreboard", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(fixture))
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func TestConfigureDefaults(t *testing.T) {
	n := New().(*NFL)
	if err := n.Configure(nil); err != nil {
		t.Fatal(err)
	}
	if n.games != 4 || n.interval != 5*time.Minute {
		t.Errorf("defaults: games=%d interval=%v", n.games, n.interval)
	}
	if n.url != defaultURL {
		t.Errorf("url = %q", n.url)
	}
}

func TestConfigureClamps(t *testing.T) {
	n := New().(*NFL)
	if err := n.Configure(map[string]any{"games": 50}); err != nil {
		t.Fatal(err)
	}
	if n.games != 10 {
		t.Errorf("games = %d, want 10", n.games)
	}
}

func TestFetch(t *testing.T) {
	srv := testServer(t)
	n := New().(*NFL)
	if err := n.Configure(map[string]any{"url": srv.URL + "/scoreboard"}); err != nil {
		t.Fatal(err)
	}
	if err := n.fetch(); err != nil {
		t.Fatal(err)
	}
	n.mu.Lock()
	scores := n.scores
	fetched := n.fetched
	n.mu.Unlock()
	if !fetched {
		t.Fatal("no snapshot after fetch")
	}
	if len(scores) != 3 {
		t.Fatalf("games = %d, want 3", len(scores))
	}
	if scores[0].awayAbbr != "KC" || scores[0].homeScore != 14 || scores[0].detail != "Final" {
		t.Errorf("game 0 = %+v", scores[0])
	}
	if scores[1].detail != "Q2 5:32" {
		t.Errorf("game 1 detail = %q", scores[1].detail)
	}
	if scores[2].state != "pre" || scores[2].scored {
		t.Errorf("game 2 = %+v", scores[2])
	}
}

func TestFetchTeamFilter(t *testing.T) {
	srv := testServer(t)
	n := New().(*NFL)
	if err := n.Configure(map[string]any{"url": srv.URL + "/scoreboard", "team": "mia"}); err != nil {
		t.Fatal(err)
	}
	if err := n.fetch(); err != nil {
		t.Fatal(err)
	}
	n.mu.Lock()
	scores := n.scores
	n.mu.Unlock()
	if len(scores) != 1 || scores[0].homeAbbr != "MIA" {
		t.Errorf("filtered games = %+v", scores)
	}
}

func TestFetchBadStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	n := New().(*NFL)
	_ = n.Configure(map[string]any{"url": srv.URL})
	if err := n.fetch(); err == nil {
		t.Error("expected error for non-200 response")
	}
}

func TestFetchNoSecondsDate(t *testing.T) {
	const fixture = `{"events":[
	  {"date":"2026-08-15T17:00Z","competitions":[{"status":{"type":{"state":"pre","detail":"Sat, Aug 15","shortDetail":"1:00 PM"}},"competitors":[
	    {"homeAway":"away","score":"","team":{"abbreviation":"CLE","displayName":"Browns"}},
	    {"homeAway":"home","score":"","team":{"abbreviation":"CAR","displayName":"Panthers"}}]}]}
	]}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(fixture))
	}))
	defer srv.Close()
	n := New().(*NFL)
	if err := n.Configure(map[string]any{"url": srv.URL}); err != nil {
		t.Fatal(err)
	}
	if err := n.fetch(); err != nil {
		t.Fatal(err)
	}
	n.mu.Lock()
	scores := n.scores
	n.mu.Unlock()
	if len(scores) != 1 || scores[0].awayAbbr != "CLE" {
		t.Fatalf("games = %+v", scores)
	}
	if want := "17:00"; scores[0].start.UTC().Format("15:04") != want {
		t.Errorf("start = %v, want %s", scores[0].start, want)
	}
}

func TestDrawBeforeFetch(t *testing.T) {
	n := New().(*NFL)
	if err := n.Configure(nil); err != nil {
		t.Fatal(err)
	}
	cv := render.NewCanvas(200, 60)
	if err := n.Draw(cv, image.Rect(0, 0, 200, 60), time.Now()); err != nil {
		t.Fatal(err)
	}
}

func TestDrawAfterFetch(t *testing.T) {
	srv := testServer(t)
	n := New().(*NFL)
	if err := n.Configure(map[string]any{"url": srv.URL + "/scoreboard", "games": 2}); err != nil {
		t.Fatal(err)
	}
	if err := n.fetch(); err != nil {
		t.Fatal(err)
	}
	cv := render.NewCanvas(260, 140)
	if err := n.Draw(cv, image.Rect(0, 0, 260, 140), time.Now()); err != nil {
		t.Fatal(err)
	}
	lit := 0
	for y := 0; y < 140; y += 2 {
		for x := 0; x < 260; x += 2 {
			if _, _, _, a := cv.Img.At(x, y).RGBA(); a > 0 {
				lit++
			}
		}
	}
	if lit == 0 {
		t.Error("no text pixels rendered")
	}
}
