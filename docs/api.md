# REST API

The admin server exposes a REST API for programmatic control. All endpoints require the admin token if configured (via `X-FrameGo-Token` header or `token` query parameter).

## Endpoints

### GET /api/config

Returns the current configuration.

**Response:**
```json
{
  "display": { "width": 800, "height": 480, "fps": 1, ... },
  "admin": { "enabled": true, "bind": "0.0.0.0:8080", "token": "" },
  "modules": [...]
}
```

### PUT /api/config

Replaces the configuration, saves to disk, and hot-reloads.

**Request body:** Full config JSON (same format as `GET /api/config`).

**Response:**
```json
{ "ok": true, "config": { ... } }
```

### GET /api/zones

Returns zone geometry and available module types.

**Response:**
```json
{
  "width": 800,
  "height": 480,
  "zones": [
    { "id": "top-left", "label": "top-left", "x": 0, "y": 0, "w": 267, "h": 160 }
  ],
  "moduleTypes": ["calendar", "clock", "date", "moon", "quote", "slideshow", "system", "weather"]
}
```

### GET /api/schemas

Returns option schemas for all registered modules.

**Response:**
```json
{
  "clock": {
    "name": "clock",
    "description": "Current time and date display",
    "fields": [
      { "key": "format", "label": "Time Format", "kind": "text", "default": "15:04" }
    ]
  }
}
```

### GET /api/status

Returns the status of all running modules.

**Response:**
```json
[
  { "name": "clock", "state": "active", "error": "" },
  { "name": "weather", "state": "active", "error": "" }
]
```

### GET /api/snapshot

Returns the current frame as a PNG image.

**Response:** `image/png`

### GET /api/presets

Returns available preset configurations.

**Response:**
```json
[
  { "name": "Minimal", "description": "Just the clock" },
  { "name": "Weather", "description": "Clock, date, and weather forecast" }
]
```

### POST /api/presets/apply

Applies a preset configuration.

**Request body:**
```json
{ "name": "Smart Mirror" }
```

**Response:**
```json
{ "ok": true, "config": { ... } }
```

### POST /api/reload

Re-reads the config file from disk and hot-reloads.

**Response:**
```json
{ "ok": true }
```

## Examples

### Save a new configuration

```bash
curl -X PUT http://localhost:8080/api/config \
  -H "Content-Type: application/json" \
  -d @config.json
```

### Apply a preset

```bash
curl -X POST http://localhost:8080/api/presets/apply \
  -H "Content-Type: application/json" \
  -d '{"name": "Smart Mirror"}'
```

### Get a snapshot

```bash
curl -o frame.png http://localhost:8080/api/snapshot
```

### Check module status

```bash
curl http://localhost:8080/api/status | jq
```
