# Weather

Current conditions and short forecast powered by the [Open-Meteo API](https://open-meteo.com/) — **no API key required**. Data is fetched on a background ticker; Draw renders the latest snapshot without blocking on the network.

## Options

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `city` | text | **required** | City name for geocoding lookup |
| `units` | select | `metric` | `metric` or `imperial` |
| `forecastDays` | number | `5` | Days of forecast (1–7) |
| `update` | duration | `600` | Seconds between API refreshes |
| `tempColor` | color | `#f5f7fa` | Temperature text color |
| `condColor` | color | `#9aa7b8` | Condition text color |
| `hiLoColor` | color | `#9aa7b8` | Hi/Lo text color |
| `locColor` | color | `#9aa7b8` | Location text color |
| `dayColor` | color | `#9aa7b8` | Forecast day label color |

## Example

```json
{
  "name": "weather",
  "zone": "middle-center",
  "visible": true,
  "options": {
    "city": "New York",
    "units": "imperial",
    "forecastDays": 3,
    "update": 300
  }
}
```

## Display

The module shows:

1. **Location name** — city from geocoding
2. **Temperature** — current temp with degree symbol
3. **Conditions** — weather description, feels-like, humidity, wind
4. **Hi/Lo** — daily high and low
5. **Forecast strip** — per-day columns at the bottom (if space permits)

## Notes

- Uses Open-Meteo's free API — no API key needed
- Geocoding happens once on startup
- Background refresh every `update` seconds
- Graceful degradation: shows "collecting..." before first fetch, "unavailable" on error
