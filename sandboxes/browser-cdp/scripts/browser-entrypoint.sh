#!/bin/bash
set -e

# Start execd
/usr/local/bin/execd &

# Wait for execd to be ready
sleep 2

# Keep container running
wait
