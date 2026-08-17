// Package web serves the embedded administration UI and its REST API.
package web

import (
	"embed"
	"io/fs"
	"net"
	"net/http"

	"framego/config"
	"framego/engine"
)

//go:embed static
var staticFS embed.FS

var staticSub, _ = fs.Sub(staticFS, "static")

// Server is the config-gated embedded HTTP administration server.
type Server struct {
	engine     *engine.Engine
	configPath string
	log        *engine.Logger
	load       func() (*config.Config, error)

	http *http.Server
	ln   net.Listener
}

// New creates an admin server for the given engine. load re-reads the config
// file from disk (used by POST /api/reload).
func New(eng *engine.Engine, configPath string, log *engine.Logger) *Server {
	return &Server{
		engine:     eng,
		configPath: configPath,
		log:        log,
		load: func() (*config.Config, error) {
			return config.Load(configPath)
		},
	}
}

// Start binds the configured address and serves in a background goroutine.
func (s *Server) Start() error {
	admin := s.engine.Config().Admin
	ln, err := net.Listen("tcp", admin.Bind)
	if err != nil {
		return err
	}
	s.ln = ln
	s.http = &http.Server{Handler: s.Handler()}
	go func() {
		if err := s.http.Serve(ln); err != nil && err != http.ErrServerClosed {
			s.log.Errorf("admin server: %v", err)
		}
	}()
	return nil
}

// Close shuts down the admin server.
func (s *Server) Close() error {
	if s.http != nil {
		return s.http.Close()
	}
	return nil
}

// Handler returns the full route mux wrapped in auth middleware.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	fileHandler := http.FileServerFS(staticSub)
	mux.Handle("GET /static/", http.StripPrefix("/static/", fileHandler))
	mux.HandleFunc("GET /{$}", s.handleIndex)
	mux.HandleFunc("GET /kiosk", s.handleKiosk)
	mux.HandleFunc("GET /api/config", s.handleGetConfig)
	mux.HandleFunc("PUT /api/config", s.handlePutConfig)
	mux.HandleFunc("GET /api/zones", s.handleZones)
	mux.HandleFunc("GET /api/status", s.handleStatus)
	mux.HandleFunc("GET /api/schemas", s.handleSchemas)
	mux.HandleFunc("GET /api/snapshot", s.handleSnapshot)
	mux.HandleFunc("POST /api/reload", s.handleReload)
	return s.auth(mux)
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	data, err := fs.ReadFile(staticSub, "index.html")
	if err != nil {
		http.Error(w, "index.html missing", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(data)
}

// handleKiosk serves the fullscreen live-view page for browser kiosk mode.
func (s *Server) handleKiosk(w http.ResponseWriter, r *http.Request) {
	data, err := fs.ReadFile(staticSub, "kiosk.html")
	if err != nil {
		http.Error(w, "kiosk.html missing", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(data)
}

// URL returns a browser-friendly URL for a bind address, mapping
// wildcard/unspecified hosts to localhost.
func URL(bind string) string {
	host, port, err := net.SplitHostPort(bind)
	if err != nil {
		return "http://" + bind
	}
	switch host {
	case "", "0.0.0.0", "::", "[::]":
		host = "localhost"
	}
	return "http://" + net.JoinHostPort(host, port)
}
