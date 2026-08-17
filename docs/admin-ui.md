# Admin UI

The Admin Studio is a web-based configuration interface served at the root of the admin server.

## Access

Open `http://<host>:8080` in your browser. If an admin token is configured, you'll need to include it in the request header or URL.

## Features

### Zone Layout Canvas

The left side shows a visual representation of your display with all 18 zones. Modules are shown as chips that can be:

- **Dragged** between zones
- **Clicked** to open configuration
- **Reordered** within a zone using arrows

### Module Registry

The right sidebar lists all configured modules. For each module:

- **Toggle visibility** — show/hide without removing
- **Change zone** — move to a different layout zone
- **Configure** — open the options editor
- **Remove** — delete from the configuration

### Form-Based Configuration

When a module has a schema, the options editor shows proper form fields:

- **Text inputs** — for strings and paths
- **Number inputs** — with min/max validation
- **Color pickers** — with hex input
- **Select dropdowns** — for enumerated options
- **Boolean toggles** — on/off switches
- **Duration inputs** — with seconds label

Switch to **JSON mode** for raw option editing.

### Presets

The **Presets** dropdown in the header provides one-click configuration templates:

- Minimal, Clock & Date, Weather, Smart Mirror, Info Dashboard, Photo Frame

Applying a preset replaces the entire configuration.

### Live Preview

The preview card shows the current frame, auto-refreshing every 2.5 seconds.

### Module Status

The telemetry card shows real-time status of all modules:

- 🟢 **active** — module is running normally
- 🔴 **faulted** — module panicked or failed
- ⚫ **suspended** — module is throttled

### Save & Reload

- **Save configuration** — writes to disk and hot-reloads
- **Reload from disk** — re-reads the config file (useful for manual edits)

## Keyboard Shortcuts

| Key | Action |
|-----|--------|
| `Ctrl+S` | Save (browser default, not intercepted) |
| `Escape` | Close expanded module editor |
