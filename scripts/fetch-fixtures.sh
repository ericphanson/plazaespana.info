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
  METADATA_FILE=$(mktemp)
  RETRY_COUNT=0
  MAX_RETRIES=3

  while [ $RETRY_COUNT -lt $MAX_RETRIES ]; do
    timeout 10 curl --max-time 10 -s -H "api_key: $AEMET_API_KEY" \
      "https://opendata.aemet.es/opendata/api/prediccion/especifica/municipio/diaria/28079" \
      -o "$METADATA_FILE" 2>/dev/null
    CURL_EXIT=$?
    if [ $CURL_EXIT -eq 0 ] && [ -s "$METADATA_FILE" ]; then
      # Success - file downloaded and is not empty
      break
    else
      RETRY_COUNT=$((RETRY_COUNT + 1))
      if [ $RETRY_COUNT -lt $MAX_RETRIES ]; then
        echo "  - Retry $RETRY_COUNT/$MAX_RETRIES (API timeout/error), waiting 5s..."
        sleep 5
      fi
    fi
  done

  if [ $RETRY_COUNT -ge $MAX_RETRIES ]; then
    echo "⚠️  AEMET API failed after $MAX_RETRIES attempts"
    echo "   The API may be down or rate-limiting requests"
    rm -f "$METADATA_FILE"
    # Don't exit - continue with icon fetching
    METADATA=""
  else
    METADATA=$(cat "$METADATA_FILE")
    rm "$METADATA_FILE"
  fi

  # Extract datos URL from metadata response using jq
  DATOS_URL=$(echo "$METADATA" | jq -r '.datos // empty')

  if [ -z "$DATOS_URL" ]; then
    echo "⚠️  Failed to get datos URL from AEMET API"
    echo "   Response: $METADATA"
  else
    echo "  - Downloading forecast data from: $DATOS_URL"
    curl --max-time 30 -s -L "$DATOS_URL" -o "$FIXTURES_DIR/aemet-madrid-forecast.json"

    # Also save the metadata for reference
    echo "$METADATA" > "$FIXTURES_DIR/aemet-madrid-metadata.json"

    echo "✓ Weather fixture downloaded to $FIXTURES_DIR/"
    echo "  AEMET forecast: $(wc -l < "$FIXTURES_DIR/aemet-madrid-forecast.json") lines"
    echo "  AEMET metadata: $(wc -l < "$FIXTURES_DIR/aemet-madrid-metadata.json") lines"
  fi
fi

echo ""
echo "Fetching AEMET weather icons..."

# Create icons directory
ICONS_DIR="$FIXTURES_DIR/aemet-icons"
mkdir -p "$ICONS_DIR"

# AEMET sky state codes (based on meteosapi/AEMET documentation)
# Note: Not all codes may have icons, but we'll try to fetch what exists
ICON_CODES=(
  # Basic conditions
  11 11n 12 12n 13 13n 14 14n 15 15n 16 16n 17 17n
  # With rain
  23 23n 24 24n 25 25n 26 26n
  # With snow
  33 33n 34 34n 35 35n 36 36n
  # With light rain
  43 43n 44 44n 45 45n 46 46n
  # With storms
  51 51n 52 52n 53 53n 54 54n
  # Additional codes (may not all exist)
  61 61n 62 62n 63 63n 64 64n
  71 71n 72 72n 73 73n 74 74n
)

ICON_COUNT=0
FAILED_COUNT=0

for code in "${ICON_CODES[@]}"; do
  # Strip 'n' suffix for filename (11n → 11.png works for both)
  BASE_CODE="${code%n}"
  ICON_FILE="$ICONS_DIR/${BASE_CODE}.png"

  # Skip if already downloaded
  if [ -f "$ICON_FILE" ]; then
    continue
  fi

  ICON_URL="https://www.aemet.es/imagenes/png/estado_cielo/${BASE_CODE}.png"

  if curl -f -s -L "$ICON_URL" -o "$ICON_FILE" 2>/dev/null; then
    ((ICON_COUNT++))
  else
    ((FAILED_COUNT++))
    rm -f "$ICON_FILE"  # Clean up failed download
  fi
done

echo "✓ AEMET icons fetched to $ICONS_DIR/"
echo "  Downloaded: $ICON_COUNT icons"
if [ $FAILED_COUNT -gt 0 ]; then
  echo "  Failed/missing: $FAILED_COUNT codes"
fi

echo ""
echo "✓ All fixtures fetched successfully"
