#!/bin/sh

[ -n "${BASH_VERSION:-}" ] || exec /bin/bash "$0" "$@"

set -eu

find_browser_bin() {
    if [ -n "${CHROMIUM_PATH:-}" ] && [ -x "${CHROMIUM_PATH}" ]; then
        printf '%s\n' "${CHROMIUM_PATH}"
        return 0
    fi

    for candidate in \
        /usr/local/bin/playwright-browsers/chromium-*/chrome-linux64/chrome \
        /root/.cache/ms-playwright/chromium-*/chrome-linux/chrome \
        /usr/bin/chromium-browser \
        /usr/bin/chromium
    do
        for expanded in $candidate; do
            if [ -x "$expanded" ]; then
                printf '%s\n' "$expanded"
                return 0
            fi
        done
    done

    return 1
}

is_true() {
    case "${1:-}" in
        1|true|TRUE|yes|YES|on|ON) return 0 ;;
        *) return 1 ;;
    esac
}

wait_for_cdp() {
    attempts=0
    max_attempts=${BROWSER_START_TIMEOUT_SECONDS:-15}
    while [ "$attempts" -lt "$max_attempts" ]; do
        if ! kill -0 "$BROWSER_PID" 2>/dev/null; then
            return 1
        fi

        if python3 - "$CDP_PORT" <<'PY' >/dev/null 2>&1
import socket
import sys

sock = socket.socket()
sock.settimeout(0.5)
try:
    sock.connect(("127.0.0.1", int(sys.argv[1])))
except OSError:
    sys.exit(1)
finally:
    sock.close()
sys.exit(0)
PY
        then
            return 0
        fi

        attempts=$((attempts + 1))
        sleep 1
    done

    return 1
}

setup_xvfb_if_needed() {
    if is_true "${BROWSER_HEADLESS:-true}"; then
        return 0
    fi

    if [ -n "${DISPLAY:-}" ]; then
        return 0
    fi

    [ -f /tmp/.X99-lock ] && rm -f /tmp/.X99-lock
    Xvfb :99 -screen 0 1024x768x16 -nolisten tcp -nolisten unix >/tmp/browser-xvfb.log 2>&1 &
    export DISPLAY=:99
}

setup_user_data_dir() {
    requested_dir="$1"
    user_data_dir="$requested_dir"

    if ! is_true "${ENABLE_OVERLAYFS_SNAPSHOTS:-false}"; then
        mkdir -p "$user_data_dir"
        printf '%s\n' "$user_data_dir"
        return 0
    fi

    if [ "${BROWSER_OVERLAYFS_SNAPSHOT_BACKEND:-}" != "overlayfs" ]; then
        echo "OverlayFS snapshot backend is disabled; using regular profile directory" >&2
        mkdir -p "$user_data_dir"
        printf '%s\n' "$user_data_dir"
        return 0
    fi

    echo "OverlayFS snapshot backend is intentionally unavailable until session-scoped snapshot storage is implemented" >&2
    mkdir -p "$requested_dir"
    printf '%s\n' "$requested_dir"
}

BROWSER_BIN=$(find_browser_bin) || {
    echo "Failed to locate Chromium binary" >&2
    exit 1
}

echo "Using browser binary: $BROWSER_BIN" >&2

CDP_PORT=${BROWSER_CDP_PORT:-0}
REQUESTED_USER_DATA_DIR=${1:-/tmp/browser-data-$$}
USER_DATA_DIR=$(setup_user_data_dir "$REQUESTED_USER_DATA_DIR")

if [ "$CDP_PORT" = "0" ]; then
    CDP_PORT=$((RANDOM % 30000 + 30000))
fi

HEADLESS_ARGS=""
if is_true "${BROWSER_HEADLESS:-true}"; then
    HEADLESS_ARGS="--headless=new --disable-gpu"
fi

setup_xvfb_if_needed

LOG_FILE=${BROWSER_LOG_FILE:-/tmp/browser-launch.log}

nohup "$BROWSER_BIN" \
    --remote-debugging-port="$CDP_PORT" \
    --user-data-dir="$USER_DATA_DIR" \
    --remote-debugging-address=0.0.0.0 \
    --remote-allow-origins=* \
    $HEADLESS_ARGS \
    ${BROWSER_LAUNCH_ARGS:-} \
    about:blank >>"$LOG_FILE" 2>&1 &

BROWSER_PID=$!

if ! wait_for_cdp; then
    echo "Browser failed to become ready on port $CDP_PORT" >&2
    kill "$BROWSER_PID" 2>/dev/null || true
    wait "$BROWSER_PID" 2>/dev/null || true
    exit 1
fi

echo "CDP_PORT:$CDP_PORT"
echo "BROWSER_PID:$BROWSER_PID"
echo "USER_DATA_DIR:$USER_DATA_DIR"
