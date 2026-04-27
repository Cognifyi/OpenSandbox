#!/bin/bash
set -e

# Disable Jupyter for browser sandboxes (not needed)
export JUPYTER_HOST=""
export JUPYTER_TOKEN=""

# Start Xvfb for headless browser support (following browserless approach)
# When docker restarts, this file is still there, so we need to kill it just in case
[ -f /tmp/.X99-lock ] && rm -f /tmp/.X99-lock

_kill_procs() {
  kill -TERM $execd_pid
  kill -TERM $xvfb_pid
}

# Relay quit commands to processes
trap _kill_procs SIGTERM SIGINT

if [ -z "$DISPLAY" ]; then
  Xvfb :99 -screen 0 1024x768x16 -nolisten tcp -nolisten unix >/dev/null 2>&1 &
  xvfb_pid=$!
  export DISPLAY=:99
fi

# Check if execd is already running (injected by OpenSandbox server)
if pgrep -x execd > /dev/null; then
    echo "execd is already running, skipping manual start"
else
    # Start execd (for standalone use without OpenSandbox server)
    /usr/local/bin/execd &
    execd_pid=$!
    # Wait for execd to be ready
    sleep 2
fi

# Keep container running
echo "browser-entrypoint.sh: keeping container alive for execd"
wait $execd_pid

if [ ! -z "$xvfb_pid" ]; then
  wait $xvfb_pid
fi
