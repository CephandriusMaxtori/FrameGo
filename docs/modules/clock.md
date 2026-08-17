# Clock

Renders the current time prominently with the date below it. No background work — renders on demand.

## Options

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `format` | text | `15:04` | Go time format string |
| `dateFormat` | text | `Mon, Jan 2` | Go date format string |
| `timezone` | text | *(local)* | IANA timezone name (e.g., `America/New_York`) |
| `timeColor` | color | `#f5f7fa` | Time text color |
| `dateColor` | color | `#9aa7b8` | Date text color |

## Example

```json
{
  "name": "clock",
  "zone": "top-center",
  "visible": true,
  "options": {
    "format": "3:04 PM",
    "dateFormat": "Monday, January 2",
    "timezone": "America/Chicago",
    "timeColor": "#ffffff",
    "dateColor": "#94a3b8"
  }
}
```

## Go Time Format Reference

| Format | Output |
|--------|--------|
| `15:04` | 14:05 |
| `3:04 PM` | 2:05 PM |
| `03:04:05` | 14:05:09 |
| `Mon, Jan 2` | Wed, Jan 2 |
| `Monday, January 2, 2006` | Wednesday, January 2, 2006 |
