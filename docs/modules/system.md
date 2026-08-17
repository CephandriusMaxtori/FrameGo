# System

CPU, memory, and disk utilization of the host machine. Stats are sampled on a background ticker using [gopsutil](https://github.com/shirou/gopsutil).

## Options

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `interval` | duration | `5` | Seconds between system samples |
| `diskPath` | text | `/` | Disk partition to monitor |
| `showCPU` | boolean | `true` | Show CPU utilization |
| `showMem` | boolean | `true` | Show memory utilization |
| `showDisk` | boolean | `true` | Show disk utilization |
| `cpuColor` | color | `#57c7ac` | CPU bar color |
| `memColor` | color | `#57a0dc` | Memory bar color |
| `diskColor` | color | `#dcbe64` | Disk bar color |
| `labelColor` | color | `#9aa7b8` | Label text color |
| `barColor` | color | `#26303e` | Progress bar track color |

## Example

```json
{
  "name": "system",
  "zone": "lower-left",
  "visible": true,
  "options": {
    "interval": 10,
    "showCPU": true,
    "showMem": true,
    "showDisk": false,
    "diskPath": "/dev/sda1"
  }
}
```

## Display

Each metric is shown as a row with:

- **Label** — CPU, MEM, or DISK
- **Progress bar** — color-filled proportional to utilization
- **Detail text** — percentage and absolute values (e.g., "45%  2.1G / 4.0G")

On Windows, disk stats may show different partition names. Use `diskPath` to specify the correct one.
