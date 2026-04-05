#!/usr/bin/env bash
set -euo pipefail

BINARY=./mycal
PORT=8089
DATA_DIR=/tmp/claude

SERVER_PID=

stop_server() {
    if [ -n "$SERVER_PID" ]; then
        kill "$SERVER_PID" 2>/dev/null || true
        wait "$SERVER_PID" 2>/dev/null || true
    fi
}

trap stop_server EXIT

"$BINARY" -port "$PORT" -data "$DATA_DIR" &
SERVER_PID=$!

# Wait for server to be ready
for i in $(seq 1 20); do
    if curl -sf "http://localhost:${PORT}/api/v1/events?from=2000-01-01T00:00:00Z&to=2000-01-02T00:00:00Z" > /dev/null 2>&1; then
        break
    fi
    if [ "$i" -eq 20 ]; then
        echo "Server failed to start" >&2
        exit 1
    fi
    sleep 0.5
done

cd e2e
playwright-test
