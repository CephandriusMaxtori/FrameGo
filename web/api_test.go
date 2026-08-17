package web

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"framego/config"
	"framego/engine"
	"framego/modules"
	_ "framego/modules/clock"
)

func testServer(t *testing.T) *Server {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "cfg.json")
	cfg := config.Default()
	cfg.Display.Width, cfg.Display.Height = 320, 200
	cfg.Modules = []config.Module{{Name: "clock", Zone: "top-center", Visible: true}}
	cfg.Admin.Token = "sekret"
	if err := cfg.Save(p); err != nil {
		t.Fatal(err)
	}
	logr := engine.NewLogger(nil)
	eng, err := engine.New(cfg, nil, logr, modules.Create)
	if err != nil {
		t.Fatal(err)
	}
	eng.Start()
	t.Cleanup(eng.Stop)
	return New(eng, p, logr)
}

func request(t *testing.T, s *Server, method, path, body, token string) *httptest.ResponseRecorder {
	t.Helper()
	var rd *bytes.Reader
	if body == "" {
		rd = bytes.NewReader(nil)
	} else {
		rd = bytes.NewReader([]byte(body))
	}
	req := httptest.NewRequest(method, path, rd)
	if token != "" {
		req.Header.Set("X-FrameGo-Token", token)
	}
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)
	return rr
}

func TestAuthRequired(t *testing.T) {
	s := testServer(t)
	if rr := request(t, s, http.MethodGet, "/api/config", "", ""); rr.Code != http.StatusUnauthorized {
		t.Errorf("no token -> %d, want 401", rr.Code)
	}
	if rr := request(t, s, http.MethodGet, "/api/config", "", "wrong"); rr.Code != http.StatusUnauthorized {
		t.Errorf("bad token -> %d, want 401", rr.Code)
	}
	if rr := request(t, s, http.MethodGet, "/api/config", "", "sekret"); rr.Code != http.StatusOK {
		t.Errorf("good token -> %d, want 200", rr.Code)
	}
}

func TestGetConfig(t *testing.T) {
	s := testServer(t)
	rr := request(t, s, http.MethodGet, "/api/config", "", "sekret")
	if rr.Code != http.StatusOK {
		t.Fatalf("status %d", rr.Code)
	}
	var cfg config.Config
	if err := json.Unmarshal(rr.Body.Bytes(), &cfg); err != nil {
		t.Fatal(err)
	}
	if len(cfg.Modules) != 1 || cfg.Modules[0].Name != "clock" {
		t.Errorf("modules = %+v", cfg.Modules)
	}
}

func TestPutConfigValid(t *testing.T) {
	s := testServer(t)
	body := `{"display":{"width":400,"height":300},"admin":{"enabled":true,"bind":"0.0.0.0:9999","token":"sekret"},"modules":[{"name":"clock","zone":"middle-center","visible":true}]}`
	rr := request(t, s, http.MethodPut, "/api/config", body, "sekret")
	if rr.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rr.Code, rr.Body.String())
	}
	got := s.engine.Config()
	if got.Display.Width != 400 || got.Modules[0].Zone != "middle-center" {
		t.Errorf("engine not reloaded: %+v", got)
	}
}

func TestPutConfigInvalid(t *testing.T) {
	s := testServer(t)
	rr := request(t, s, http.MethodPut, "/api/config", `{"modules":[]}`, "sekret")
	if rr.Code != http.StatusBadRequest {
		t.Errorf("empty modules -> %d, want 400", rr.Code)
	}
	rr = request(t, s, http.MethodPut, "/api/config", `{not json`, "sekret")
	if rr.Code != http.StatusBadRequest {
		t.Errorf("bad json -> %d, want 400", rr.Code)
	}
}

func TestZones(t *testing.T) {
	s := testServer(t)
	rr := request(t, s, http.MethodGet, "/api/zones", "", "sekret")
	if rr.Code != http.StatusOK {
		t.Fatalf("status %d", rr.Code)
	}
	var resp struct {
		Width int `json:"width"`
		Zones []struct {
			ID string `json:"id"`
			W  int    `json:"w"`
			H  int    `json:"h"`
		} `json:"zones"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Width != 320 {
		t.Errorf("width = %d", resp.Width)
	}
	if len(resp.Zones) < 10 {
		t.Errorf("zones = %d", len(resp.Zones))
	}
}

func TestSnapshot(t *testing.T) {
	s := testServer(t)
	rr := request(t, s, http.MethodGet, "/api/snapshot", "", "sekret")
	if rr.Code != http.StatusOK {
		t.Fatalf("status %d", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); !strings.HasPrefix(ct, "image/png") {
		t.Errorf("content-type = %q", ct)
	}
	if rr.Body.Len() == 0 {
		t.Error("empty snapshot")
	}
}

func TestStatus(t *testing.T) {
	s := testServer(t)
	rr := request(t, s, http.MethodGet, "/api/status", "", "sekret")
	if rr.Code != http.StatusOK {
		t.Fatalf("status %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "clock") {
		t.Errorf("status body = %s", rr.Body.String())
	}
}

func TestIndex(t *testing.T) {
	s := testServer(t)
	rr := request(t, s, http.MethodGet, "/", "", "sekret")
	if rr.Code != http.StatusOK {
		t.Fatalf("status %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "<html") {
		t.Error("index does not contain html")
	}
}

func TestKiosk(t *testing.T) {
	s := testServer(t)
	rr := request(t, s, http.MethodGet, "/kiosk", "", "sekret")
	if rr.Code != http.StatusOK {
		t.Fatalf("status %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "/api/snapshot") {
		t.Error("kiosk does not reference the snapshot endpoint")
	}
}

func TestReload(t *testing.T) {
	s := testServer(t)
	rr := request(t, s, http.MethodPost, "/api/reload", "", "sekret")
	if rr.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rr.Code, rr.Body.String())
	}
}
