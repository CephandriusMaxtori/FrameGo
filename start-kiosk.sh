#!/bin/bash

# Set the display for Xvfb
export DISPLAY=:99

# Start Xvfb virtual frame buffer in the background
Xvfb :99 -screen 0 1024x600x24 -ac &
Xvfb_PID=$!

# Wait a moment for Xvfb to initialize
sleep 1

# Start the FrameGo backend binary in the background
./framego &
Backend_PID=$!

# Wait for the backend server to spin up on port 3001
sleep 2

# Launch Chromium in kiosk mode
chromium --kiosk --no-first-run --disable-pinch http://localhost:3001 &
Browser_PID=$!

# Keep the script running so systemd tracks it properly
wait $Xvfb_PID $Backend_PID $Browser_PID
