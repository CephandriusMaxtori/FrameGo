# Modules

FrameGo includes 9 built-in modules. Each module implements the `engine.Module` interface and renders into a zone on the canvas.

## Module Lifecycle

1. **Configure** — options are applied from the config file or Admin UI
2. **Start** — background work begins (network fetching, sampling)
3. **Draw** — module renders its content into the zone bounds
4. **Stop** — background work halts

## Available Modules

| Module | Type | Description |
|--------|------|-------------|
| [Clock](clock.md) | Stateless | Current time and date |
| [Date](date.md) | Stateless | Standalone day and date readout |
| [Weather](weather.md) | Background | Current conditions and forecast |
| [Calendar](calendar.md) | Background | Upcoming events from ICS feeds |
| [System](system.md) | Background | CPU, memory, and disk stats |
| [Moon](moon.md) | Stateless | Current moon phase |
| [Quote](quote.md) | Stateless | Rotating quotations |
| [Slideshow](slideshow.md) | Stateless | Rotating photo display |

## Font Sizing

All modules use proportional font sizing via `fonts.Scaled()`. Text size scales automatically based on the zone height relative to a 480px reference (optimized for 7-inch displays). This means:

- On a 480px tall zone, fonts render at their base size
- On a 960px tall zone, fonts render at 2× their base size
- On a 240px tall zone, fonts render at 0.5× their base size

## Writing Custom Modules

See [Contributing](../contributing.md) for how to add your own module.
