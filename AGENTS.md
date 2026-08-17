# AGENTS.md

Guidance for AI coding agents and maintainers working in this repository.
FrameGo is a pure-Go smart-mirror / kiosk engine: modules render onto an RGBA
canvas via a zero-dependency rasterizer, with an optional admin web UI.

## Repository layout

| Path                 | Purpose                                              |
|----------------------|------------------------------------------------------|
| `main.go`            | Entrypoint: flags, backend selection, engine wiring  |
| `config/`            | `Config` types, validation, JSON/YAML load & save    |
| `engine/`            | `Engine`, event `Bus`, `Supervisor` (fault isolation), `Logger`, `Module` interface |
| `layout/`            | 12-zone grid solver and stacking                     |
| `render/`            | `Canvas` rasterizer: text, rects, circles, images    |
| `fonts/`             | Embedded Go font faces (Regular/Medium/Bold)         |
| `modules/`           | Widgets; `registry.go`; shared option helpers in `opt/` |
| `web/`               | Embedded admin UI + REST API                         |
| `tools/snapshot/`    | Offline frame renderer for CI/preview                |
| `examples/`          | Per-module and offline config samples                |

## Build / test / verify

```sh
go build ./...
go vet ./...
go test ./...
go run ./tools/snapshot -config examples/offline.json -out frame.png
```

All tests run offline — network modules must be tested against `httptest`
servers with injectable base URLs, never the real network. Always run the
full `go build ./...`, `go vet ./...`, and `go test ./...` after changes.

## Conventions

- **Doc comments only.** Do not add explanatory inline comments to code unless
  explicitly asked. Existing files use package + exported-symbol doc comments.
- Pure Go, no cgo. No external display/browser dependencies.
- Cross-platform code uses the existing `backend_linux.go` / `backend_other.go`
  build-tag pattern when behavior differs by OS.
- Errors are returned and wrapped with `%w`; modules that fail to configure
  return an error from `Configure` (surfaced at startup/reload), while
  transient runtime failures (network, sampling) are logged and rendered as a
  degraded placeholder, never fatal.

## Module contract

`engine.Module` (see `engine/module.go`) requires:

```go
Name() string
Configure(opts map[string]any) error
Start(bus *engine.Bus, log *engine.Logger) error
Stop() error
Draw(c *render.Canvas, bounds image.Rectangle, now time.Time) error
```

Lifecycle rules enforced by `engine/supervisor.go`:

- `Configure` runs before registration and on hot reload. **Never panic.**
- `Start` begins background work (goroutines, subscriptions); `Stop` must halt
  it. Every started goroutine needs a `done chan struct{}` and a `WaitGroup`.
- `Draw` is called at the display frame rate. **It must never block on I/O.**
  Network/data-fetching modules keep a mutex-guarded snapshot refreshed by the
  background loop and render that snapshot.
- The supervisor recovers panics per module and marks the module `faulted`
  (renders a degraded placeholder) while the frame loop continues.

### Adding a module

1. `modules/<name>/<name>.go` — package implementing `engine.Module`.
   Register in `init()`:
   ```go
   func init() { modules.Register("name", New) }
   ```
2. Blank-import it in **both** `main.go` and `tools/snapshot/main.go`
   (`_ "framego/modules/name"`). Without these imports the registry won't know
   the module and startup fails with `unknown module`.
3. Read options with the `framego/modules/opt` helpers (`opt.Str`, `opt.Int`,
   `opt.Float`, `opt.Bool`, `opt.Color`, `opt.Duration`) — options arrive as
   `map[string]any` from JSON/YAML.
4. Add `modules/<name>/<name>_test.go`:
   - `Configure` defaults and error cases,
   - `Draw` smoke test asserting pixels were written (sample the canvas alpha;
     see `modules/clock` for the pattern),
   - network modules: `httptest` server + overridable `geoURL`/`forecastURL`/
     `url` fields on the module (see `modules/weather`, `modules/calendar`).
5. Update `README.md` (module table + option table) and, if the module changes
   config surface, `config.json.example` / an `examples/*.json`.

### Stateless vs. background modules

Prefer stateless: derive content purely from the `now` argument
(`date`, `moon`, `quote`, `slideshow` index by `now.Unix()/interval`). Only
use a background goroutine when real work is needed (`system` sampling,
`weather`, `calendar` fetching). Rendering must not depend on network state.

## Event bus

`engine.Bus` is an asynchronous publish/subscribe router. Standard topics:
`system:clock:tick`, `system:power:dim`, `system:network:status`,
`module:state:change`. Subscribers get buffered channels; publishes are
non-blocking so a slow subscriber can't stall the frame loop. Prefer the bus
for cross-module communication over direct coupling.

## Render primitives

`render.Canvas` provides: `Fill`, `FillRect`, `FillRoundRect`, `FillCircle`,
`DividerH`, `DividerV`, `StatusDot`, `WarningIcon`, `DrawText`,
`DrawTextCentered`, `DrawTextBlock`, `DrawLabel`, `WrapText`, `TextSize`,
`FaceMetrics`, `Ascent`, and `DrawImageFit` (bilinear `contain`/`cover`
scaling). Masks (rounded rects/circles) are cached. Add primitives only when
a module genuinely needs them, with tests.

## Config surface

`config.Config` (`display`, `admin`, `modules[]`). `Display` drives the layout
solver and canvas size. `Module` has `name`, `zone` (see `layout/zones.go`),
`visible`, and free-form `options`. `Validate()` rejects empty display
dimensions and duplicate/nameless/zoneless modules. Keep example configs valid
so `-snapshot` and the admin UI keep working.
