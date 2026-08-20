// Command framego is the smart mirror / kiosk engine.
package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"framego/config"
	"framego/engine"
	"framego/input"
	"framego/modules"
	_ "framego/modules/calendar"
	_ "framego/modules/clock"
	_ "framego/modules/date"
	_ "framego/modules/moon"
	_ "framego/modules/nfl"
	_ "framego/modules/quote"
	_ "framego/modules/slideshow"
	_ "framego/modules/smarthome"
	_ "framego/modules/system"
	_ "framego/modules/weather"
	"framego/render"
	"framego/web"
)

func main() {
	var configPath, out, fbPath, backendKind, touchPath string
	var snapshot bool
	flag.StringVar(&configPath, "config", "", "path to config.json or config.yaml (auto-detected if empty)")
	flag.StringVar(&backendKind, "backend", "auto", "display backend: auto|png|fb")
	flag.StringVar(&out, "out", "frame.png", "output path for the png backend / snapshot")
	flag.StringVar(&fbPath, "fb", "/dev/fb0", "linux framebuffer device")
	flag.StringVar(&touchPath, "touch", "", "touch input device path (empty=auto-detect, \"none\"=disabled)")
	flag.BoolVar(&snapshot, "snapshot", false, "render a single frame to -out and exit")
	flag.Parse()

	if configPath == "" {
		for _, p := range []string{"config.json", "config.yaml", "config.yml"} {
			if _, err := os.Stat(p); err == nil {
				configPath = p
				break
			}
		}
	}
	if configPath == "" {
		if err := copyTemplate("config.json.example", "config.json"); err != nil {
			fmt.Fprintf(os.Stderr, "no config found and could not create config.json: %v\n", err)
			os.Exit(2)
		}
		fmt.Fprintln(os.Stderr, "no config found; created config.json from config.json.example")
		configPath = "config.json"
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	logr := engine.NewLogger(os.Stdout)

	var backend render.Backend
	var note string
	if !snapshot {
		backend, note, err = openBackend(backendKind, out, fbPath)
		if err != nil {
			log.Fatalf("backend: %v", err)
		}
		if note != "" {
			logr.Printf("%s", note)
		}
	}

	eng, err := engine.New(cfg, backend, logr, modules.Create)
	if err != nil {
		log.Fatalf("engine: %v", err)
	}

	var touchDev *input.Evdev
	if touchPath != "none" && !snapshot {
		if touchPath == "" {
			detected, err := input.AutoTouchDevice(logr)
			if err != nil {
				logr.Printf("touch: %v", err)
			} else {
				touchPath = detected
				if name := input.DeviceName(detected); name != "" {
					logr.Printf("touch: using %s (%s)", detected, name)
				} else {
					logr.Printf("touch: using %s", detected)
				}
			}
		}
		if touchPath != "" && touchPath != "none" {
			dev, err := input.NewEvdev(touchPath, eng.Bus(), logr, cfg.Display.Width, cfg.Display.Height)
			if err != nil {
				logr.Printf("touch: %v", err)
			} else {
				touchDev = dev
				touchDev.Start()
			}
		}
	}

	if snapshot {
		eng.Start()
		img := eng.RenderFrame()
		eng.Stop()
		if err := render.NewPNGBackend(out).Present(img); err != nil {
			log.Fatalf("snapshot: %v", err)
		}
		fmt.Printf("frame rendered to %s\n", out)
		return
	}

	eng.Start()
	defer func() {
		if touchDev != nil {
			touchDev.Stop()
		}
		eng.Stop()
	}()

	if cfg.Admin.Enabled {
		srv := web.New(eng, configPath, logr)
		if err := srv.Start(); err != nil {
			log.Fatalf("admin server: %v", err)
		}
		defer srv.Close()
		if cfg.Admin.Token == "" && !isLocalBind(cfg.Admin.Bind) {
			logr.Printf("warning: admin server bound to %s with no auth token; set admin.token", cfg.Admin.Bind)
		}
		logr.Printf("UI: go to %s", web.URL(cfg.Admin.Bind))
	} else {
		logr.Printf("UI disabled: set admin.enabled to true in %s to open the admin UI", configPath)
	}

	logr.Printf("framego running (config %s)", configPath)
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig
	logr.Printf("shutting down")
}

// copyTemplate copies the template config to the target path if absent.
func copyTemplate(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("read template %q: %w", src, err)
	}
	return os.WriteFile(dst, data, 0o644)
}

// isLocalBind reports whether a bind address is loopback-only.
func isLocalBind(bind string) bool {
	host := bind
	if h, _, err := net.SplitHostPort(bind); err == nil {
		host = h
	}
	switch strings.ToLower(host) {
	case "", "localhost", "127.0.0.1", "::1":
		return true
	}
	return false
}
