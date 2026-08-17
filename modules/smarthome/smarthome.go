// Package smarthome implements a Smart Home module that displays Home
// Assistant entity states. Data is fetched on a background ticker; Draw
// renders the latest snapshot without blocking on the network.
package smarthome

import (
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"framego/engine"
	"framego/fonts"
	"framego/modules"
	"framego/modules/opt"
	"framego/render"
)

const defaultStatesURL = "/api/states"

type entity struct {
	EntityID    string         `json:"entity_id"`
	State       string         `json:"state"`
	FriendlyName string        `json:"attributes"`
	Attributes  map[string]any `json:"-"`
}

type entityRaw struct {
	EntityID    string         `json:"entity_id"`
	State       string         `json:"state"`
	Attributes  map[string]any `json:"attributes"`
}

func (e *entity) UnmarshalJSON(data []byte) error {
	var raw entityRaw
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	e.EntityID = raw.EntityID
	e.State = raw.State
	e.Attributes = raw.Attributes
	if name, ok := raw.Attributes["friendly_name"].(string); ok {
		e.FriendlyName = name
	} else {
		e.FriendlyName = raw.EntityID
	}
	return nil
}

type deviceEntry struct {
	name    string
	state   string
	on      bool
	domain  string
}

type snapshot struct {
	devices []deviceEntry
	fetched time.Time
}

// SmartHome renders Home Assistant entity states.
type SmartHome struct {
	url         string
	token       string
	statesURL   string
	entityIDs   []string
	maxDevices  int
	interval    time.Duration
	client      *http.Client

	titleColor  color.RGBA
	stateColor  color.RGBA
	onColor     color.RGBA
	offColor    color.RGBA

	done chan struct{}
	wg   sync.WaitGroup

	mu    sync.Mutex
	snap  *snapshot
	lastErr error
}

func New() engine.Module { return &SmartHome{} }

func init() { modules.Register("smarthome", New) }

func (sh *SmartHome) Name() string { return "smarthome" }

func (sh *SmartHome) Configure(opts map[string]any) error {
	sh.url = strings.TrimRight(opt.Str(opts, "url", ""), "/")
	if sh.url == "" {
		return fmt.Errorf("smarthome: option \"url\" is required (Home Assistant base URL)")
	}
	sh.token = opt.Str(opts, "token", "")
	if sh.token == "" {
		return fmt.Errorf("smarthome: option \"token\" is required (long-lived access token)")
	}
	raw := opt.Str(opts, "entities", "")
	if raw != "" {
		for _, s := range strings.Split(raw, ",") {
			s = strings.TrimSpace(s)
			if s != "" {
				sh.entityIDs = append(sh.entityIDs, s)
			}
		}
	}
	sh.maxDevices = opt.Int(opts, "maxDevices", 10)
	if sh.maxDevices < 1 {
		sh.maxDevices = 1
	}
	if sh.maxDevices > 20 {
		sh.maxDevices = 20
	}
	sh.interval = opt.Duration(opts, "update", 30)
	if sh.interval < 5 {
		sh.interval = 5
	}
	sh.statesURL = opt.Str(opts, "statesURL", "")
	if sh.statesURL == "" {
		sh.statesURL = sh.url + defaultStatesURL
	}
	if sh.client == nil {
		sh.client = &http.Client{Timeout: 10 * time.Second}
	}

	sh.titleColor = opt.Color(opts, "titleColor", color.RGBA{R: 245, G: 247, B: 250, A: 255})
	sh.stateColor = opt.Color(opts, "stateColor", color.RGBA{R: 154, G: 167, B: 184, A: 255})
	sh.onColor = opt.Color(opts, "onColor", color.RGBA{R: 87, G: 199, B: 172, A: 255})
	sh.offColor = opt.Color(opts, "offColor", color.RGBA{R: 100, G: 116, B: 139, A: 255})
	return nil
}

func (sh *SmartHome) Schema() *engine.Schema {
	return &engine.Schema{
		Name:        "smarthome",
		Description: "Home Assistant entity states (lights, thermostats, locks, sensors, …)",
		Fields: []engine.Field{
			{Key: "url", Label: "Home Assistant URL", Kind: engine.FieldText, Required: true, Placeholder: "http://192.168.1.100:8123", Hint: "Base URL of your HA instance"},
			{Key: "token", Label: "Access Token", Kind: engine.FieldPassword, Required: true, Hint: "Long-lived access token from HA profile"},
			{Key: "entities", Label: "Entities", Kind: engine.FieldText, Placeholder: "light.living_room, thermostat.bedroom", Hint: "Comma-separated entity IDs (blank = all)"},
			{Key: "maxDevices", Label: "Max Devices", Kind: engine.FieldNumber, Default: "10", Min: 1, Max: 20},
			{Key: "update", Label: "Update Interval (s)", Kind: engine.FieldDuration, Default: "30"},
			{Key: "titleColor", Label: "Name Color", Kind: engine.FieldColor, Default: "#f5f7fa"},
			{Key: "stateColor", Label: "State Color", Kind: engine.FieldColor, Default: "#9aa7b8"},
			{Key: "onColor", Label: "On Indicator", Kind: engine.FieldColor, Default: "#57c7ac"},
			{Key: "offColor", Label: "Off Indicator", Kind: engine.FieldColor, Default: "#64748b"},
		},
	}
}

func (sh *SmartHome) Start(_ *engine.Bus, log *engine.Logger) error {
	sh.done = make(chan struct{})
	sh.wg.Add(1)
	go func() {
		defer sh.wg.Done()
		if err := sh.fetch(); err != nil {
			sh.setErr(err)
			log.Errorf("smarthome: %v", err)
		}
		t := time.NewTicker(sh.interval)
		defer t.Stop()
		for {
			select {
			case <-t.C:
				if err := sh.fetch(); err != nil {
					sh.setErr(err)
					log.Errorf("smarthome: %v", err)
				}
			case <-sh.done:
				return
			}
		}
	}()
	return nil
}

func (sh *SmartHome) Stop() error {
	if sh.done != nil {
		close(sh.done)
		sh.wg.Wait()
	}
	return nil
}

func (sh *SmartHome) setErr(err error) {
	sh.mu.Lock()
	sh.lastErr = err
	sh.mu.Unlock()
}

func (sh *SmartHome) fetch() error {
	entities, err := sh.getStates(context.Background())
	if err != nil {
		return err
	}
	filtered := sh.filterEntities(entities)
	snap := &snapshot{
		devices: filtered,
		fetched: time.Now(),
	}
	sh.mu.Lock()
	sh.snap = snap
	sh.lastErr = nil
	sh.mu.Unlock()
	return nil
}

func (sh *SmartHome) getStates(ctx context.Context) ([]entity, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, sh.statesURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+sh.token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := sh.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HA %s: %s", resp.Status, body)
	}
	var entities []entity
	if err := json.Unmarshal(body, &entities); err != nil {
		return nil, fmt.Errorf("decode states: %w", err)
	}
	return entities, nil
}

func (sh *SmartHome) filterEntities(all []entity) []deviceEntry {
	var filtered []entity
	if len(sh.entityIDs) > 0 {
		lookup := make(map[string]bool, len(sh.entityIDs))
		for _, id := range sh.entityIDs {
			lookup[strings.ToLower(id)] = true
		}
		for _, e := range all {
			if lookup[strings.ToLower(e.EntityID)] {
				filtered = append(filtered, e)
			}
		}
	} else {
		for _, e := range all {
			domain := entityDomain(e.EntityID)
			switch domain {
			case "light", "switch", "climate", "lock", "cover", "fan", "media_player", "sensor", "binary_sensor":
				filtered = append(filtered, e)
			}
		}
	}
	var devices []deviceEntry
	for _, e := range filtered {
		domain := entityDomain(e.EntityID)
		name := e.FriendlyName
		if name == "" {
			name = entityIDShort(e.EntityID)
		}
		state := formatState(domain, e.State, e.Attributes)
		on := isOn(domain, e.State)
		devices = append(devices, deviceEntry{
			name:   name,
			state:  state,
			on:     on,
			domain: domain,
		})
	}
	sort.Slice(devices, func(i, j int) bool {
		if devices[i].on != devices[j].on {
			return devices[i].on
		}
		return devices[i].name < devices[j].name
	})
	if sh.maxDevices > 0 && len(devices) > sh.maxDevices {
		devices = devices[:sh.maxDevices]
	}
	return devices
}

func (sh *SmartHome) Draw(cv *render.Canvas, bounds image.Rectangle, _ time.Time) error {
	sh.mu.Lock()
	snap := sh.snap
	lastErr := sh.lastErr
	sh.mu.Unlock()

	if snap == nil {
		msg := "smart home: connecting…"
		if lastErr != nil {
			msg = "smart home: unavailable"
		}
		return drawCenter(cv, bounds, msg, sh.stateColor)
	}

	if len(snap.devices) == 0 {
		return drawCenter(cv, bounds, "smart home: no devices", sh.stateColor)
	}

	titleF := fonts.Scaled(bounds, 20, fonts.Medium)
	nameF := fonts.Scaled(bounds, 14, fonts.Regular)
	stateF := fonts.Scaled(bounds, 12, fonts.Regular)

	rowH := int(float64(bounds.Dy()) / float64(sh.maxDevices+1))
	if rowH < 28 {
		rowH = 28
	}
	if rowH > 48 {
		rowH = 48
	}

	dotR := 4
	padX := 10
	padY := 6
	titleH := int(float64(rowH) * 0.7)
	y := bounds.Min.Y + padY

	titleStr := fmt.Sprintf("Home (%d)", len(snap.devices))
	_, _ = cv.TextSize(titleF, titleStr)
	cv.DrawText(image.Pt(bounds.Min.X+padX, y+cv.Ascent(titleF)), titleStr, titleF, sh.titleColor)
	y += titleH

	for _, d := range snap.devices {
		if y+rowH > bounds.Max.Y-padY {
			break
		}

		dotColor := sh.offColor
		if d.on {
			dotColor = sh.onColor
		}
		dotY := y + rowH/2
		dotX := bounds.Min.X + padX + dotR
		cv.FillCircle(image.Pt(dotX, dotY), dotR, dotColor)

		textX := dotX + dotR + 10
		availW := bounds.Dx() - (textX-bounds.Min.X) - padX

		nameW, nameH := cv.TextSize(nameF, d.name)
		if nameW > availW*6/10 {
			nameW = availW * 6 / 10
		}
		cv.DrawText(image.Pt(textX, y+rowH/2-nameH/2+cv.Ascent(nameF)), d.name, nameF, sh.titleColor)

		stateW, stateH := cv.TextSize(stateF, d.state)
		stateX := bounds.Max.X - padX - stateW
		if stateX > textX+nameW+10 {
			cv.DrawText(image.Pt(stateX, y+rowH/2-stateH/2+cv.Ascent(stateF)), d.state, stateF, sh.stateColor)
		}

		y += rowH
	}
	return nil
}

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

func entityDomain(entityID string) string {
	parts := strings.SplitN(entityID, ".", 2)
	if len(parts) == 2 {
		return parts[0]
	}
	return ""
}

func entityIDShort(entityID string) string {
	parts := strings.SplitN(entityID, ".", 2)
	if len(parts) == 2 {
		name := parts[1]
		name = strings.ReplaceAll(name, "_", " ")
		return name
	}
	return entityID
}

func isOn(domain, state string) bool {
	switch domain {
	case "light", "switch", "fan":
		return strings.EqualFold(state, "on")
	case "lock":
		return strings.EqualFold(state, "unlocked")
	case "cover":
		return strings.EqualFold(state, "open") || strings.EqualFold(state, "opening")
	case "media_player":
		return strings.EqualFold(state, "playing") || strings.EqualFold(state, "paused")
	case "climate":
		return !strings.EqualFold(state, "off")
	case "binary_sensor":
		return strings.EqualFold(state, "on") || strings.EqualFold(state, "open") || strings.EqualFold(state, "detected")
	default:
		return state != "" && state != "unavailable" && state != "unknown"
	}
}

func formatState(domain, state string, attrs map[string]any) string {
	switch domain {
	case "climate":
		if t, ok := attrs["temperature"].(float64); ok {
			return fmt.Sprintf("%.0f°C", t)
		}
		return state
	case "light":
		if br, ok := attrs["brightness"].(float64); ok && br > 0 {
			pct := int(br / 255.0 * 100)
			return fmt.Sprintf("%d%%", pct)
		}
		return state
	case "sensor", "binary_sensor":
		if u, ok := attrs["unit_of_measurement"].(string); ok {
			return state + u
		}
		return state
	case "lock":
		return state
	default:
		return state
	}
}
