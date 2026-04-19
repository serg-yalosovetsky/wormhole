#!/usr/bin/env bash
# Build the Android AAR from the native Go package.
# Prerequisites: Go 1.21+, gomobile installed (go install golang.org/x/mobile/cmd/gomobile@latest)

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
OUT_DIR="$SCRIPT_DIR/../android/app/libs"

mkdir -p "$OUT_DIR"

echo "Initialising gomobile..."
gomobile init

echo "Building wormhole.aar for Android..."
cd "$SCRIPT_DIR"
gomobile bind \
  -target android \
  -androidapi 24 \
  -o "$OUT_DIR/wormhole.aar" \
  .

echo "Done → $OUT_DIR/wormhole.aar"
