#!/bin/bash
# Madrid Events UI Screenshot Capture Script
# Usage: ./capture.sh [timestamp]

set -e

TIMESTAMP=${1:-$(date +%Y%m%d-%H%M%S)}
OUTPUT_DIR="/workspace/screenshots/${TIMESTAMP}"
mkdir -p "${OUTPUT_DIR}"

BASE_URL="http://localhost:8080"

echo "📸 Capturing screenshots to ${OUTPUT_DIR}"

# Main events page - multiple viewports
echo "  → Events page (desktop full)..."
shot-scraper "${BASE_URL}" -o "${OUTPUT_DIR}/events-desktop-full.png" --width 1400

echo "  → Events page (desktop viewport)..."
shot-scraper "${BASE_URL}" -o "${OUTPUT_DIR}/events-desktop.png" --width 1400 --height 900

echo "  → Events page (tablet)..."
shot-scraper "${BASE_URL}" -o "${OUTPUT_DIR}/events-tablet.png" --width 768 --height 1024

echo "  → Events page (mobile)..."
shot-scraper "${BASE_URL}" -o "${OUTPUT_DIR}/events-mobile.png" --width 375 --height 812

echo "✅ Screenshots captured: $(ls -1 ${OUTPUT_DIR}/*.png | wc -l) files"
echo "📁 Location: ${OUTPUT_DIR}"
ls -lh "${OUTPUT_DIR}" | grep "\.png$"
