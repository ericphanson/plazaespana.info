#!/usr/bin/env bash
# Generate content-hash filenames for CSS assets
set -euo pipefail

ASSETS_DIR="assets"
PUBLIC_ASSETS_DIR="public/assets"

mkdir -p "$PUBLIC_ASSETS_DIR"

# Hash main site CSS and copy to public/assets
CSS_FILE="$ASSETS_DIR/site.css"
if [ -f "$CSS_FILE" ]; then
  HASH=$(sha256sum "$CSS_FILE" | cut -c1-8)
  cp "$CSS_FILE" "$PUBLIC_ASSETS_DIR/site.$HASH.css"
  echo "$HASH" > "$PUBLIC_ASSETS_DIR/css.hash"
  echo "Generated: public/assets/site.$HASH.css"
else
  echo "Warning: $CSS_FILE not found"
fi

# Copy build report CSS (no hash needed - not cache-busted)
BUILD_REPORT_CSS="$ASSETS_DIR/build-report.css"
if [ -f "$BUILD_REPORT_CSS" ]; then
  cp "$BUILD_REPORT_CSS" "$PUBLIC_ASSETS_DIR/build-report.css"
  echo "Copied: public/assets/build-report.css"
else
  echo "Warning: $BUILD_REPORT_CSS not found"
fi
