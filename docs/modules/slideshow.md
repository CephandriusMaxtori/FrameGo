# Slideshow

Rotating display of images from a local directory. Supports PNG, JPEG, GIF, and WebP. No network access.

## Options

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `dir` | file | **required** | Path to directory containing images |
| `interval` | duration | `15` | Seconds between image transitions |
| `fit` | select | `contain` | `contain` (fit within bounds) or `cover` (fill and crop) |

## Example

```json
{
  "name": "slideshow",
  "zone": "middle-center",
  "visible": true,
  "options": {
    "dir": "/home/user/photos",
    "interval": 10,
    "fit": "cover"
  }
}
```

## Supported Formats

- PNG
- JPEG/JPG
- GIF
- WebP (via `golang.org/x/image/webp`)

## Fit Modes

| Mode | Description |
|------|-------------|
| `contain` | Image fits entirely within the zone, may leave letterbox bars |
| `cover` | Image fills the zone, excess is cropped from edges |

## Notes

- Images are sorted alphabetically by filename
- The current image index is derived from the clock (`now.Unix() / interval`)
- Images are decoded once and cached in memory
- If the directory is empty or missing, shows "no images" placeholder
