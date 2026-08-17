# FrameGo
[![Release](https://github.com/CephandriusMaxtori/FrameGo/actions/workflows/release.yml/badge.svg)](https://github.com/CephandriusMaxtori/FrameGo/actions/workflows/release.yml)

A dependency-light smart mirror / kiosk engine written in Go. FrameGo renders a
grid of widget **modules** onto a display (PNG, Linux framebuffer) using a
pure-Go 2D rasterizer — no cgo, no browser, no window manager required.

## Features

- **Module system** — clock, date, moon phase, quotes, photo slideshow,
  weather, system stats, and calendar; add your own in a few lines.
- **12-zone grid layout** with top/bottom full-width bars, reconfigurable live.
- **Fault isolation** — a panicking module is quarantined; the frame loop keeps
  running and renders a degraded placeholder.
- **Hot reload** — save config changes via the embedded admin web UI and the
  engine diffs/rebuilds the module set at runtime.
- **Zero-network modules work offline**; network modules degrade to an
  "unavailable" state without taking the display down.
- Pure-Go: builds and runs on Linux (framebuffer) and Windows (PNG), no CGO.

## Quick start

```sh
go build -o framego .
# no config? one is created from config.json.example
./framego -snapshot            # render one frame to frame.png and exit
./framego -backend png -out frame.png
./framego -backend fb -fb /dev/fb0   # Linux framebuffer display
./framego -config config.json        # run with the admin UI
```

Rendering a snapshot from an example config (no display needed):

```sh
go run ./tools/snapshot -config examples/offline.json -out frame.png
```

### Command-line flags

| Flag       | Default      | Description                                   |
|------------|--------------|-----------------------------------------------|
| `-config`  | (auto)       | Path to `config.json` / `config.yaml` / `.yml` |
| `-backend` | `auto`       | Display backend: `auto`, `png`, or `fb`        |
| `-out`     | `frame.png`  | Output path for the PNG backend / snapshot     |
| `-fb`      | `/dev/fb0`   | Linux framebuffer device                       |
| `-snapshot`| `false`      | Render a single frame to `-out` and exit       |

## Configuration

`config.json` is a single document with `display`, `admin`, and `modules`
sections. YAML is supported too (auto-detected by extension).

```json
{
  "display": { "width": 800, "height": 480, "margin": 16, "gap": 8, "fps": 1, "background": "#0b0f14" },
  "admin":   { "enabled": true, "bind": "0.0.0.0:8080", "token": "" },
  "modules": [
    { "name": "clock", "zone": "middle-center", "visible": true,
      "options": { "format": "15:04", "dateFormat": "Mon, Jan 2" } }
  ]
}
```

- **display.fps** is the refresh rate of the frame loop (1 = 1 frame/second).
- **admin.token** gates the web UI / API (`X-FrameGo-Token` header or `?token=`);
  leave empty only for loopback binds.
- **modules[]** entries are matched to registered module types by `name`; the
  `options` map is decoded by each module (JSON/YAML values).

### Zones

Five horizontal bands with 3 columns plus full-width bars on the top and
bottom bands: `top-left|top-center|top-right|top-bar`, `upper-*`, `middle-*`,
`lower-*`, `bottom-left|bottom-center|bottom-right|bottom-bar`. Multiple
modules in one zone stack vertically.

## Modules

| Module      | Requires network | Options |
|-------------|------------------|---------|
| [`clock`](modules/clock)    | no | `format`, `dateFormat`, `timezone`, `timeColor`, `dateColor` |
| [`date`](modules/date)      | no | `format`, `weekdayFormat`, `timezone`, `dayColor`, `dateColor` |
| [`moon`](modules/moon)      | no | `moonColor`, `shadowColor`, `labelColor`, `showPercent`, `timezone` |
| [`nfl`](modules/nfl)        | yes | `team`, `games`, `update`, colors |
| [`quote`](modules/quote)    | no | `file`, `interval`, `prefix`, `textColor` |
| [`slideshow`](modules/slideshow) | no | `dir`, `interval`, `fit` (`contain`/`cover`) |
| [`system`](modules/system)  | no | `interval`, `diskPath`, `showCPU`/`showMem`/`showDisk`, colors |
| [`weather`](modules/weather)| yes | `city`, `units`, `forecastDays`, `update`, colors |
| [`calendar`](modules/calendar) | yes | `url`, `username`, `password`, `days`, `maxEvents`, `update`, colors |

Full per-module option tables follow.

### clock

Renders the current time with the date beneath it.

| Option      | Default         | Description                     |
|-------------|-----------------|---------------------------------|
| `format`    | `15:04`         | Go time layout                  |
| `dateFormat`| `Mon, Jan 2`    | Go time layout for the date     |
| `timezone`  | system local    | IANA name, e.g. `Europe/Berlin` |
| `timeColor` | `#f5f7fa`       | Time text color                 |
| `dateColor` | `#9aa7b8`       | Date text color                 |

### date

Standalone large date readout (independent of the clock).

| Option          | Default        | Description                       |
|-----------------|----------------|-----------------------------------|
| `format`        | `Jan 2, 2006`  | Go time layout                    |
| `weekdayFormat` | `Monday`       | Go time layout for the weekday    |
| `timezone`      | system local   | IANA name                         |
| `dayColor`      | `#f5f7fa`      | Weekday color                     |
| `dateColor`     | `#9aa7b8`      | Date color                        |

### moon

Astronomical moon phase (computed locally, no network) drawn as a crescent
with the phase name and illumination.

| Option        | Default    | Description            |
|---------------|------------|------------------------|
| `moonColor`   | `#f0f4fa`  | Lit crescent color     |
| `shadowColor` | `#1a2230`  | Terminator/umbra color |
| `labelColor`  | `#9aa7b8`  | Phase label color      |
| `showPercent` | `true`     | Show illumination %    |
| `timezone`    | system local | IANA name            |

### nfl

Live NFL scores and schedules from the public
[ESPN scoreboard API](https://site.api.espn.com/apis/site/v2/sports/football/nfl/scoreboard)
— **no API key required**. Each game shows the matchup, the score once the
game starts, and a right-aligned status (`Final`, `Q3 2:15`, or the kickoff
time). Fetched on a background ticker; the display never blocks on the network.

| Option      | Default | Description                                  |
|-------------|---------|----------------------------------------------|
| `team`      | `""`    | Abbreviation to filter on (e.g. `KC`); blank shows all games |
| `games`     | `4`     | Number of games shown (1–10)                 |
| `update`    | `300`   | Seconds between refreshes                    |
| `teamColor` | `#f5f7fa`| Matchup text color                          |
| `scoreColor`| `#f5f7fa`| Score text color                            |
| `timeColor` | `#9aa7b8`| Status / kickoff text color                 |

### quote

Rotates a quotation. Works out of the box with an embedded list; point `file`
at your own quotes.

| Option      | Default        | Description                                    |
|-------------|----------------|------------------------------------------------|
| `file`      | (embedded)     | Path to a quotes file: plain text (one per line) or JSON array of strings |
| `interval`  | `86400`        | Seconds between quote changes                  |
| `prefix`    | `""`           | Text prefixed before each quote                |
| `textColor` | `#9aa7b8`      | Quote text color                               |

### slideshow

Rotates images from a local directory. Supports PNG, JPEG, GIF, and WebP;
images are scaled to fit (`contain`) or cover (`cover`) the zone.

| Option     | Default     | Description                  |
|------------|-------------|------------------------------|
| `dir`      | *(required)*| Directory to scan for images |
| `interval` | `15`        | Seconds per image            |
| `fit`      | `contain`   | `contain` or `cover`         |

### system

Host CPU / memory / disk utilization with mini progress bars (via
[gopsutil/v3](https://github.com/shirou/gopsutil)).

| Option      | Default  | Description                             |
|-------------|----------|-----------------------------------------|
| `interval`  | `5`      | Seconds between samples                 |
| `diskPath`  | `/`      | Mount point to report disk usage from   |
| `showCPU`   | `true`   | Show the CPU row                        |
| `showMem`   | `true`   | Show the memory row                     |
| `showDisk`  | `true`   | Show the disk row                       |
| `cpuColor`  | `#57c7ac`| CPU bar color                           |
| `memColor`  | `#57a0dc`| Memory bar color                        |
| `diskColor` | `#dcbe64`| Disk bar color                          |
| `labelColor`| `#9aa7b8` | Row label / value color                |
| `barColor`  | `#26303e` | Bar track color                         |

### weather

Current conditions plus a short forecast from
[Open-Meteo](https://open-meteo.com) — **no API key required**. Fetched on a
background ticker; the display never blocks on the network.

| Option         | Default   | Description                          |
|----------------|-----------|--------------------------------------|
| `city`         | *(required)* | City name, e.g. `Berlin`          |
| `units`        | `metric`  | `metric` or `imperial`               |
| `forecastDays` | `5`       | Days of forecast strip (1–7)         |
| `update`       | `600`     | Seconds between refreshes            |
| `tempColor`    | `#f5f7fa` | Temperature text color               |
| `condColor`    | `#9aa7b8` | Condition / detail text color        |
| `hiLoColor`    | `#9aa7b8` | High/low text color                  |
| `locColor`     | `#9aa7b8` | Location label color                 |
| `dayColor`     | `#9aa7b8` | Forecast-day column color            |

### calendar

Upcoming events from an ICS/Google-calendar feed
(via [golang-ical](https://github.com/arran4/golang-ical)). Use a **public
ICS URL** (Google: calendar settings → "Public address in iCal format"), or
private feeds with `username`/`password` basic auth.

| Option      | Default | Description                        |
|-------------|---------|------------------------------------|
| `url`       | *(required)* | ICS feed URL                  |
| `username`  | `""`    | Basic-auth username (optional)     |
| `password`  | `""`    | Basic-auth password (optional)     |
| `days`      | `3`     | Look-ahead window in days          |
| `maxEvents` | `5`     | Maximum events shown               |
| `update`    | `3600`  | Seconds between refreshes          |
| `dateColor` | `#9aa7b8`| Date/time column color             |
| `eventColor`| `#9aa7b8`| Event summary color                |

## Admin web UI

When `admin.enabled` is true the engine serves a management UI
(`http://<bind>/`): drag modules onto zones, toggle visibility, edit module
`options` as JSON, and save — which hot-reloads the running engine.

API surface (all gated by `admin.token` when set):

| Endpoint          | Method | Purpose                                  |
|-------------------|--------|------------------------------------------|
| `/kiosk`          | GET    | Fullscreen live view for browser kiosk mode |
| `/api/config`     | GET    | Current configuration                    |
| `/api/config`     | PUT    | Validate, save, and hot-reload config    |
| `/api/zones`      | GET    | Zone geometry + registered module types  |
| `/api/status`     | GET    | Per-module lifecycle status              |
| `/api/snapshot`   | GET    | Live PNG of the current frame            |
| `/api/reload`     | POST   | Reload config from disk                  |

## Browser kiosk display

Instead of the Linux framebuffer, run a browser in kiosk mode pointed at the
admin server's `/kiosk` page (a black, fullscreen view that auto-refreshes the
live snapshot at `display.fps`). On a Raspberry Pi with Chromium:

```sh
# run framego with admin.enabled=true (bind 0.0.0.0:8080)
./framego -config config.json

# launch the display in a second terminal (add ?token=... if admin.token is set)
chromium-browser --kiosk --noerrdialogs --disable-infobars \
  --autoplay-policy=no-user-gesture-required \
  --check-for-updates-interval=31536000 \
  http://localhost:8080/kiosk
```

For a bare-metal boot-to-display setup, start both under `xinit`/`lightdm`
autologin or a systemd unit, with `admin.bind` kept on loopback and a token set
if the Pi is reachable from the network.

## Writing a module

1. Create `modules/<name>/<name>.go` implementing the `engine.Module`
   interface: `Name()`, `Configure(opts)`, `Start(bus, log)`, `Stop()`,
   `Draw(canvas, bounds, now)`.
2. Register it in `init()`: `modules.Register("name", New)`.
3. Blank-import the package in `main.go` and `tools/snapshot/main.go`.

Modules own their timing and data fetching. Network modules fetch on a
background goroutine (started in `Start`, stopped via a `done` channel in
`Stop`) and expose a mutex-guarded snapshot that `Draw` reads — `Draw` must
never block on I/O. Pure display modules can be stateless and derive their
content from the `now` argument. The supervisor isolates panics; `Configure`
errors are surfaced at startup / reload.

See [`AGENTS.md`](AGENTS.md) for the full authoring guide and conventions.

## Development

```sh
go build ./...      # build everything
go vet ./...        # static checks
go test ./...       # unit tests (all offline, no network)
go run ./tools/snapshot -config examples/offline.json -out frame.png
```

Requires Go 1.26+. Dependencies: `golang.org/x/image`, `gopkg.in/yaml.v3`,
`github.com/shirou/gopsutil/v3`, `github.com/arran4/golang-ical`.
