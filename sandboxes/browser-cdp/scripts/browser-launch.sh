#!/bin/bash

BROWSER_BIN=${CHROMIUM_PATH:-/usr/bin/chromium-browser}
CDP_PORT=${BROWSER_CDP_PORT:-0}
USER_DATA_DIR=${1:-/tmp/browser-data-$$}
ENABLE_OVERLAYFS=${ENABLE_OVERLAYFS_SNAPSHOTS:-false}

# OverlayFS configuration
OVERLAY_BASE_DIR=${OVERLAY_BASE_DIR:-/var/lib/overlay}
OVERLAY_WORK_DIR="$OVERLAY_BASE_DIR/work"
OVERLAY_UPPER_DIR="$OVERLAY_BASE_DIR/upper"
OVERLAY_LOWER_DIR="$OVERLAY_BASE_DIR/lower"
OVERLAY_MERGE_DIR="$OVERLAY_BASE_DIR/merged"

# Setup OverlayFS if enabled
if [ "$ENABLE_OVERLAYFS" = "true" ]; then
    echo "Setting up OverlayFS for snapshot support..."

    # Create OverlayFS directory structure
    mkdir -p "$OVERLAY_WORK_DIR" "$OVERLAY_UPPER_DIR" "$OVERLAY_LOWER_DIR" "$OVERLAY_MERGE_DIR"

    # Initialize lower layer if empty (copy from base browser profile)
    if [ -z "$(ls -A $OVERLAY_LOWER_DIR)" ]; then
        echo "Initializing lower layer with base browser profile..."
        mkdir -p "$OVERLAY_LOWER_DIR/chromium-profile"
    fi

    # Mount OverlayFS
    mount -t overlay overlay \
        -o lowerdir="$OVERLAY_LOWER_DIR" \
        -o upperdir="$OVERLAY_UPPER_DIR" \
        -o workdir="$OVERLAY_WORK_DIR" \
        "$OVERLAY_MERGE_DIR"

    # Use merged directory as user data directory
    USER_DATA_DIR="$OVERLAY_MERGE_DIR/chromium-profile"
    mkdir -p "$USER_DATA_DIR"

    echo "OverlayFS mounted at $OVERLAY_MERGE_DIR"
else
    # Create user data directory directly
    mkdir -p "$USER_DATA_DIR"
fi

# Launch Chromium with CDP enabled in background
"$BROWSER_BIN" \
    --remote-debugging-port="$CDP_PORT" \
    --user-data-dir="$USER_DATA_DIR" \
    $BROWSER_LAUNCH_ARGS \
    "$@" &

BROWSER_PID=$!

# Wait for browser to start
sleep 2

# If port was 0 (auto-assign), find the actual port
if [ "$CDP_PORT" = "0" ]; then
    # Find the listening port for this process
    CDP_PORT=$(ss -tlnp 2>/dev/null | grep "pid=$BROWSER_PID" | awk '{print $4}' | cut -d: -f2 | head -1)
    if [ -z "$CDP_PORT" ]; then
        # Fallback: use lsof
        CDP_PORT=$(lsof -nP -iTCP -sTCP:LISTEN -p $BROWSER_PID 2>/dev/null | awk '{print $9}' | cut -d: -f2 | head -1)
    fi
    # If still empty, default to 9222
    if [ -z "$CDP_PORT" ]; then
        CDP_PORT=9222
    fi
fi

# Output the actual CDP port to stdout
echo "$CDP_PORT"

# Keep the script running (don't exit, let browser run)
wait $BROWSER_PID
