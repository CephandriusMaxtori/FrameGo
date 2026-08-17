# Quote

Rotating quotation display. Quotes rotate on a configurable cadence. Can source from a local file or the embedded default list. No network access.

## Options

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `interval` | duration | `86400` | Seconds between quote rotations (24h) |
| `textColor` | color | `#9aa7b8` | Quote text color |
| `prefix` | text | *(empty)* | Text prepended to each quote |
| `file` | file | *(empty)* | Path to quotes file (blank = built-in) |

## Example

```json
{
  "name": "quote",
  "zone": "middle-center",
  "visible": true,
  "options": {
    "interval": 43200,
    "prefix": "Quote of the day:",
    "textColor": "#cbd5e1"
  }
}
```

## Quotes File Format

The file can be either:

**JSON array:**
```json
[
  "The only way to do great work is to love what you do. — Steve Jobs",
  "Simplicity is the ultimate sophistication. — Leonardo da Vinci"
]
```

**Plain text (one per line):**
```
The only way to do great work is to love what you do. — Steve Jobs
Simplicity is the ultimate sophistication. — Leonardo da Vinci
```

## Built-in Quotes

If no file is configured, FrameGo uses 8 inspirational quotes from figures like Steve Jobs, Albert Einstein, and Confucius.
