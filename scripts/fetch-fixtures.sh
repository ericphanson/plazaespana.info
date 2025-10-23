#!/bin/bash
set -euo pipefail

FIXTURES_DIR="generator/testdata/fixtures"
mkdir -p "$FIXTURES_DIR"

echo "Fetching Madrid event data fixtures..."

# JSON
echo "  - Downloading JSON..."
curl -f -s -L "https://datos.madrid.es/egob/catalogo/300107-0-agenda-actividades-eventos.json" \
  -o "$FIXTURES_DIR/madrid-events.json"

# XML
echo "  - Downloading XML..."
curl -f -s -L "https://datos.madrid.es/egob/catalogo/300107-0-agenda-actividades-eventos.xml" \
  -o "$FIXTURES_DIR/madrid-events.xml"

# CSV
echo "  - Downloading CSV..."
curl -f -s -L "https://datos.madrid.es/egob/catalogo/300107-0-agenda-actividades-eventos.csv" \
  -o "$FIXTURES_DIR/madrid-events.csv"

echo "✓ Fixtures downloaded to $FIXTURES_DIR/"
echo "  JSON: $(wc -l < "$FIXTURES_DIR/madrid-events.json") lines"
echo "  XML:  $(wc -l < "$FIXTURES_DIR/madrid-events.xml") lines"
echo "  CSV:  $(wc -l < "$FIXTURES_DIR/madrid-events.csv") lines"
