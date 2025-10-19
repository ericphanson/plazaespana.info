# Madrid Events Near Plaza de España

Static site generator that fetches events from Madrid's open data portal, filters to events near Plaza de España, and generates HTML/JSON output for deployment to NearlyFreeSpeech.NET.

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

This will fetch real Madrid event data, filter to Plaza de España area, and serve the site locally.

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

## Output

The site generator produces:
- `public/index.html` - Main event listing page
- `public/events.json` - Machine-readable API
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

Fetches events from Madrid's open data API → Filters by location (Plaza de España ±350m) and time → Generates static HTML/JSON → Caches for resilience.

**Features:**
- Three-tier fallback (JSON → XML → CSV)
- Snapshot resilience when API is down
- Haversine geographic filtering
- Timezone-aware (Europe/Madrid)
- Atomic file writes
- Zero dependencies

See [`docs/design.md`](docs/design.md) for architecture details.

## License

Data source: [Ayuntamiento de Madrid – datos.madrid.es](https://datos.madrid.es)
Attribution required per Madrid's open data license.
