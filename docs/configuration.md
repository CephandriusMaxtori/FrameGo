# Configuration

FrameGo uses a JSON or YAML configuration file. The default is `config.json`.

## Structure

```json
{
  "display": {
    "width": 800,
    "height": 480,
    "fps": 1,
    "margin": 16,
    "gap": 8,
    "background": "#000000"
  },
  "admin": {
    "enabled": true,
    "bind": "0.0.0.0:8080",
    "token": ""
  },
  "modules": [
    {
      "name": "clock",
      "zone": "top-center",
      "visible": true,
      "options": {
        "format": "15:04"
      }
    }
  ]
}
```

## Display

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `width` | int | `800` | Canvas width in pixels |
| `height` | int | `480` | Canvas height in pixels |
| `fps` | int | `1` | Frame rate (frames per second) |
| `margin` | int | `16` | Outer margin in pixels |
| `gap` | int | `8` | Gap between stacked modules |
| `background` | string | `#0b0f14` | Background color (hex) |

## Admin

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `enabled` | bool | `false` | Enable the admin web server |
| `bind` | string | `0.0.0.0:8080` | Listen address |
| `token` | string | `""` | Auth token (blank = no auth) |

## Modules

Each module entry has:

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Module type (e.g., `clock`, `weather`) |
| `zone` | string | Layout zone (see below) |
| `visible` | bool | Whether to render this module |
| `options` | object | Module-specific configuration |

## Zones

FrameGo uses a 18-zone grid layout:

```
┌─────────────────────────────────────┐
│            top-left  top-center  top-right            │
│            ┌──────────────────────┐ │
│ upper-left │    middle-center     │ upper-right       │
│            └──────────────────────┘ │
│ lower-left │                       │ lower-right       │
│            └──────────────────────┘ │
│          bottom-left bottom-center bottom-right       │
├─────────────────────────────────────┤
│         top-bar                     │
├─────────────────────────────────────┤
│         bottom-bar                  │
└─────────────────────────────────────┘
```

**Bar zones** (`top-bar`, `bottom-bar`) span the full width and are ideal for status strips.

**Column zones** stack vertically when multiple modules share a zone.

## Presets

The Admin UI includes quick-start presets:

| Preset | Description |
|--------|-------------|
| Minimal | Just the clock |
| Clock & Date | Clock on top, date below |
| Weather | Clock, date, and weather forecast |
| Smart Mirror | Full smart mirror: clock, weather, calendar, system |
| Info Dashboard | System stats, weather, and calendar |
| Photo Frame | Clock with rotating photo slideshow |

## Hot Reload

Changes made through the Admin UI are saved to disk and hot-reloaded instantly. No restart required.

You can also reload from disk manually:

```bash
curl -X POST http://localhost:8080/api/reload
```
