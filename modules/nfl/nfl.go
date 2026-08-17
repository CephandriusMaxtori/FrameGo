// Package nfl implements the NFL module: live scores and schedules from the
// public ESPN scoreboard API (no API key required), refreshed on a background
// ticker and rendered from a mutex-guarded snapshot.
package nfl

import (
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"framego/engine"
	"framego/fonts"
	"framego/modules"
	"framego/modules/opt"
	"framego/render"
)

// defaultURL is the ESPN NFL scoreboard endpoint. Overridable on the module
// for tests via the "url" option.
const defaultURL = "https://site.api.espn.com/apis/site/v2/sports/football/nfl/scoreboard"

// NFL renders the current week's games, live scores, and game statuses.
type NFL struct {
	team     string
	games    int
	interval time.Duration
	url      string
	client   *http.Client

	done chan struct{}
	wg   sync.WaitGroup

	mu      sync.Mutex
	scores  []game
	fetched bool
	lastErr error

	teamColor  color.RGBA
	scoreColor color.RGBA
	timeColor  color.RGBA
}

type game struct {
	awayAbbr  string
	homeAbbr  string
	awayScore int
	homeScore int
	scored    bool
	state     string
	detail    string
	start     time.Time
}

// New constructs an NFL module.
func New() engine.Module { return &NFL{} }

func init() { modules.Register("nfl", New) }

// Name identifies the module.
func (n *NFL) Name() string { return "nfl" }

// Configure applies module options.
func (n *NFL) Configure(opts map[string]any) error {
	n.team = strings.ToUpper(opt.Str(opts, "team", ""))
	n.games = opt.Int(opts, "games", 4)
	if n.games < 1 {
		n.games = 1
	}
	if n.games > 10 {
		n.games = 10
	}
	n.interval = opt.Duration(opts, "update", 300)
	if n.interval <= 0 {
		n.interval = 5 * time.Minute
	}
	n.url = opt.Str(opts, "url", defaultURL)
	if n.client == nil {
		n.client = &http.Client{Timeout: 20 * time.Second}
	}

	n.teamColor = opt.Color(opts, "teamColor", color.RGBA{R: 245, G: 247, B: 250, A: 255})
	n.scoreColor = opt.Color(opts, "scoreColor", color.RGBA{R: 245, G: 247, B: 250, A: 255})
	n.timeColor = opt.Color(opts, "timeColor", color.RGBA{R: 154, G: 167, B: 184, A: 255})
	return nil
}

// Start launches the background fetch loop.
func (n *NFL) Start(_ *engine.Bus, log *engine.Logger) error {
	n.done = make(chan struct{})
	n.wg.Add(1)
	go func() {
		defer n.wg.Done()
		if err := n.fetch(); err != nil {
			n.setErr(err)
			log.Errorf("nfl: %v", err)
		}
		t := time.NewTicker(n.interval)
		defer t.Stop()
		for {
			select {
			case <-t.C:
				if err := n.fetch(); err != nil {
					n.setErr(err)
					log.Errorf("nfl: %v", err)
				}
			case <-n.done:
				return
			}
		}
	}()
	return nil
}

// Stop halts the background fetch loop.
func (n *NFL) Stop() error {
	if n.done != nil {
		close(n.done)
		n.wg.Wait()
	}
	return nil
}

func (n *NFL) setErr(err error) {
	n.mu.Lock()
	n.lastErr = err
	n.mu.Unlock()
}

// fetch pulls the scoreboard and stores the game list.
func (n *NFL) fetch() error {
	games, err := n.fetchGames(context.Background())
	if err != nil {
		return err
	}
	n.mu.Lock()
	defer n.mu.Unlock()
	n.scores = games
	n.fetched = true
	n.lastErr = nil
	return nil
}

// fetchGames performs the network/parse work for fetch.
func (n *NFL) fetchGames(ctx context.Context) ([]game, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, n.url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := n.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("scoreboard %s: %s", n.url, resp.Status)
	}

	var sb scoreboard
	if err := json.Unmarshal(body, &sb); err != nil {
		return nil, fmt.Errorf("parse scoreboard: %w", err)
	}

	var games []game
	for _, ev := range sb.Events {
		g, ok := parseGame(ev)
		if !ok {
			continue
		}
		if n.team != "" && !g.involves(n.team) {
			continue
		}
		games = append(games, g)
	}
	sort.Slice(games, func(i, j int) bool { return games[i].start.Before(games[j].start) })
	if len(games) > n.games {
		games = games[:n.games]
	}
	return games, nil
}

// involves reports whether the game features the given team abbreviation.
func (g game) involves(team string) bool {
	return strings.EqualFold(g.awayAbbr, team) || strings.EqualFold(g.homeAbbr, team)
}

// parseGame converts an ESPN event into a game, or false on malformed input.
func parseGame(ev event) (game, bool) {
	if len(ev.Competitions) == 0 {
		return game{}, false
	}
	comp := ev.Competitions[0]
	g := game{
		state: strings.ToLower(comp.Status.Type.State),
		start: ev.Date.Time,
	}
	for _, c := range comp.Competitors {
		abbr := strings.ToUpper(c.Team.Abbreviation)
		if abbr == "" {
			abbr = strings.ToUpper(c.Team.DisplayName)
		}
		if c.HomeAway == "home" {
			g.homeAbbr = abbr
			g.homeScore, _ = strconv.Atoi(c.Score)
		} else {
			g.awayAbbr = abbr
			g.awayScore, _ = strconv.Atoi(c.Score)
		}
	}
	if g.homeAbbr == "" || g.awayAbbr == "" {
		return game{}, false
	}
	if g.state == "pre" {
		g.detail = g.start.In(time.Local).Format("Mon 3:04 PM")
	} else if g.state == "in" {
		g.detail = comp.Status.Type.ShortDetail
	} else if g.state == "post" {
		g.detail = "Final"
	} else {
		g.detail = comp.Status.Type.Detail
	}
	g.scored = g.homeScore != 0 || g.awayScore != 0
	return g, true
}

// Draw renders the game list, or a placeholder before the first fetch.
func (n *NFL) Draw(cv *render.Canvas, bounds image.Rectangle, now time.Time) error {
	n.mu.Lock()
	scores := n.scores
	fetched := n.fetched
	lastErr := n.lastErr
	n.mu.Unlock()

	if !fetched {
		msg := "nfl: collecting…"
		if lastErr != nil {
			msg = "nfl: unavailable"
		}
		return drawCenter(cv, bounds, msg, n.timeColor)
	}
	if len(scores) == 0 {
		return drawCenter(cv, bounds, "no games scheduled", n.timeColor)
	}

	tf := fonts.Face(15, fonts.Medium)
	sf := fonts.Face(15, fonts.Bold)
	df := fonts.Face(13, fonts.Regular)
	rowH := 36
	ascentT := cv.Ascent(tf)
	ascentS := cv.Ascent(sf)
	_, _, tfH := cv.FaceMetrics(tf)

	startY := bounds.Min.Y + (bounds.Dy()-rowH*len(scores))/2
	y := startY
	for _, g := range scores {
		matchup := g.awayAbbr + "  @  " + g.homeAbbr
		baseline := y + ascentT
		cv.DrawText(image.Pt(bounds.Min.X, baseline), matchup, tf, n.teamColor)
		dw, _ := cv.TextSize(df, g.detail)
		cv.DrawText(image.Pt(bounds.Min.X+bounds.Dx()-dw, baseline), g.detail, df, n.timeColor)
		if g.scored {
			score := fmt.Sprintf("%d  –  %d", g.awayScore, g.homeScore)
			cv.DrawText(image.Pt(bounds.Min.X, y+tfH+ascentS), score, sf, n.scoreColor)
		}
		y += rowH
	}
	return nil
}

// drawCenter renders a single centered status line.
func drawCenter(cv *render.Canvas, bounds image.Rectangle, msg string, col color.RGBA) error {
	f := fonts.Face(16, fonts.Regular)
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

type scoreboard struct {
	Events []event `json:"events"`
}

type event struct {
	Date         gameTime      `json:"date"`
	Competitions []competition `json:"competitions"`
}

// gameTime is a time.Time that also accepts RFC3339 timestamps without a
// seconds component (e.g. "2026-08-15T17:00Z"), which ESPN sometimes emits.
type gameTime struct {
	time.Time
}

// UnmarshalJSON parses the event date from the common ESPN layouts.
func (t *gameTime) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), `"`)
	if s == "" || s == "null" {
		return nil
	}
	for _, layout := range []string{
		time.RFC3339,
		"2006-01-02T15:04Z07:00",
		"2006-01-02T15:04:05",
		"2006-01-02T15:04",
	} {
		if v, err := time.Parse(layout, s); err == nil {
			t.Time = v
			return nil
		}
	}
	return fmt.Errorf("parse event date %q: unsupported layout", s)
}

type competition struct {
	Status struct {
		Type struct {
			State       string `json:"state"`
			Detail      string `json:"detail"`
			ShortDetail string `json:"shortDetail"`
		} `json:"type"`
	} `json:"status"`
	Competitors []struct {
		HomeAway string `json:"homeAway"`
		Score    string `json:"score"`
		Team     struct {
			Abbreviation string `json:"abbreviation"`
			DisplayName  string `json:"displayName"`
		} `json:"team"`
	} `json:"competitors"`
}
