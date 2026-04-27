#!/bin/bash

# Use Playwright-installed Chromium path
if [ -n "$CHROMIUM_PATH" ]; then
    # Expand wildcard if present
    BROWSER_BIN=$(echo $CHROMIUM_PATH)
else
    # Try to find Playwright chromium
    PLAYWRIGHT_CHROMIUM=$(npx playwright show-trace 2>&1 | grep -o '/root/.cache/ms-playwright/chromium-[0-9]*/chrome-linux/chrome' | head -1)
    if [ -n "$PLAYWRIGHT_CHROMIUM" ] && [ -f "$PLAYWRIGHT_CHROMIUM" ]; then
        BROWSER_BIN="$PLAYWRIGHT_CHROMIUM"
    else
        # Fallback to common paths
        BROWSER_BIN="/usr/bin/chromium-browser"
    fi
fi

echo "Using browser binary: $BROWSER_BIN"

CDP_PORT=${BROWSER_CDP_PORT:-0}
USER_DATA_DIR=${1:-/tmp/browser-data-$$}
ENABLE_OVERLAYFS=${ENABLE_OVERLAYFS_SNAPSHOTS:-false}

# If port is 0, generate a random port in range 30000-60000
if [ "$CDP_PORT" = "0" ]; then
    CDP_PORT=$((RANDOM % 30000 + 30000))
fi

# OverlayFS configuration
OVERLAY_BASE_DIR=${OVERLAY_BASE_DIR:-/var/lib/overlay}
OVERLAY_WORK_DIR="$OVERLAY_BASE_DIR/work"
OVERLAY_UPPER_DIR="$OVERLAY_BASE_DIR/upper"
OVERLAY_LOWER_DIR="$OVERLAY_BASE_DIR/lower"
OVERLAY_MERGE_DIR="$OVERLAY_BASE_DIR/merged"

# Setup OverlayFS if enabled
if [ "$ENABLE_OVERLAYFS" = "true" ]; then
    echo "Setting up OverlayFS..."
    # Create OverlayFS directory structure
    mkdir -p "$OVERLAY_WORK_DIR" "$OVERLAY_UPPER_DIR" "$OVERLAY_LOWER_DIR" "$OVERLAY_MERGE_DIR"

    # Initialize lower layer if empty (copy from base browser profile)
    if [ -z "$(ls -A $OVERLAY_LOWER_DIR)" ]; then
        mkdir -p "$OVERLAY_LOWER_DIR/chromium-profile"
    fi

    # Mount OverlayFS
    mount -t overlay overlay \
        -o lowerdir="$OVERLAY_LOWER_DIR" \
        -o upperdir="$OVERLAY_UPPER_DIR" \
        -o workdir="$OVERLAY_WORK_DIR" \
        "$OVERLAY_MERGE_DIR" || {
        echo "Warning: Failed to mount OverlayFS, using regular directory"
        USER_DATA_DIR="/tmp/browser-data-$$"
        mkdir -p "$USER_DATA_DIR"
    }

    # Use merged directory as user data directory
    if mountpoint -q "$OVERLAY_MERGE_DIR"; then
        USER_DATA_DIR="$OVERLAY_MERGE_DIR/chromium-profile"
        mkdir -p "$USER_DATA_DIR"
    fi
else
    echo "OverlayFS disabled, using regular directory"
    # Create user data directory directly
    mkdir -p "$USER_DATA_DIR"
fi

# Launch Chromium with CDP enabled in background
"$BROWSER_BIN" \
    --remote-debugging-port="$CDP_PORT" \
    --user-data-dir="$USER_DATA_DIR" \
    --remote-debugging-address=0.0.0.0 \
    --remote-allow-origins=* \
    $BROWSER_LAUNCH_ARGS \
    "$@" &

BROWSER_PID=$!

# Output the CDP port to stdout in a specific format
echo "CDP_PORT:$CDP_PORT"

# Keep the script running (don't exit, let browser run)
wait $BROWSER_PID
