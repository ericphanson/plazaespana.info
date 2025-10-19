# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Static site generator for Madrid events near Plaza de España, deployed to NearlyFreeSpeech.NET (FreeBSD hosting). Built with Go, runs as a cron job to fetch event data from Madrid's open data portal, filter by location, and regenerate static HTML/JSON output.

**Key constraints:**
- Target platform: FreeBSD/amd64 (NearlyFreeSpeech.NET)
- Must cross-compile with `GOOS=freebsd GOARCH=amd64 CGO_ENABLED=0`
- Fully static binary (no CGO dependencies)
- Robust fallback handling when upstream data sources fail
- Atomic file writes to prevent serving partial updates

## Build Commands

### Cross-compile for FreeBSD/amd64 (production)
```bash
./scripts/build-freebsd.sh
# or manually:
GOOS=freebsd GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o build/buildsite ./cmd/buildsite
```

### Build for local testing (Linux)
```bash
go build -o build/buildsite ./cmd/buildsite
```

### Run tests
```bash
go test ./...
```

### Run a single test
```bash
go test -v -run TestFunctionName ./internal/package
```

### Run locally
```bash
./build/buildsite \
  -json-url https://datos.madrid.es/egob/catalogo/300107-0-agenda-actividades-eventos.json \
  -xml-url https://datos.madrid.es/egob/catalogo/300107-0-agenda-actividades-eventos.xml \
  -csv-url https://datos.madrid.es/egob/catalogo/300107-0-agenda-actividades-eventos.csv \
  -out-dir ./public \
  -data-dir ./data \
  -lat 40.42338 -lon -3.71217 -radius-km 0.35 \
  -timezone Europe/Madrid
```

## Architecture

### Code Structure

The codebase follows a standard Go CLI application pattern:

```
cmd/buildsite/          # Main entry point
  main.go               # CLI flags, orchestration, error handling

internal/
  fetch/                # HTTP client for Madrid open data API
    client.go           # Multi-format fetcher (JSON → XML → CSV fallback)
    types.go            # Raw event structure matching Madrid's API

  parse/                # Format-specific decoders
    json.go             # Decode JSON events
    xml.go              # Decode XML events (fallback)
    csv.go              # Decode CSV events (second fallback)

  filter/               # Location and time filtering
    geo.go              # Haversine distance calculation
    time.go             # Parse/normalize to Europe/Madrid timezone
    dedupe.go           # Deduplicate by ID-EVENTO

  render/               # Static site generation
    html.go             # Generate index.html via html/template
    json.go             # Generate events.json machine-readable output
    ics.go              # (Optional) Generate iCalendar feed

  snapshot/             # Resilience/fallback system
    manager.go          # Save/load last successful fetch

templates/
  index.tmpl.html       # HTML template for main page

assets/
  site.css              # Hand-rolled CSS (hashed filename in production)

ops/
  htaccess              # Apache caching rules for NFSN deployment
```

### Data Flow

1. **Fetch**: Try JSON, fall back to XML, then CSV if needed
2. **Parse**: Decode into normalized `Event` structs
3. **Filter**:
   - Geographic: Haversine distance ≤ 0.35 km from Plaza de España (40.42338, -3.71217)
   - Temporal: Drop events in the past
   - Deduplication: By `ID-EVENTO` field
4. **Render**: Generate `index.html` and `events.json` in temp files
5. **Atomic write**: Rename temp files to final location (prevents partial updates)
6. **Snapshot**: Save successful fetch for future fallback

### Robustness Strategy

- **Three-tier fallback**: JSON → XML → CSV
- **Last successful snapshot**: If all sources fail, serve previously cached data with "stale" indicator
- **Atomic writes**: Use temp files + rename to prevent serving incomplete output
- **Graceful degradation**: Missing fields (e.g., HORA) treated as all-day events
- **Timezone normalization**: All times parsed to Europe/Madrid

## Key Implementation Details

### Event Structure

Madrid's open data API provides events with these key fields (see README.md section 10 for full structure):
- `ID-EVENTO`: Unique identifier (use for deduplication)
- `TITULO`: Event title
- `FECHA` / `FECHA-FIN`: Start/end dates
- `HORA`: Time (may be missing → all-day event)
- `COORDENADA-LATITUD` / `COORDENADA-LONGITUD`: GPS coordinates
- `NOMBRE-INSTALACION`: Venue name
- `CONTENT-URL`: Link to full event details

### Geographic Filtering

Use Haversine formula to calculate distance between event coordinates and Plaza de España reference point (40.42338, -3.71217). Filter radius: 0.35 km (configurable via `-radius-km` flag).

Secondary text match: If coordinates missing, check if `NOMBRE-INSTALACION` or `DIRECCION` contains "Plaza de España" (case-insensitive).

### Time Handling

- Parse dates with timezone awareness (Europe/Madrid)
- Handle missing `HORA` field → treat as all-day event
- Filter out events where end time is in the past
- Sort remaining events by start datetime

### Deployment to NFSN

**Target environment:**
- OS: FreeBSD (member sites)
- Document root: `/home/public`
- Binary location: `/home/bin/buildsite`
- Data directory: `/home/data`

**Scheduled task (cron):**
Run hourly via NFSN's Scheduled Tasks UI. Command:
```bash
/home/bin/buildsite -json-url https://datos.madrid.es/egob/catalogo/300107-0-agenda-actividades-eventos.json -xml-url https://datos.madrid.es/egob/catalogo/300107-0-agenda-actividades-eventos.xml -csv-url https://datos.madrid.es/egob/catalogo/300107-0-agenda-actividades-eventos.csv -out-dir /home/public -data-dir /home/data -lat 40.42338 -lon -3.71217 -radius-km 0.35 -timezone Europe/Madrid
```

**Caching headers:**
Copy `ops/htaccess` to `/home/public/.htaccess`:
- HTML/JSON: 5 minutes (frequent updates)
- CSS/images: 30 days (hashed filenames)

## Dependencies

- **Standard library only** for core functionality (net/http, encoding/json, encoding/xml, encoding/csv, html/template, time)
- **No CGO**: Must be `CGO_ENABLED=0` for FreeBSD cross-compilation
- If SQLite is later added: Use `modernc.org/sqlite` (pure Go) instead of `mattn/go-sqlite3` (requires CGO)

## Development Environment

Uses devcontainer with:
- Go toolchain
- FreeBSD cross-compilation support
- Claude Code integration
- Git, delta, zsh for developer experience

## Data Sources

Primary: Madrid open data portal (Ayuntamiento de Madrid)
- JSON: https://datos.madrid.es/egob/catalogo/300107-0-agenda-actividades-eventos.json
- XML: https://datos.madrid.es/egob/catalogo/300107-0-agenda-actividades-eventos.xml
- CSV: https://datos.madrid.es/egob/catalogo/300107-0-agenda-actividades-eventos.csv

License: Attribution required to "Ayuntamiento de Madrid – datos.madrid.es"

## Testing Strategy

- Unit tests for each internal package
- Mock HTTP responses for fetch package tests
- Golden file tests for render output
- Test CSV parsing with both comma and semicolon delimiters
- Test haversine calculation with known distances
- Test timezone edge cases (DST transitions)

## Important Notes

- **Never use CGO**: Cross-compilation to FreeBSD requires pure Go
- **Atomic writes**: Always write to temp file, then rename
- **User-Agent header**: Set helpful User-Agent with contact info when fetching
- **Rate limiting**: Implement backoff on HTTP errors
- **Attribution**: Include Madrid open data attribution in rendered output
- **Timezone**: All time operations must use Europe/Madrid (not UTC)

## Development Workflow (for Claude Code)

When executing implementation plans:

1. **Logging**: Create and maintain a log file in `docs/logs/` for each plan execution
   - One log per plan (e.g., `docs/logs/2025-10-19-madrid-events-implementation.md`)
   - Update the log after completing each subtask with status and notes
   - Track progress, issues encountered, and resolutions

2. **Commit workflow**: After each subtask:
   - Format/lint the code
   - Run tests to verify functionality
   - Update the log file
   - Commit with yourself as coauthor:
     ```
     Co-Authored-By: Claude <noreply@anthropic.com>
     ```

3. **Console output**: Minimize spam to main console
   - Don't echo large diffs or verbose output
   - Summarize results concisely
   - Direct verbose output to log files

4. **Subagent coordination**: When using parallel subagents
   - Avoid file conflicts by ensuring subtasks operate on different files
   - Execute tasks sequentially if they touch the same code
   - Coordinate through shared log file for status updates

5. **Tooling priority**: Address tooling/devcontainer issues upfront
   - Verify development tools (Go version, linters, formatters) early
   - Test build scripts before implementing features
   - Identify missing dependencies before deep implementation work
   - This allows fixing environment issues before they block progress
