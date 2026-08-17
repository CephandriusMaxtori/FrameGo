# Kiosk Mode

FrameGo supports two kiosk modes: **browser-based** (recommended) and **Linux framebuffer** (headless).

## Browser Kiosk Mode

The browser kiosk mode serves a fullscreen live frame at `/kiosk` with touchscreen-optimized controls.

### Setup

1. Enable the admin server in `config.json`:
   ```json
   {
     "admin": {
       "enabled": true,
       "bind": "0.0.0.0:8080"
     }
   }
   ```

2. On your display device, open a browser to:
   ```
   http://<host>:8080/kiosk
   ```

3. For Chromium kiosk mode (Raspberry Pi):
   ```bash
   chromium-browser --kiosk --noerrdialogs --disable-infobars \
     http://localhost:8080/kiosk
   ```

### Touch Controls

- **Tap anywhere** — opens the control overlay
- **Tap backdrop** — closes the overlay
- **Escape key** — closes the overlay (desktop)
- **Space/Enter** — opens the overlay (desktop)

The overlay panel provides:

- **Open Admin UI** — opens the configuration interface in a new tab
- **Reload Configuration** — re-reads config from disk
- **Sleep Display** — blacks out the frame (tap again to wake)
- **Module Status** — live status of all running modules

The overlay auto-hides after 8 seconds of inactivity.

### Auth

Pass the admin token as a query parameter:
```
http://localhost:8080/kiosk?token=your-secret-token
```

## Linux Framebuffer Mode

For headless displays connected directly to `/dev/fb0`:

```bash
./framego -backend fb -fb /dev/fb0
```

### Touch Input (evdev)

On Linux, FrameGo can read touchscreen input from `/dev/input/event0` via evdev. Touch events are published on the `input:touch` event bus topic for modules to consume.

```bash
# Ensure the user has access to input devices
sudo usermod -a -G input $USER
```

### Auto-Start

Create a systemd service:

```ini
[Unit]
Description=FrameGo Kiosk
After=network.target

[Service]
ExecStart=/usr/local/bin/framego -backend fb -config /etc/framego/config.json
Restart=always
User=pi

[Install]
WantedBy=multi-user.target
```

## Performance Tips

- Set `fps` to `1` for static content (clock, date, weather)
- Set `fps` to `5-10` for animated content (slideshow)
- Higher FPS increases CPU usage — monitor with the System module
- On Raspberry Pi, 1 FPS is usually sufficient for smart mirror use
