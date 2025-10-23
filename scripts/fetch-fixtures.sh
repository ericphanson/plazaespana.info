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

# EsMadrid city events
echo "  - Downloading EsMadrid XML..."
curl -f -s -L "https://www.esmadrid.com/opendata/agenda_v1_es.xml" \
  -o "$FIXTURES_DIR/esmadrid-agenda.xml"

echo "✓ Event fixtures downloaded to $FIXTURES_DIR/"
echo "  Madrid JSON: $(wc -l < "$FIXTURES_DIR/madrid-events.json") lines"
echo "  Madrid XML:  $(wc -l < "$FIXTURES_DIR/madrid-events.xml") lines"
echo "  Madrid CSV:  $(wc -l < "$FIXTURES_DIR/madrid-events.csv") lines"
echo "  EsMadrid XML: $(wc -l < "$FIXTURES_DIR/esmadrid-agenda.xml") lines"

# AEMET weather data (requires API key)
echo ""
echo "Fetching AEMET weather data fixture..."

if [ -z "${AEMET_API_KEY:-}" ]; then
  echo "⚠️  AEMET_API_KEY not set - skipping weather fixture"
  echo "   To fetch weather data:"
  echo "   1. Register at https://opendata.aemet.es/centrodedescargas/altaUsuario"
  echo "   2. Set AEMET_API_KEY environment variable"
  echo "   3. Run this script again"
else
  echo "  - Requesting forecast metadata for Madrid (28079)..."
  METADATA=$(curl -f -s -H "api_key: $AEMET_API_KEY" \
    "https://opendata.aemet.es/opendata/api/prediccion/especifica/municipio/diaria/28079")

  # Extract datos URL from metadata response
  DATOS_URL=$(echo "$METADATA" | grep -o '"datos":"[^"]*"' | cut -d'"' -f4)

  if [ -z "$DATOS_URL" ]; then
    echo "⚠️  Failed to get datos URL from AEMET API"
    echo "   Response: $METADATA"
  else
    echo "  - Downloading forecast data from: $DATOS_URL"
    curl -f -s -L "$DATOS_URL" -o "$FIXTURES_DIR/aemet-madrid-forecast.json"

    # Also save the metadata for reference
    echo "$METADATA" > "$FIXTURES_DIR/aemet-madrid-metadata.json"

    echo "✓ Weather fixture downloaded to $FIXTURES_DIR/"
    echo "  AEMET forecast: $(wc -l < "$FIXTURES_DIR/aemet-madrid-forecast.json") lines"
    echo "  AEMET metadata: $(wc -l < "$FIXTURES_DIR/aemet-madrid-metadata.json") lines"
  fi
fi

echo ""
echo "✓ All fixtures fetched successfully"
