# Architecture

FrameGo is a single-binary Go application that renders widget modules onto an RGBA canvas. This document describes how the pieces fit together.

## High-Level Flow

```
┌──────────┐    ┌──────────┐    ┌──────────┐    ┌──────────┐
│  Config   │───▶│  Engine   │───▶│  Canvas  │───▶│ Backend  │
│  Loader   │    │  + Bus   │    │ (raster) │    │ (fb/png) │
└──────────┘    └────┬─────┘    └──────────┘    └──────────┘
                     │
              ┌──────┴──────┐
              │  Supervisor  │
              │ (per-module) │
              └──────┬──────┘
                     │
          ┌──────────┼──────────┐
          ▼          ▼          ▼
      ┌────────┐ ┌────────┐ ┌────────┐
      │Module 1│ │Module 2│ │Module N│
      │(clock) │ │(weather│ │(system)│
      └────────┘ └────────┘ └────────┘
```

## Core Packages

| Package | Purpose |
|---------|---------|
| `config/` | Config types, validation, JSON/YAML loading and saving |
| `engine/` | `Engine` (frame loop), `Bus` (async pub/sub), `Supervisor` (fault isolation), `Logger` |
| `render/` | `Canvas` rasterizer, `Backend` interface, PNG and Linux framebuffer backends |
| `fonts/` | Embedded Go font faces, scaling helpers |
| `layout/` | 18-zone grid solver and stacking |
| `modules/` | Built-in module implementations and registry |
| `input/` | Linux evdev touch input reader |
| `presets/` | Built-in configuration presets |
| `web/` | Admin Studio UI (embedded) and REST API |

## Frame Loop

The engine runs a simple frame loop:

1. Check if it's time for the next frame (based on `fps`)
2. Create a new `Canvas` with the display dimensions
3. Fill with the background color
4. For each active module in zone order:
   a. Compute the zone bounds from the layout solver
   b. Call `module.Draw(canvas, bounds, now)`
5. Send the canvas to the backend (`fb` writes to `/dev/fb0`, `png` encodes to a file)
6. Publish a `system:clock:tick` event on the bus

## Module Lifecycle

```
Configure(opts) ──▶ Start(bus, log) ──▶ Draw() ──▶ ... ──▶ Stop()
     │                    │                 ▲                  │
     │                    │                 │                  │
     │                    └─────────────────┘ (each frame)    │
     │                                                       │
     └────────── on reload: Stop() ──▶ Configure() ──▶ Start() ─┘
```

- **Configure** runs before registration and on hot reload. It must never panic.
- **Start** begins background work. Every goroutine needs a `done chan` and a `WaitGroup`.
- **Draw** is called at the display frame rate. It must never block on I/O.
- **Stop** halts background work.

### Supervisor

Each module runs under a `Supervisor` that:

- Recovers panics and marks the module as `faulted`
- Renders a degraded placeholder for faulted modules
- Allows the frame loop to continue even when one module fails

## Event Bus

`engine.Bus` is an asynchronous publish/subscribe router. Standard topics:

| Topic | Published When |
|-------|----------------|
| `system:clock:tick` | Every frame |
| `system:power:dim` | Display dimming |
| `system:network:status` | Network connectivity change |
| `module:state:change` | Module status transition |
| `input:touch` | Touch event from evdev |

Subscribers get buffered channels. Publishes are non-blocking — a slow subscriber can't stall the frame loop.

## Render Primitives

The `render.Canvas` provides:

- `Fill` / `FillRect` / `FillRoundRect` / `FillCircle` — shape primitives
- `DividerH` / `DividerV` — horizontal and vertical rule lines
- `StatusDot` — colored status indicator
- `DrawText` / `DrawTextCentered` / `DrawTextBlock` / `DrawLabel` — text rendering
- `WrapText` / `TextSize` / `FaceMetrics` / `Ascent` — text measurement
- `DrawImageFit` — bilinear `contain`/`cover` image scaling

Masks for rounded rects and circles are cached for performance.

## Layout System

The 18-zone grid divides the display canvas:

```
┌─────────────────────────────────────┐
│ top-left  │ top-center │ top-right  │   each column: 1/3 width
├───────────┼────────────┼────────────┤   upper/middle/lower: 1/3 height
│ upper-left│ middle     │ upper-right│
├───────────┤ center     ├────────────┤   bar zones: full width, 1/6 height
│ lower-left│            │ lower-right│
├───────────┼────────────┼────────────┤   bottom-left/center/right: 1/3 width
│bot-left   │bot-center  │ bot-right  │
├───────────┴────────────┴────────────┤
│           top-bar (full width)      │
├─────────────────────────────────────┤
│         bottom-bar (full width)     │
└─────────────────────────────────────┘
```

Multiple modules in the same column zone stack vertically with configurable gap.

## Font System

Fonts are embedded in the binary via Go's `embed` package:

- `Inter-Regular` (weight 400)
- `Inter-Medium` (weight 500)
- `Inter-Bold` (weight 600)

`fonts.Scaled(bounds, base, weight)` scales font size proportionally to zone height, using 480px as the reference. This means fonts automatically adapt to any display size.
