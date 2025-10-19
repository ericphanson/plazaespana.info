#!/usr/bin/env bash
# Generate content-hash filenames for CSS assets
set -euo pipefail

ASSETS_DIR="assets"
PUBLIC_ASSETS_DIR="public/assets"

mkdir -p "$PUBLIC_ASSETS_DIR"

# Hash CSS and copy to public/assets
CSS_FILE="$ASSETS_DIR/site.css"
if [ -f "$CSS_FILE" ]; then
  HASH=$(sha256sum "$CSS_FILE" | cut -c1-8)
  cp "$CSS_FILE" "$PUBLIC_ASSETS_DIR/site.$HASH.css"
  echo "$HASH" > "$PUBLIC_ASSETS_DIR/css.hash"
  echo "Generated: public/assets/site.$HASH.css"
else
  echo "Warning: $CSS_FILE not found"
fi
