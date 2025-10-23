#!/usr/bin/env bash
# Build script for cross-compiling to FreeBSD/amd64 (NFSN target)
set -euo pipefail

echo "Building for FreeBSD/amd64..."

# Ensure build directory exists
mkdir -p build

# Build the static binary for FreeBSD
cd generator
GOOS=freebsd GOARCH=amd64 CGO_ENABLED=0 go build \
  -trimpath \
  -ldflags="-s -w" \
  -o ../build/buildsite \
  ./cmd/buildsite
cd ..

echo "Build complete: build/buildsite"
echo "Binary info:"
file build/buildsite 2>/dev/null || echo "  (file command not available)"
ls -lh build/buildsite
echo ""
echo "Ready to deploy to NearlyFreeSpeech.NET"
echo "Don't forget to upload config.toml along with the binary!"
