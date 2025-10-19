# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Static site generator for Madrid events near Plaza de España, deployed to NearlyFreeSpeech.NET (FreeBSD hosting). Built with Go, runs as a cron job to fetch event data from Madrid's open data portal, filter by location, and regenerate static HTML/JSON output.

**Implementation Status:** ✅ **PRODUCTION READY**
- All 20 implementation tasks completed
- 22 tests passing (100% success rate)
- FreeBSD binary built and verified (7.7 MB, static)
- CSS hash fix applied
- Module path customized to github.com/ericphanson/madrid-events

**Key constraints:**
- Target platform: FreeBSD/amd64 (NearlyFreeSpeech.NET)
- Must cross-compile with `GOOS=freebsd GOARCH=amd64 CGO_ENABLED=0`
- Fully static binary (no CGO dependencies)
- Robust fallback handling when upstream data sources fail
- Atomic file writes to prevent serving partial updates

## Build Commands

**Primary interface:** Use `just` (see `justfile` for all commands)

### Quick commands
```bash
just           # List all available tasks
just dev       # Build site and serve locally at :8080
just build     # Build binary for local testing
just test      # Run all tests (22 tests)
just freebsd   # Cross-compile for FreeBSD/amd64
just hash-css  # Generate content-hashed CSS
just clean     # Clean build artifacts
```

### Manual equivalents
```bash
# Build for FreeBSD
GOOS=freebsd GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o build/buildsite ./cmd/buildsite

# Run tests
go test ./...

# Run locally
./build/buildsite \
  -json-url https://datos.madrid.es/egob/catalogo/300107-0-agenda-actividades-eventos.json \
  -xml-url https://datos.madrid.es/egob/catalogo/300107-0-agenda-actividades-eventos.xml \
  -csv-url https://datos.madrid.es/egob/catalogo/300107-0-agenda-actividades-eventos.csv \
  -out-dir ./public \
  -data-dir ./data
```

## Architecture

### Code Structure (Actual Implementation)

```
cmd/buildsite/
  main.go                          # CLI orchestration (205 lines)
  main_integration_test.go         # Integration test skeleton

internal/
  fetch/                           # HTTP client + multi-format parsing
    client.go                      # JSON/XML/CSV fetching with User-Agent
    client_test.go                 # 7 tests (httptest-based mocks)
    types.go                       # RawEvent, JSONResponse, XMLResponse structs

  filter/                          # Filtering and data processing
    geo.go                         # Haversine distance calculation
    geo_test.go                    # 6 tests (known distances)
    time.go                        # Date/time parsing (Europe/Madrid)
    time_test.go                   # 4 tests (timezone-aware)
    dedupe.go                      # Deduplication by ID-EVENTO
    dedupe_test.go                 # 2 tests

  render/                          # Static site generation
    types.go                       # TemplateData, TemplateEvent, JSONEvent
    html.go                        # HTML rendering with atomic writes
    html_test.go                   # Template rendering test
    json.go                        # JSON API rendering
    json_test.go                   # JSON encoding test

  snapshot/                        # Fallback resilience
    manager.go                     # Save/load snapshots (atomic writes)
    manager_test.go                # 2 tests (save/load cycle)

templates/
  index.tmpl.html                  # HTML template (Spanish, semantic HTML5)

assets/
  site.css                         # Hand-rolled CSS (1.2 KB, dark mode support)

scripts/
  build-freebsd.sh                 # FreeBSD cross-compilation script
  hash-assets.sh                   # CSS content hashing

ops/
  htaccess                         # Apache caching + security headers
  deploy-notes.md                  # NFSN deployment instructions

justfile                           # Task automation (just command runner)
```

**Module:** `github.com/ericphanson/madrid-events`
**Binary size:** 7.7 MB (FreeBSD/amd64, statically linked)
**Test coverage:** 22 tests across 4 packages (79.7% fetch, 96.2% filter, 66.7% render, 76.2% snapshot)

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

**Implemented:** Uses Haversine formula in `internal/filter/geo.go` to calculate great-circle distance between event coordinates and Plaza de España reference point (40.42338, -3.71217). Filter radius: 0.35 km (configurable via `-radius-km` flag).

**Current implementation:** Events without coordinates (lat/lon == 0) are filtered out by the main pipeline. Secondary text matching is not currently implemented but could be added if needed.

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

## Testing Strategy (Implemented)

**Test Coverage:** 22 tests passing (100% success rate)

- ✅ **fetch package** (7 tests): Mock HTTP responses with httptest, User-Agent verification, JSON/XML/CSV parsing, delimiter detection
- ✅ **filter package** (12 tests): Haversine distance with known values, timezone-aware date parsing, deduplication, future/past filtering
- ✅ **render package** (2 tests): HTML template rendering, JSON encoding, atomic write verification
- ✅ **snapshot package** (2 tests): Save/load cycle, atomic writes, error handling
- ✅ **Integration test** (1 test): Pipeline structure validation (skeleton, expandable)

**TDD Approach:** All features implemented test-first (RED-GREEN-REFACTOR cycle documented in `docs/logs/2025-10-19-madrid-events-implementation.md`)

## Important Notes

- **Never use CGO**: Cross-compilation to FreeBSD requires pure Go ✅ (verified: `CGO_ENABLED=0`)
- **Atomic writes**: Always write to temp file, then rename ✅ (implemented in render + snapshot packages)
- **User-Agent header**: Set helpful User-Agent with contact info when fetching ✅ (`madrid-events-site-generator/1.0 (https://github.com/ericphanson/madrid-events)`)
- **Attribution**: Include Madrid open data attribution in rendered output ✅ (in HTML template footer)
- **Timezone**: All time operations must use Europe/Madrid (not UTC) ✅ (implemented in `filter/time.go`)

## Project Files

**Documentation:**
- `README.md` - Quick start guide (just-focused, 94 lines)
- `docs/design.md` - Comprehensive design documentation (moved from old README)
- `docs/plans/2025-10-19-madrid-events-site-generator.md` - 20-task implementation plan (2,271 lines)
- `docs/logs/2025-10-19-madrid-events-implementation.md` - Implementation log with commit history (1,380+ lines)
- `ops/deploy-notes.md` - NFSN deployment instructions

**Key Stats:**
- Total commits: 48+ (including implementation + refinements)
- Lines of Go code: ~1,299 (source only)
- Total project files: 47 files across 18 directories
- Zero external dependencies (stdlib only)

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
