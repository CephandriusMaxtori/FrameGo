# Smart Home

Displays the status of Home Assistant entities — lights, thermostats, locks, sensors, and more. Connects via the Home Assistant REST API with a long-lived access token. Data is fetched on a background ticker; Draw renders the latest snapshot without blocking on the network.

## Options

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `url` | text | **required** | Home Assistant base URL (e.g., `http://192.168.1.100:8123`) |
| `token` | password | **required** | Long-lived access token |
| `entities` | text | *(all)* | Comma-separated entity IDs to show (blank = all common devices) |
| `maxDevices` | number | `10` | Max devices to display (1–20) |
| `update` | duration | `30` | Seconds between API refreshes |
| `titleColor` | color | `#f5f7fa` | Device name text color |
| `stateColor` | color | `#9aa7b8` | State text color |
| `onColor` | color | `#57c7ac` | Green dot for active devices |
| `offColor` | color | `#64748b` | Gray dot for inactive devices |

## Example

```json
{
  "name": "smarthome",
  "zone": "middle-center",
  "visible": true,
  "options": {
    "url": "http://192.168.1.100:8123",
    "token": "eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9...",
    "entities": "light.kitchen, light.living_room, thermostat.bedroom, lock.front_door",
    "maxDevices": 8,
    "update": 30
  }
}
```

## Getting a Long-Lived Access Token

1. Open Home Assistant in your browser
2. Click your user profile (bottom-left)
3. Scroll to **Long-Lived Access Tokens**
4. Click **Create Token**
5. Give it a name (e.g., `FrameGo`) and copy the token

## Supported Entity Types

| Domain | Display | On/Off Logic |
|--------|---------|--------------|
| `light` | Brightness % or on/off | `on` = on |
| `switch` | on/off | `on` = on |
| `climate` | Temperature value | `off` = off |
| `lock` | locked/unlocked | `unlocked` = on |
| `cover` | open/closed | `open`/`opening` = on |
| `fan` | on/off | `on` = on |
| `media_player` | playing/paused | `playing`/`paused` = on |
| `sensor` | Value + unit | Available = on |
| `binary_sensor` | State | `on`/`open`/`detected` = on |

Other domains are ignored when no entity list is specified.

## Display

The module shows:

1. **Header** — "Home (N)" with the count of displayed devices
2. **Device rows** — each row shows:
   - **Colored dot** — green for active, gray for inactive
   - **Friendly name** — from Home Assistant's `friendly_name` attribute
   - **State** — formatted value (brightness %, temperature, units, etc.)
3. Active devices sort to the top
4. Unknown or unavailable entities are hidden

## Filtering

By default, the module shows all entities from common smart home domains. To show only specific devices, set `entities` to a comma-separated list:

```
light.kitchen, light.living_room, thermostat.bedroom, lock.front_door
```

Entity IDs are shown as-is in Home Assistant → Developer Tools → States.

## Notes

- Uses the [Home Assistant REST API](https://developers.home-assistant.io/docs/api/rest/) — no extra integrations needed
- The token is sent as a Bearer token in the `Authorization` header
- If the HA instance is unreachable, shows "smart home: unavailable"
- On first load, shows "smart home: connecting…" until the first successful fetch
- Background refresh every `update` seconds
