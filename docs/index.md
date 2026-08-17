# FrameGo

**A pure-Go smart mirror and kiosk engine for the Raspberry Pi and beyond.**

FrameGo renders configurable widget modules onto an RGBA canvas with a zero-dependency rasterizer. It supports Linux framebuffer output for headless displays and a browser-based kiosk mode with full touchscreen support.

## Features

- **Pure Go** — no cgo, no browser engine, no external dependencies
- **9 built-in modules** — clock, date, weather, calendar, system stats, moon phase, quotes, slideshow, NFL scores
- **Touchscreen kiosk mode** — browser-based fullscreen display with touch overlay controls
- **Admin Studio UI** — form-based module configuration with live preview
- **Presets** — one-click setup for common smart mirror layouts
- **Proportional fonts** — text scales automatically to any display size
- **Event bus** — async pub/sub for cross-module communication
- **Hot reload** — change config and see results instantly
- **CI/CD** — GitHub Actions builds for Windows and Linux ARM64

## Quick Start

```bash
# Clone and build
git clone https://github.com/CephandriusMaxtori/FrameGo.git
cd FrameGo
go build -o framego .

# Run with default config
./framego

# Or with a specific config
./framego -config config.json
```

Open `http://localhost:8080` to access the Admin Studio UI.

## Hardware

FrameGo is designed for:

- **Raspberry Pi** (any model) with a 7-inch touchscreen display
- **Linux SBCs** (Orange Pi, Banana Pi, etc.) with framebuffer output
- **Desktop** (Windows/macOS/Linux) for development and preview
- **Any system** with a web browser for kiosk mode

## Documentation

- [Getting Started](getting-started.md) — installation and first run
- [Configuration](configuration.md) — config file reference
- [Modules](modules/index.md) — all built-in modules and their options
- [Kiosk Mode](kiosk.md) — touchscreen and browser kiosk setup
- [Admin UI](admin-ui.md) — the web-based configuration interface
- [REST API](api.md) — programmatic control
- [Architecture](architecture.md) — how FrameGo works internally
- [Contributing](contributing.md) — development guide

## License

MIT
