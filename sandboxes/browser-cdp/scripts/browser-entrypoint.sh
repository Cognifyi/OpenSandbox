#!/bin/bash
set -e

# Disable Jupyter for browser sandboxes (not needed)
export JUPYTER_HOST=""
export JUPYTER_TOKEN=""

# Start execd
/usr/local/bin/execd &

# Wait for execd to be ready
sleep 2

# Keep container running
wait
