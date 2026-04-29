#!/bin/bash
set -e

# Disable Jupyter for browser sandboxes (not needed)
export JUPYTER_HOST=""
export JUPYTER_TOKEN=""

# Check if execd is already running (injected by OpenSandbox server)
if pgrep -x execd > /dev/null; then
    echo "execd is already running, skipping manual start"
else
    # Start execd (for standalone use without OpenSandbox server)
    /usr/local/bin/execd &
    # Wait for execd to be ready
    sleep 2
fi

# Keep container running by tailing /dev/null
# execd is managed by OpenSandbox bootstrap.sh and runs independently
echo "browser-entrypoint.sh: keeping container alive for execd"
tail -f /dev/null
