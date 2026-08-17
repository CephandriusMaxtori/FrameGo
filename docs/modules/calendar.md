# Calendar

Upcoming events from a public ICS calendar feed. Supports Google Calendar, Apple Calendar, Outlook, and any standard iCalendar feed.

## Options

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `url` | text | **required** | Public ICS feed URL |
| `username` | text | *(empty)* | Basic auth username (if required) |
| `password` | password | *(empty)* | Basic auth password (if required) |
| `days` | number | `3` | Days ahead to show (1–14) |
| `maxEvents` | number | `5` | Maximum events to display (1–20) |
| `update` | duration | `3600` | Seconds between feed refreshes |
| `dateColor` | color | `#9aa7b8` | Date text color |
| `eventColor` | color | `#9aa7b8` | Event summary text color |

## Example

```json
{
  "name": "calendar",
  "zone": "middle-center",
  "visible": true,
  "options": {
    "url": "https://calendar.google.com/calendar/ical/standard/public.ics",
    "days": 7,
    "maxEvents": 8,
    "update": 1800
  }
}
```

## Getting Your ICS URL

### Google Calendar

1. Open Google Calendar Settings
2. Select your calendar
3. Scroll to "Integrate calendar"
4. Copy the "Public address in iCal format"

### Apple Calendar

1. Open Calendar app
2. File → Share Calendar
3. Enable Public Calendar
4. Copy the URL

## Display

Events are shown as a list with:

- **Date column** — "Today", "Tomorrow", or "Mon 1/2"
- **Time** — start time (omitted for all-day events)
- **Summary** — event title, truncated to fit

Events are sorted by start time and filtered to the configured time window.
