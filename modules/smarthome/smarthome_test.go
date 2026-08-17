package smarthome

import (
	"encoding/json"
	"image"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"framego/render"
)

func TestConfigure(t *testing.T) {
	sh := &SmartHome{}
	if err := sh.Configure(map[string]any{}); err == nil {
		t.Fatal("expected error for missing url")
	}
	if err := sh.Configure(map[string]any{"url": "http://localhost:8123"}); err == nil {
		t.Fatal("expected error for missing token")
	}
	if err := sh.Configure(map[string]any{"url": "http://localhost:8123", "token": "abc"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestConfigureEntities(t *testing.T) {
	sh := &SmartHome{}
	if err := sh.Configure(map[string]any{
		"url":      "http://localhost:8123",
		"token":    "abc",
		"entities": "light.kitchen, switch.living_room",
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sh.entityIDs) != 2 {
		t.Fatalf("expected 2 entities, got %d", len(sh.entityIDs))
	}
	if sh.entityIDs[0] != "light.kitchen" {
		t.Errorf("expected light.kitchen, got %s", sh.entityIDs[0])
	}
}

func TestSchema(t *testing.T) {
	sh := &SmartHome{}
	s := sh.Schema()
	if s == nil {
		t.Fatal("schema is nil")
	}
	if len(s.Fields) == 0 {
		t.Fatal("schema has no fields")
	}
}

func TestFetchAndDraw(t *testing.T) {
	haStates := []entityRaw{
		{
			EntityID: "light.kitchen",
			State:    "on",
			Attributes: map[string]any{
				"friendly_name": "Kitchen Light",
				"brightness":    128.0,
			},
		},
		{
			EntityID: "switch.living_room",
			State:    "off",
			Attributes: map[string]any{
				"friendly_name": "Living Room Switch",
			},
		},
		{
			EntityID: "climate.bedroom",
			State:    "heat",
			Attributes: map[string]any{
				"friendly_name": "Bedroom Thermostat",
				"temperature":   21.0,
			},
		},
		{
			EntityID: "sensor.hallway",
			State:    "22.5",
			Attributes: map[string]any{
				"friendly_name":          "Hallway Temp",
				"unit_of_measurement": "°C",
			},
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(haStates)
	}))
	defer srv.Close()

	sh := &SmartHome{}
	if err := sh.Configure(map[string]any{
		"url":       srv.URL,
		"token":     "test-token",
		"maxDevices": 10,
		"update":    60,
		"statesURL": srv.URL + "/api/states",
	}); err != nil {
		t.Fatalf("configure: %v", err)
	}
	sh.client = srv.Client()

	if err := sh.fetch(); err != nil {
		t.Fatalf("fetch: %v", err)
	}

	sh.mu.Lock()
	snap := sh.snap
	sh.mu.Unlock()

	if snap == nil {
		t.Fatal("snapshot is nil after fetch")
	}
	if len(snap.devices) != 4 {
		t.Fatalf("expected 4 devices, got %d", len(snap.devices))
	}

	if snap.devices[0].name != "Bedroom Thermostat" {
		t.Errorf("first device should be Bedroom Thermostat (on, alphabetically first), got %q", snap.devices[0].name)
	}
	if !snap.devices[0].on {
		t.Error("Bedroom Thermostat should be on")
	}

	bounds := image.Rect(0, 0, 800, 480)
	canvas := render.NewCanvas(800, 480)
	if err := sh.Draw(canvas, bounds, time.Now()); err != nil {
		t.Fatalf("draw: %v", err)
	}

	hasContent := false
	for y := 0; y < bounds.Dy(); y++ {
		for x := 0; x < bounds.Dx(); x++ {
			r, g, b, a := canvas.Img.RGBAAt(x, y).RGBA()
			if a > 0 && (r > 0 || g > 0 || b > 0) {
				hasContent = true
				break
			}
		}
		if hasContent {
			break
		}
	}
	if !hasContent {
		t.Error("canvas is blank after draw")
	}
}

func TestFetchUnauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	sh := &SmartHome{}
	sh.Configure(map[string]any{
		"url":       srv.URL,
		"token":     "wrong",
		"statesURL": srv.URL + "/api/states",
	})
	sh.client = srv.Client()

	if err := sh.fetch(); err == nil {
		t.Fatal("expected error for unauthorized request")
	}
}

func TestIsOn(t *testing.T) {
	tests := []struct {
		domain, state string
		want          bool
	}{
		{"light", "on", true},
		{"light", "off", false},
		{"switch", "ON", true},
		{"switch", "OFF", false},
		{"lock", "unlocked", true},
		{"lock", "locked", false},
		{"cover", "open", true},
		{"cover", "closed", false},
		{"climate", "heat", true},
		{"climate", "off", false},
		{"sensor", "22.5", true},
		{"sensor", "unavailable", false},
	}
	for _, tt := range tests {
		if got := isOn(tt.domain, tt.state); got != tt.want {
			t.Errorf("isOn(%q, %q) = %v, want %v", tt.domain, tt.state, got, tt.want)
		}
	}
}

func TestFilterEntitiesWithList(t *testing.T) {
	sh := &SmartHome{entityIDs: []string{"light.kitchen"}}
	all := []entity{
		{EntityID: "light.kitchen", State: "on", FriendlyName: "Kitchen"},
		{EntityID: "light.bedroom", State: "on", FriendlyName: "Bedroom"},
		{EntityID: "switch.living", State: "on", FriendlyName: "Living"},
	}
	filtered := sh.filterEntities(all)
	if len(filtered) != 1 {
		t.Fatalf("expected 1, got %d", len(filtered))
	}
	if filtered[0].name != "Kitchen" {
		t.Errorf("expected Kitchen, got %s", filtered[0].name)
	}
}

func TestFormatState(t *testing.T) {
	if got := formatState("climate", "heat", map[string]any{"temperature": 21.5}); got != "22°C" {
		t.Errorf("expected 22°C, got %s", got)
	}
	if got := formatState("light", "on", map[string]any{"brightness": 127.5}); got != "50%" {
		t.Errorf("expected 50%%, got %s", got)
	}
	if got := formatState("sensor", "22.5", map[string]any{"unit_of_measurement": "°C"}); got != "22.5°C" {
		t.Errorf("expected 22.5°C, got %s", got)
	}
}
