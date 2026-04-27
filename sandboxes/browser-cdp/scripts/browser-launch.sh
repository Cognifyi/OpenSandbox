#!/bin/bash
set -e

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

# Launch Chromium with CDP enabled
exec "$BROWSER_BIN" \
    --remote-debugging-port="$CDP_PORT" \
    --user-data-dir="$USER_DATA_DIR" \
    $BROWSER_LAUNCH_ARGS \
    "$@"
