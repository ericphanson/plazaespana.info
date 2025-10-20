# Madrid Events Near Plaza de España

Static site generator with **dual pipeline support**: Fetches cultural events from Madrid's open data portal (datos.madrid.es) and city events from ESMadrid tourism portal (esmadrid.com), filters to events near Plaza de España, and generates HTML/JSON output for deployment to NearlyFreeSpeech.NET.

## Quick Start

### Prerequisites

- Go 1.21+ (tested with 1.25.3)
- [just](https://github.com/casey/just) - command runner (install: `curl --proto '=https' --tlsv1.2 -sSf https://just.systems/install.sh | bash -s -- --to /usr/local/bin`)

### Install

```bash
git clone https://github.com/ericphanson/madrid-events.git
cd madrid-events
just build
```

### Run Locally

```bash
# Build site and start local dev server at http://localhost:8080
just dev
```

This will:
1. Fetch cultural events from datos.madrid.es (JSON/XML/CSV fallback)
2. Fetch city events from esmadrid.com (XML)
3. Filter both to Plaza de España area
4. Serve the combined site locally

**Configuration**: Uses `config.toml` (see `config.toml.example` for all options)

### Run Tests

```bash
just test              # All tests
just test-coverage     # With coverage
just test-integration  # Integration tests
```

## Available Commands

```bash
just           # List all available tasks
just dev       # Build and serve locally at :8080
just build     # Build binary
just test      # Run tests
just freebsd   # Cross-compile for FreeBSD
just clean     # Clean build artifacts
just fmt       # Format code
just lint      # Run linter
```

See [`justfile`](justfile) for all commands and manual equivalents.

## Configuration

### Using TOML Config File (Recommended)

Copy `config.toml.example` to `config.toml` and customize:

```bash
cp config.toml.example config.toml
# Edit config.toml to set your preferences
./build/buildsite -config config.toml
```

### Using CLI Flags

Override individual settings without a config file:

```bash
./build/buildsite \
  -json-url https://datos.madrid.es/egob/catalogo/300107-0-agenda-actividades-eventos.json \
  -xml-url https://datos.madrid.es/egob/catalogo/300107-0-agenda-actividades-eventos.xml \
  -csv-url https://datos.madrid.es/egob/catalogo/300107-0-agenda-actividades-eventos.csv \
  -esmadrid-url https://www.esmadrid.com/opendata/agenda_v1_es.xml \
  -out-dir ./public \
  -data-dir ./data \
  -lat 40.42338 -lon -3.71217 -radius-km 0.35
```

### Mixed Mode

Use config file + CLI flags to override specific settings:

```bash
# Use config.toml but override output directory
./build/buildsite -config config.toml -out-dir /custom/path
```

## Output

The site generator produces:
- `public/index.html` - Main event listing page (combined cultural + city events)
- `public/events.json` - Machine-readable API with separated event types:
  ```json
  {
    "cultural_events": [...],  // datos.madrid.es
    "city_events": [...],       // esmadrid.com
    "last_updated": "2025-10-20T12:00:00Z"
  }
  ```
- `public/assets/site.<hash>.css` - Content-hashed CSS
- `data/last_success.json` - Snapshot for fallback resilience

## Deploy to NearlyFreeSpeech.NET

### Build

```bash
just freebsd   # Cross-compile for FreeBSD
just hash-css  # Generate content-hashed CSS
```

### Upload & Configure

See [`ops/deploy-notes.md`](ops/deploy-notes.md) for complete deployment instructions including:
- SFTP file upload
- Permission setup
- Cron configuration (hourly regeneration)
- Apache caching rules

## How It Works

**Dual Pipeline Architecture:**

1. **Cultural Events Pipeline** (datos.madrid.es)
   - Three-tier fallback: JSON → XML → CSV
   - Distrito-based filtering (primary) + GPS radius fallback
   - Filters events in CENTRO and MONCLOA-ARAVACA districts

2. **City Events Pipeline** (esmadrid.com)
   - Fetches tourism/city events from ESMadrid XML
   - GPS radius filtering (Plaza de España ±350m)
   - Complementary to cultural events

3. **Rendering**
   - Merges both event types
   - Generates combined HTML and separated JSON output
   - Atomic file writes for reliability

**Features:**
- Dual data sources for comprehensive coverage
- Three-tier fallback for cultural events (JSON → XML → CSV)
- Distrito-based + GPS radius filtering
- Snapshot resilience when API is down
- Timezone-aware (Europe/Madrid)
- Zero external dependencies

**Configuration:**
- TOML config file (see `config.toml.example`)
- CLI flags override config settings
- Full backward compatibility

See [`docs/design.md`](docs/design.md) for architecture details.

## License

Data source: [Ayuntamiento de Madrid – datos.madrid.es](https://datos.madrid.es)
Attribution required per Madrid's open data license.
