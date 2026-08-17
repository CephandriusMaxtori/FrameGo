package web

import (
	"encoding/json"
	"image/png"
	"io"
	"net/http"

	"framego/config"
	"framego/engine"
	"framego/layout"
	"framego/modules"
)

// auth gates requests with the configured admin token. When no token is set,
// the API is open. The token is read live from the engine so hot reloads of
// the admin section take effect immediately.
func (s *Server) auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := s.engine.Config().Admin.Token
		if token != "" && r.Header.Get("X-FrameGo-Token") != token && r.URL.Query().Get("token") != token {
			writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "unauthorized"})
			return
		}
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]any{"error": msg})
}

func (s *Server) handleGetConfig(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.engine.Config())
}

func (s *Server) handlePutConfig(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "read body: "+err.Error())
		return
	}
	cfg, err := config.Parse(body, ".json")
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid config: "+err.Error())
		return
	}
	if err := cfg.Validate(); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := cfg.Save(s.configPath); err != nil {
		writeErr(w, http.StatusInternalServerError, "save config: "+err.Error())
		return
	}
	if err := s.engine.Reload(cfg); err != nil {
		writeErr(w, http.StatusInternalServerError, "reload: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "config": cfg})
}

type zoneInfo struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	X     int    `json:"x"`
	Y     int    `json:"y"`
	W     int    `json:"w"`
	H     int    `json:"h"`
}

func (s *Server) handleZones(w http.ResponseWriter, r *http.Request) {
	cfg := s.engine.Config()
	solver := layout.NewSolver(cfg.Display)
	rects := solver.Resolve()
	zones := make([]zoneInfo, 0, len(rects))
	for _, z := range layout.AllZones {
		rc := rects[z]
		zones = append(zones, zoneInfo{
			ID: string(z), Label: string(z),
			X: rc.Min.X, Y: rc.Min.Y, W: rc.Dx(), H: rc.Dy(),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"width":       cfg.Display.Width,
		"height":      cfg.Display.Height,
		"zones":       zones,
		"moduleTypes": modules.Names(),
	})
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.engine.Status())
}

func (s *Server) handleSnapshot(w http.ResponseWriter, r *http.Request) {
	img := s.engine.RenderFrame()
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "no-store")
	if err := png.Encode(w, img); err != nil {
		s.log.Errorf("snapshot encode: %v", err)
	}
}

func (s *Server) handleReload(w http.ResponseWriter, r *http.Request) {
	cfg, err := s.load()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "reload config: "+err.Error())
		return
	}
	if err := s.engine.Reload(cfg); err != nil {
		writeErr(w, http.StatusInternalServerError, "reload: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleSchemas(w http.ResponseWriter, r *http.Request) {
	schemas := map[string]*engine.Schema{}
	for _, name := range modules.Names() {
		mod, ok := modules.Create(name)
		if !ok {
			continue
		}
		if sm, ok := mod.(engine.SchemaModule); ok {
			schemas[name] = sm.Schema()
		}
	}
	writeJSON(w, http.StatusOK, schemas)
}
