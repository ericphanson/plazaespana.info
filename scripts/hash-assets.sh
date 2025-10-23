#!/usr/bin/env bash
# Generate content-hash filenames for CSS assets
set -euo pipefail

ASSETS_DIR="generator/assets"
PUBLIC_ASSETS_DIR="public/assets"

mkdir -p "$PUBLIC_ASSETS_DIR"
mkdir -p "public"

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

# Hash build report CSS and copy to public/assets
BUILD_REPORT_CSS="$ASSETS_DIR/build-report.css"
if [ -f "$BUILD_REPORT_CSS" ]; then
  REPORT_HASH=$(sha256sum "$BUILD_REPORT_CSS" | cut -c1-8)
  cp "$BUILD_REPORT_CSS" "$PUBLIC_ASSETS_DIR/build-report.$REPORT_HASH.css"
  echo "$REPORT_HASH" > "$PUBLIC_ASSETS_DIR/build-report-css.hash"
  echo "Generated: public/assets/build-report.$REPORT_HASH.css"
else
  echo "Warning: $BUILD_REPORT_CSS not found"
fi

# Copy robots.txt to public root
ROBOTS_FILE="ops/robots.txt"
if [ -f "$ROBOTS_FILE" ]; then
  cp "$ROBOTS_FILE" "public/robots.txt"
  echo "Copied: public/robots.txt"
else
  echo "Warning: $ROBOTS_FILE not found"
fi

# Copy weather icons to public/assets/weather-icons
WEATHER_ICONS_SRC="generator/testdata/fixtures/aemet-icons"
WEATHER_ICONS_DEST="$PUBLIC_ASSETS_DIR/weather-icons"
if [ -d "$WEATHER_ICONS_SRC" ]; then
  mkdir -p "$WEATHER_ICONS_DEST"
  cp "$WEATHER_ICONS_SRC"/*.png "$WEATHER_ICONS_DEST/" 2>/dev/null || true
  ICON_COUNT=$(ls -1 "$WEATHER_ICONS_DEST"/*.png 2>/dev/null | wc -l || echo 0)
  echo "Copied: $ICON_COUNT weather icons to public/assets/weather-icons/"
else
  echo "Warning: $WEATHER_ICONS_SRC not found (weather icons skipped)"
fi
