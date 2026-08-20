# Todo

## Done

- [x] Fix touchscreen: wire evdev into main.go with --touch flag and lifecycle
- [x] Support single-touch events (ABS_X/ABS_Y) alongside multitouch (ABS_MT_POSITION_X/Y)
- [x] Support mouse-as-touch for virtual framebuffer setups (EV_REL, BTN_LEFT)
- [x] Add AutoTouchDevice() for auto-detecting touch devices via ioctl
- [x] Add input/evdev_linux_test.go with unit tests
