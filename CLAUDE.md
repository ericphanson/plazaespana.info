# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Rules

- our `justfile` is the primary way we run things. CI scripts should call `just`. We should be able to run various tasks locally or in CI.
- this is a static site regenerated ~hourly. We only use CSS/HTML, no JS.
- we input data feeds, process them, and generate CSS/HTML/JSON. Since data processing is a core function, we generate an audit log so we can understand what has been filtered etc.
- we are careful about our upstream requests, especially during development. We use only fixtures during automated testing, and cache and wait during development runs.
- we keep anonymized aggregate stats in repo with awsstats
- we use github as a development platform, not deployment platform. Site regeneration happens on NFSN, not github.

## Project Overview

Static site generator for Madrid events near Plaza de España, deployed to NearlyFreeSpeech.NET (FreeBSD hosting). Built with Go, runs as a cron job to fetch event data from Madrid's open data portal, filter by location, and regenerate static HTML/JSON output.

**Implementation Status:** ✅ **PRODUCTION READY**
- All 20 implementation tasks completed
- 22 tests passing (100% success rate)
- FreeBSD binary built and verified (7.7 MB, static)
- CSS hash fix applied
- Module path customized to github.com/ericphanson/plazaespana.info

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
cd generator && GOOS=freebsd GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o ../build/buildsite ./cmd/buildsite

# Run tests
cd generator && go test ./...

# Run locally
./build/buildsite \
  -json-url https://datos.madrid.es/egob/catalogo/300107-0-agenda-actividades-eventos.json \
  -xml-url https://datos.madrid.es/egob/catalogo/300107-0-agenda-actividades-eventos.xml \
  -csv-url https://datos.madrid.es/egob/catalogo/300107-0-agenda-actividades-eventos.csv \
  -out-dir ./public \
  -data-dir ./data
```

## Architecture

### Code Structure

```
generator/          # All Go code and generator resources
  cmd/              # Main entry point
  internal/         # fetch, filter, render, pipeline, snapshot, etc.
  templates/        # HTML templates
  assets/           # CSS source files
  testdata/         # Test fixtures
  go.mod, go.sum    # Go module files

scripts/            # Build and deployment scripts
ops/                # Server configuration (htaccess, cron, awstats)
docs/               # Documentation
awstats-data/       # Anonymized aggregate stats
justfile            # Task automation
config.toml         # Runtime configuration
```

**Module:** `github.com/ericphanson/plazaespana.info`
**Binary size:** 7.7 MB (FreeBSD/amd64, statically linked)
**Test coverage:** 22 tests across 4 packages (79.7% fetch, 96.2% filter, 66.7% render, 76.2% snapshot)

### Data Flow

1. **Fetch**: Try JSON, fall back to XML, then CSV if needed
2. **Parse**: Decode into normalized `Event` structs
3. **Filter**:
   - Geographic: Haversine distance ≤ 0.35 km from Plaza de España (40.42338, -3.71217)
   - Temporal: Drop events in the past
   - Deduplication: By `ID-EVENTO` field
4. **Weather**: Fetch AEMET forecast, match to event dates
5. **Render**: Generate `index.html` and `events.json` in temp files
6. **Atomic write**: Rename temp files to final location (prevents partial updates)
7. **Snapshot**: Save successful fetch for future fallback

### Weather Integration (AEMET)

**Status:** ✅ Implemented (always enabled)

Enriches event cards with weather forecasts from AEMET (Spanish State Meteorological Agency):

**Architecture:**
- `internal/weather/` package: Client, types, matcher, icon utilities
- Two-step AEMET API fetch: metadata URL → forecast data URL
- Weather map: date string → Weather struct (temp, precip prob, sky icon)
- Template integration: conditional weather display on event cards

**Configuration:**
- `config.toml`: `[weather]` section (api_key_env, municipality_code)
- Environment variable: `AEMET_API_KEY` (register at https://opendata.aemet.es/)
- Weather fetch is required; build fails if API key missing or fetch fails

**Data displayed:**
- Max temperature for event date
- Precipitation probability (if >30%)
- AEMET official sky state icon
- Weather category for CSS styling (clear/cloudy/rain/etc)

**Icons:**
- Source: AEMET official PNG icons (~31 icons, ~1.3KB each)
- Storage: Committed to `generator/testdata/fixtures/aemet-icons/`
- Deployment: Copied to `public/assets/weather-icons/` during build
- License: Spain Law 18/2015 (open data, attribution required)

**Error handling:**
- If API key missing: logs error to stderr and exits with non-zero status
- If AEMET fetch fails: dumps full API response to stderr and exits with non-zero status
- If no forecast for event date: no weather shown for that event (not an error)

**API details:**
- Municipality: Madrid (code 28079)
- Forecast range: 7 days
- API key validity: Indefinite (since Sept 2017 policy change)
- Attribution: © AEMET (displayed in site footer)

### Robustness Strategy

- **Three-tier fallback**: JSON → XML → CSV
- **Last successful snapshot**: If all sources fail, serve previously cached data with "stale" indicator
- **Atomic writes**: Use temp files + rename to prevent serving incomplete output
- **Graceful degradation**: Missing fields (e.g., HORA) treated as all-day events
- **Timezone normalization**: All times parsed to Europe/Madrid

### Static Site Architecture - No JavaScript Required

**CRITICAL: This is a pure static site. All interactivity is CSS-only.**

**Why no JavaScript:**
1. **Security**: `.htaccess` enforces strict Content-Security-Policy that blocks JavaScript and inline styles
2. **Performance**: No JS = instant load, no runtime overhead
3. **Accessibility**: Works without JS enabled, on ancient browsers
4. **Simplicity**: All data is known at build time, no need for client-side computation

**How we achieve interactivity without JS:**
- **Filter toggles**: CSS-only using checkbox hack (`#toggle-cultural:checked ~ main`)
- **Section reordering**: Data attributes computed at build time + CSS `order` property
- **Distance filtering**: CSS display rules based on `data-at-plaza` attributes
- **Event counts**: Multiple count values embedded in template, CSS shows/hides relevant ones

**Example: Section Reordering**
```html
<!-- Template computes counts for all filter states at build time -->
<section data-count-plaza="5" data-count-nearby="10"
         data-count-plaza-city="3" data-count-nearby-city="7">
```

```css
/* CSS selects which count to check based on filter state */
#toggle-cultural:checked ~ #distance-plaza:checked ~ main
  .event-section[data-count-plaza="0"] {
  order: 999; /* Sink empty sections to bottom */
}
```

**Key principle**: Generate all possible state information at build time, use CSS selectors to show/hide based on user input. Never compute or manipulate data at runtime.

### Respectful Upstream Fetching

**Problem:** During development, we run builds 10-20+ times per hour to test changes. Without throttling, this looks like an attack to upstream servers (datos.madrid.es, esmadrid.com) and could get us blocked.

**Solution:** Comprehensive respectful fetching system with dual modes:

**Development Mode** (default, for frequent testing):
- **Cache TTL**: 1 hour (aggressive caching to minimize upstream hits)
- **Min delay**: 5 seconds between requests to same host
- **Rate limit**: 1 request per 5 minutes per URL
- **Logging**: `[development] Waiting 5s before fetching...`
- **Purpose**: Allows rapid testing without hitting upstream servers

**Production Mode** (for hourly cron):
- **Cache TTL**: 30 minutes (fresh data every cron run)
- **Min delay**: 2 seconds between requests to same host
- **Rate limit**: 1 request per hour per URL
- **Purpose**: Standard respectful behavior for automated systems

**Implementation:**
```
internal/fetch/
  mode.go       # ClientMode types, ProductionMode/DevelopmentMode configs
  cache.go      # HTTPCache with TTL, If-Modified-Since, atomic writes
  throttle.go   # RequestThrottle enforces per-host delays
  audit.go      # RequestAuditor tracks all HTTP requests
  client.go     # fetch() method integrates all respectful features
```

**Features:**
1. **HTTP Caching**:
   - Persistent cache stored in `data/http-cache/` (SHA256 filenames)
   - Uses `If-Modified-Since` headers to minimize bandwidth
   - Server returns 304 Not Modified → use cached data (no body transfer)
   - Cache hit: No HTTP request, instant return

2. **Request Throttling**:
   - Per-host minimum delays (tracks last request time per hostname)
   - Enforced via `time.Sleep()` in both fetch() and pipeline
   - User-visible logging: `[Pipeline] Waiting 5s before fetching next format (respectful delay)...`

3. **Rate Limit Detection**:
   - Detects 429 (Too Many Requests), 403 (Forbidden), 503 (Service Unavailable)
   - Marks requests as rate-limited in audit trail
   - Clear error messages if blocked

4. **Request Auditing**:
   - Every HTTP request tracked in `data/request-audit.json`
   - Records: URL, timestamp, cache hit, status code, delay, errors
   - Used for build reports and debugging

**Pipeline Integration:**
- Explicit delays between JSON → XML → CSV fetches
- Both pipeline sleep + fetch throttle (double protection)
- Clear logging: User always knows why build is slow

**Configuration:** (config.toml)
```toml
[fetch]
mode = "development"  # or "production"
cache_dir = "data/http-cache"
audit_path = "data/request-audit.json"
```

**Flag:** Use `-fetch-mode production` for cron jobs, `-fetch-mode development` (default) for testing.

**Result:** Safe to run `just dev` 20+ times during development without risk of getting blocked.

### Build Report

Every build generates an HTML report (`public/build-report.html`) with detailed metrics tracking both data pipelines:

**Dual Pipeline Architecture:**
- **Cultural Events Pipeline** (datos.madrid.es) - Purple accent 🎭
  - Fetches from 3 sources (JSON, XML, CSV)
  - Merges and deduplicates across sources
  - Filters by distrito (CENTRO, MONCLOA-ARAVACA) + GPS radius + time
  - Typically yields ~137 events

- **City Events Pipeline** (esmadrid.com) - Orange accent 🎉
  - Fetches from single XML source
  - Filters by GPS radius + category + time
  - Typically yields ~19 events

**Report Structure:**
```
internal/report/
  types.go      # BuildReport, PipelineReport, PipelineFetchReport, PipelineFilterReport
  html.go       # HTML report rendering with CSS-based styling
```

**Key Metrics Tracked:**
- Fetch attempts per source (with timing, status, error details)
- Merge/deduplication stats (total, unique, duplicates, source coverage)
- Distrito filtering (for cultural events)
- Geographic filtering (GPS radius, missing coords, outside radius)
- Time filtering (past events removed, parse failures)
- Category filtering (for city events, currently disabled)
- Pipeline durations and event counts

**Design Features:**
- Responsive grid layouts (auto-fit for mobile/desktop)
- Dark mode support via `prefers-color-scheme`
- Color-coded sections (purple for cultural, orange for city)
- Side-by-side pipeline overview cards

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
- **User-Agent header**: Set helpful User-Agent with contact info when fetching ✅ (`plazaespana-info-site-generator/1.0 (https://github.com/ericphanson/plazaespana.info)`)
- **Attribution**: Include Madrid open data attribution in rendered output ✅ (in HTML template footer)
- **Timezone**: All time operations must use Europe/Madrid (not UTC) ✅ (implemented in `filter/time.go`)

## Documentation

See `docs/README.md` for structure. Key files:
- `README.md` - Quick start guide
- `docs/deployment.md` - Deployment instructions
- `docs/plans/` - Dated implementation plans (archived)
- `docs/logs/` - Dated implementation logs (archived)

### README.md Policy

**IMPORTANT: The README.md is intentionally minimal and uses the author's voice.**

When updating README.md:
- ✅ **DO**: Make only the smallest tweaks required for accuracy (e.g., updating numbers, adding new data sources to lists)
- ❌ **DO NOT**: Reword or expand sections
- ❌ **DO NOT**: Add new sections (setup guides, detailed instructions, etc.)
- ❌ **DO NOT**: Change the author's casual tone or phrasing
- 🎯 **Preserve**: Original wording like "The site just collects this data and tries to render them cleanly"

**Why:** The README reflects the author's personality and minimalist philosophy. Detailed setup instructions belong in `config.toml` comments and `docs/deployment.md`.

**Example changes that ARE allowed:**
- "This is powered by two data feeds" → "This is powered by three data feeds" (factual accuracy)
- Adding "AEMET" to an existing list of data providers

**Example changes that are NOT allowed:**
- Rewriting intro paragraphs to be more "professional"
- Adding detailed setup instructions
- Expanding "See config.toml" into a multi-paragraph configuration guide

## Development Workflow (for Claude Code)

When executing implementation plans:

1. **Logging**: Create and maintain a log file in `docs/logs/` for each plan execution
   - One log per plan (e.g., `docs/logs/2025-10-19-madrid-events-implementation.md`)
   - Update the log after completing each subtask with status and notes
   - Track progress, issues encountered, and resolutions

2. **Commit workflow**: After each subtask:
   - **Format code**: Always run `gofmt -w .` before committing (CI will fail if code is not formatted)
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
