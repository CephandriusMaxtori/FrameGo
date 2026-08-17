# FrameGo
[![Release](https://github.com/CephandriusMaxtori/FrameGo/actions/workflows/release.yml/badge.svg)](https://github.com/CephandriusMaxtori/FrameGo/actions/workflows/release.yml)

**A pure-Go smart mirror and kiosk engine.**

FrameGo renders configurable widget modules onto an RGBA canvas with a zero-dependency rasterizer. No cgo, no browser engine, no window manager required.

## Quick Start

```bash
git clone https://github.com/CephandriusMaxtori/FrameGo.git
cd FrameGo
go build -o framego .

./framego                          # run with config.json (auto-created on first run)
./framego -snapshot -out frame.png  # render one frame and exit
./framego -backend fb              # Linux framebuffer output
```

Open `http://localhost:8080` for the Admin Studio UI (requires `admin.enabled: true`).

## Features

| | |
|---|---|
| **9 modules** | Clock, Date, Weather, Calendar, System, Moon, Quote, Slideshow, NFL |
| **18-zone grid** | Layout modules into columns with top/bottom full-width bars |
| **Touchscreen kiosk** | Browser-based fullscreen display with touch overlay controls |
| **Admin Studio** | Form-based module configuration with live preview |
| **Presets** | One-click setup for common smart mirror layouts |
| **Hot reload** | Change config and see results instantly — no restart |
| **Fault isolation** | Panicking modules are quarantined; the frame loop continues |
| **Proportional fonts** | Text scales automatically to any display size |
| **CI/CD** | GitHub Actions builds for Windows and Linux ARM64 |

## Documentation

Full documentation is available at the [docs site](https://cephandriusmaxtori.github.io/FrameGo/):

- [Getting Started](https://cephandriusmaxtori.github.io/FrameGo/getting-started/) — installation and first run
- [Configuration](https://cephandriusmaxtori.github.io/FrameGo/configuration/) — config file reference
- [Modules](https://cephandriusmaxtori.github.io/FrameGo/modules/) — all built-in modules and options
- [Kiosk Mode](https://cephandriusmaxtori.github.io/FrameGo/kiosk/) — touchscreen and browser kiosk setup
- [Admin UI](https://cephandriusmaxtori.github.io/FrameGo/admin-ui/) — the web configuration interface
- [REST API](https://cephandriusmaxtori.github.io/FrameGo/api/) — programmatic control
- [Architecture](https://cephandriusmaxtori.github.io/FrameGo/architecture/) — how FrameGo works internally
- [Contributing](https://cephandriusmaxtori.github.io/FrameGo/contributing/) — development guide

## CLI Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-config` | *(auto)* | Path to `config.json` / `config.yaml` |
| `-backend` | `auto` | Display backend: `auto`, `png`, or `fb` |
| `-out` | `frame.png` | Output path for PNG backend / snapshot |
| `-fb` | `/dev/fb0` | Linux framebuffer device |
| `-snapshot` | `false` | Render a single frame to `-out` and exit |

## Tech Stack

- **Go 1.22+** — pure Go, no cgo
- **gopsutil** — system stats (CPU, memory, disk)
- **golang-ical** — ICS calendar parsing
- **golang.org/x/image** — WebP support
- **golang.org/x/sys** — Linux evdev touch input

## License

MIT
