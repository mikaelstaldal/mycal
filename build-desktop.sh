#!/usr/bin/env bash
# Cross-build the optional desktop client; the server remains a separate build.
set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")"

OUTPUT_DIR="${1:-dist/desktop}"
if [ "$#" -gt 1 ]; then
    echo "usage: ./build-desktop.sh [OUTPUT_DIRECTORY]" >&2
    exit 1
fi
mkdir -p "$OUTPUT_DIR"

for desktop_os in linux windows darwin; do
    for desktop_arch in amd64 arm64; do
        suffix=""
        if [ "$desktop_os" = windows ]; then
            suffix=".exe"
        fi
        CGO_ENABLED=0 GOOS="$desktop_os" GOARCH="$desktop_arch" \
            go build -trimpath -buildvcs=true -tags netgo \
            -o "$OUTPUT_DIR/mycal-open-${desktop_os}-${desktop_arch}${suffix}" ./cmd/open
    done
done
