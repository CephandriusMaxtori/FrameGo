# Moon

Current moon phase computed entirely locally (no network). Displays a crescent icon with a text label showing the phase name and illumination percentage.

## Options

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `moonColor` | color | `#f0f4fa` | Lit portion color |
| `shadowColor` | color | `#1a2230` | Shadow portion color |
| `labelColor` | color | `#9aa7b8` | Phase label text color |
| `showPercent` | boolean | `true` | Show illumination percentage |
| `timezone` | text | *(local)* | IANA timezone name |

## Example

```json
{
  "name": "moon",
  "zone": "upper-right",
  "visible": true,
  "options": {
    "showPercent": true,
    "moonColor": "#e2e8f0",
    "shadowColor": "#0f172a"
  }
}
```

## Display

- **Crescent icon** — rendered as two overlapping circles (lit + shadow)
- **Phase label** — e.g., "Waxing Crescent 35%"
- Phase names: New Moon, Waxing Crescent, First Quarter, Waxing Gibbous, Full Moon, Waning Gibbous, Last Quarter, Waning Crescent
