# Getting Started

## Prerequisites

- **Go 1.22+** installed
- **Git** for cloning the repository
- Optional: a Linux system with `/dev/fb0` for framebuffer output

## Build

```bash
git clone https://github.com/CephandriusMaxtori/FrameGo.git
cd FrameGo
go build -o framego .
```

On Windows, the binary will be `framego.exe`.

## Run

```bash
./framego
```

On first run, FrameGo copies `config.json.example` to `config.json` if no config exists. The engine starts rendering frames and, if enabled, launches the admin web server.

## First Steps

1. Open `http://localhost:8080` in your browser
2. You'll see the **Admin Studio** with a live preview
3. Use the **Presets** dropdown to try a quick layout
4. Click **Configure** on any module to edit its options
5. Click **Save configuration** to apply changes

## Command Line Flags

```
-config string    Path to config.json or config.yaml (auto-detected if empty)
-backend string   Display backend: auto|png|fb (default "auto")
-out string       Output path for png backend / snapshot (default "frame.png")
-fb string        Linux framebuffer device (default "/dev/fb0")
-snapshot         Render a single frame to -out and exit
```

## Backends

| Backend | Description | Platform |
|---------|-------------|----------|
| `png` | Renders to a PNG file | All |
| `fb` | Writes to Linux framebuffer (`/dev/fb0`) | Linux amd64/arm64 |
| `auto` | Uses `fb` on Linux, `png` elsewhere | All |

## Snapshot Mode

Render a single frame without starting the full engine:

```bash
./framego -snapshot -config config.json -out frame.png
```

This is useful for CI/CD and testing.
