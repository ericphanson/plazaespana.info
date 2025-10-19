# Madrid Events Near Plaza de España

Static site generator that fetches events from Madrid's open data portal, filters to events near Plaza de España, and generates HTML/JSON output for deployment to NearlyFreeSpeech.NET.

## Quick Start

### Prerequisites

- Go 1.21+ (tested with 1.25.3)
- FreeBSD/amd64 target for deployment (cross-compiles from Linux/macOS)

### Install

```bash
# Clone repository
git clone https://github.com/ericphanson/madrid-events.git
cd madrid-events

# Build for local testing (Linux/macOS)
go build -o build/buildsite ./cmd/buildsite

# Build for FreeBSD deployment
./scripts/build-freebsd.sh
```

### Run Locally

```bash
# Generate site with real Madrid API data
./build/buildsite \
  -json-url https://datos.madrid.es/egob/catalogo/300107-0-agenda-actividades-eventos.json \
  -xml-url https://datos.madrid.es/egob/catalogo/300107-0-agenda-actividades-eventos.xml \
  -csv-url https://datos.madrid.es/egob/catalogo/300107-0-agenda-actividades-eventos.csv \
  -out-dir ./public \
  -data-dir ./data

# View generated files
open public/index.html
```

### Run Tests

```bash
# All tests
go test ./...

# Integration test
go test -tags=integration ./cmd/buildsite

# Specific package
go test -v ./internal/fetch
```

## Usage

### Command-Line Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-json-url` | *(required)* | Madrid events JSON URL |
| `-xml-url` | *(optional)* | XML fallback URL |
| `-csv-url` | *(optional)* | CSV fallback URL |
| `-out-dir` | `./public` | Output directory for HTML/JSON |
| `-data-dir` | `./data` | Data directory for snapshots |
| `-lat` | `40.42338` | Reference latitude (Plaza de España) |
| `-lon` | `-3.71217` | Reference longitude |
| `-radius-km` | `0.35` | Filter radius in kilometers |
| `-timezone` | `Europe/Madrid` | Timezone for event times |

### Output Files

- `public/index.html` - Main event listing page
- `public/events.json` - Machine-readable API
- `public/assets/site.<hash>.css` - Cached CSS
- `data/last_success.json` - Snapshot for fallback resilience

## Deploy to NearlyFreeSpeech.NET

### 1. Build FreeBSD Binary

```bash
./scripts/build-freebsd.sh
./scripts/hash-assets.sh
```

### 2. Upload Files

```bash
# Via SFTP
sftp username@ssh.phx.nearlyfreespeech.net

# Upload binary
put build/buildsite /home/bin/buildsite

# Upload template
put templates/index.tmpl.html /home/templates/index.tmpl.html

# Upload htaccess
put ops/htaccess /home/public/.htaccess

# Upload CSS
put public/assets/site.*.css /home/public/assets/
```

### 3. Set Permissions

```bash
ssh username@ssh.phx.nearlyfreespeech.net
chmod +x /home/bin/buildsite
mkdir -p /home/data /home/public/assets /home/templates
```

### 4. Configure Cron

In NFSN web UI → Scheduled Tasks, add:

**Command:**
```bash
/home/bin/buildsite -json-url https://datos.madrid.es/egob/catalogo/300107-0-agenda-actividades-eventos.json -xml-url https://datos.madrid.es/egob/catalogo/300107-0-agenda-actividades-eventos.xml -csv-url https://datos.madrid.es/egob/catalogo/300107-0-agenda-actividades-eventos.csv -out-dir /home/public -data-dir /home/data -lat 40.42338 -lon -3.71217 -radius-km 0.35 -timezone Europe/Madrid
```

**Schedule:** Every hour (or `*/10` for 10-minute intervals)

## How It Works

1. **Fetch** - Tries JSON → XML → CSV from Madrid's API with fallback chain
2. **Filter** - Geographic (Haversine distance ≤ 0.35 km) + temporal (future events only)
3. **Deduplicate** - By `ID-EVENTO` field
4. **Render** - HTML template + JSON API with atomic writes
5. **Snapshot** - Saves successful fetch for fallback when API is down

## Features

- ✅ Three-tier data fallback (JSON → XML → CSV)
- ✅ Snapshot resilience (serves cached data if API fails)
- ✅ Geographic filtering (Haversine distance calculation)
- ✅ Timezone-aware (Europe/Madrid)
- ✅ Atomic file writes (no partial updates)
- ✅ Content-hashed CSS for cache busting
- ✅ Static binary (no dependencies)

## Development

```bash
# Run tests on file change
go test ./... -v

# Check test coverage
go test -cover ./...

# Format code
go fmt ./...

# Run linter
go vet ./...
```

See [`docs/design.md`](docs/design.md) for detailed architecture and design documentation.

## License

Data source: [Ayuntamiento de Madrid – datos.madrid.es](https://datos.madrid.es)
Attribution required per Madrid's open data license.
