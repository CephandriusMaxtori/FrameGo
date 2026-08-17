// Command snapshot renders a single FrameGo frame to a PNG for offline
// previewing and CI verification:  go run ./tools/snapshot -config examples/clock.json -out frame.png
package main

import (
	"flag"
	"log"
	"os"

	"framego/config"
	"framego/engine"
	"framego/modules"
	_ "framego/modules/calendar"
	_ "framego/modules/clock"
	_ "framego/modules/date"
	_ "framego/modules/moon"
	_ "framego/modules/nfl"
	_ "framego/modules/quote"
	_ "framego/modules/slideshow"
	_ "framego/modules/system"
	_ "framego/modules/weather"
	"framego/render"
)

func main() {
	var configPath, out string
	flag.StringVar(&configPath, "config", "examples/clock.json", "config file")
	flag.StringVar(&out, "out", "frame.png", "output png path")
	flag.Parse()

	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	logr := engine.NewLogger(os.Stderr)
	eng, err := engine.New(cfg, nil, logr, modules.Create)
	if err != nil {
		log.Fatalf("engine: %v", err)
	}
	eng.Start()
	img := eng.RenderFrame()
	if err := render.NewPNGBackend(out).Present(img); err != nil {
		log.Fatalf("write %s: %v", out, err)
	}
	eng.Stop()
	log.Printf("wrote %s", out)
}
