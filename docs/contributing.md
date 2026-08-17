# Contributing

FrameGo welcomes contributions. This guide explains the conventions and workflow.

## Prerequisites

- Go 1.22+
- Git
- (Optional) `mkdocs` + `mkdocs-material` for docs

## Building

```bash
go build -o framego .
```

## Testing

```bash
go build ./...
go vet ./...
go test ./...
```

All tests run offline. Network modules must be tested against `httptest` servers with injectable base URLs — never the real network.

## Repository Layout

```
main.go              Entrypoint: flags, backend selection, engine wiring
config/              Config types, validation, JSON/YAML load & save
engine/              Engine, Bus, Supervisor, Logger, Module interface
layout/              12-zone grid solver and stacking
render/              Canvas rasterizer and backends
fonts/               Embedded font faces and scaling
modules/             Built-in widgets and registry
input/               Linux evdev touch reader
presets/             Built-in configuration presets
web/                 Admin UI (embedded) and REST API
tools/snapshot/      Offline frame renderer
examples/            Sample config files
```

## Module Contract

Every module implements `engine.Module`:

```go
type Module interface {
    Name() string
    Configure(opts map[string]any) error
    Start(bus *Bus, log *Logger) error
    Stop() error
    Draw(c *Canvas, bounds image.Rectangle, now time.Time) error
}
```

### Rules

1. **Configure** must never panic. Return an error for invalid options.
2. **Start** begins background work. Every goroutine needs a `done chan` and a `WaitGroup`.
3. **Draw** must never block on I/O. Use a mutex-guarded snapshot pattern for network modules.
4. **Stop** must halt all background work.

### Adding a New Module

1. Create `modules/<name>/<name>.go`:
   ```go
   package name

   import (
       "framego/engine"
       "framego/modules"
   )

   type mod struct {
       done chan struct{}
       wg   sync.WaitGroup
   }

   func init() { modules.Register("name", New) }

   func New() engine.Module { return &mod{} }
   ```

2. Implement the `Schema()` method (optional, enables form-based UI):
   ```go
   func (m *mod) Schema() engine.Schema {
       return engine.Schema{
           Name:        "name",
           Description: "Module description",
           Fields: []engine.Field{
               {Key: "option", Label: "Option", Kind: engine.FieldText, Default: "value"},
           },
       }
   }
   ```

3. Add a test file `modules/<name>/<name>_test.go`:
   - Test `Configure` defaults and error cases
   - Test `Draw` by sampling canvas pixels (see `modules/clock` for the pattern)
   - Network modules: use `httptest` server + injectable URL field

4. Blank-import it in **both** `main.go` and `tools/snapshot/main.go`:
   ```go
   import _ "framego/modules/name"
   ```

5. Read options with `modules/opt` helpers:
   ```go
   func (m *mod) Configure(opts map[string]any) error {
       m.interval = opt.Duration(opts, "interval", 600)
       m.color = opt.Color(opts, "color", "#ffffff")
       return nil
   }
   ```

## Code Style

- **Doc comments only** — no explanatory inline comments unless asked
- **No cgo** — pure Go, no external display/browser dependencies
- **Cross-platform** — use `backend_linux.go` / `backend_other.go` build tags
- **Error wrapping** — use `fmt.Errorf("context: %w", err)`
- **No secrets in code** — never commit API keys or tokens

## Commit Convention

FrameGo uses short, descriptive commit messages:

```
Add feature name

- What was done
- Why (if non-obvious)
```

## CI/CD

Pushes to `main` run `go build`, `go vet`, and `go test` via GitHub Actions.

Tagged releases (`v*`) build binaries for Windows amd64 and Linux arm64 and publish them to GitHub Releases.
