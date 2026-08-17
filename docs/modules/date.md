# Date

Standalone day and date readout, independent of the Clock module. Renders the weekday above the full date, centered in bounds.

## Options

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `format` | text | `Jan 2, 2006` | Go date format string |
| `weekdayFormat` | text | `Monday` | Go weekday format string |
| `timezone` | text | *(local)* | IANA timezone name |
| `dayColor` | color | `#f5f7fa` | Weekday text color |
| `dateColor` | color | `#9aa7b8` | Date text color |

## Example

```json
{
  "name": "date",
  "zone": "top-center",
  "visible": true,
  "options": {
    "format": "January 2, 2006",
    "weekdayFormat": "Monday",
    "timezone": "America/New_York"
  }
}
```
