[![CI](https://github.com/ericphanson/plaza-espana-calendar/actions/workflows/ci.yml/badge.svg)](https://github.com/ericphanson/plaza-espana-calendar/actions/workflows/ci.yml)

# Madrid Events Near Plaza de España

**Generates a static website showing upcoming events near Plaza de España in Madrid.**

Fetches from two sources:
- **Cultural events** from datos.madrid.es (theater, museums, exhibitions)
- **City events** from esmadrid.com (tourism, festivals, outdoor activities)

**Live site:** [plazaespana.info](https://plazaespana.info) (also at [plazaespana.nfshost.com](https://plazaespana.nfshost.com))

## Quick Start (3 Steps)

### 1. Install Prerequisites

**You need:**
- **Go 1.21+** - [Install Go](https://go.dev/doc/install)
- **just** (command runner) - Install with:
  ```bash
  curl --proto '=https' --tlsv1.2 -sSf https://just.systems/install.sh | bash -s -- --to /usr/local/bin
  ```

### 2. Clone and Setup

```bash
git clone https://github.com/ericphanson/madrid-events.git
cd madrid-events
```

### 3. Run It!

```bash
just dev
```

**That's it!** Opens automatically at http://localhost:8080

The site will:
- ✅ Fetch events from Madrid APIs (takes 10-15 seconds first time)
- ✅ Filter to events within 350m of Plaza de España
- ✅ Show combined cultural + city events
- ✅ Cache data for 1 hour (fast subsequent builds)

**Press Ctrl+C to stop the server.**

## Common Tasks

### Just Want to See the Site?

```bash
just dev
# Opens at http://localhost:8080
```

### Test Your Changes?

```bash
just test
# Runs all tests (takes ~60 seconds due to throttling)
```

### Need Fresh Data?

```bash
just clean   # Remove cached data
just dev     # Rebuild from scratch
```

### Deploy to Production?

```bash
just freebsd         # Build for FreeBSD
just hash-css        # Generate hashed CSS
# Then upload - see ops/deploy-notes.md
```

## All Available Commands

Run `just` to see this help menu:

```bash
🚀 Getting Started:
  just dev          - Build site and serve locally
  just test         - Run all tests

🔨 Build Commands:
  just build        - Build binary for local use
  just freebsd      - Build for FreeBSD (deployment)
  just hash-css     - Generate content-hashed CSS

🌐 Development:
  just serve        - Serve existing site (faster)
  just kill         - Stop server

🧹 Maintenance:
  just clean        - Remove build artifacts
  just fmt          - Format Go code
  just lint         - Run linter

📝 Configuration:
  just config       - Validate config.toml
```

**See [`justfile`](justfile) for all commands.**

## Configuration (Optional)

**By default, `just dev` works out of the box** using `config.toml`.

### Want to Customize?

Edit `config.toml` to change:
- **Location** - Change from Plaza de España to another area
- **Radius** - Adjust search radius (default: 350m)
- **Districts** - Filter by Madrid districts

Example:
```toml
[filter]
latitude = 40.41682    # Puerta del Sol instead
longitude = -3.70379
radius_km = 0.5        # Wider area
```

### Advanced: CLI Flags

You can override config settings with flags:

```bash
./build/buildsite -config config.toml -radius-km 1.0
```

See all flags with:
```bash
./build/buildsite -help
```

## What Gets Generated?

After running `just dev`, you'll find:

```
public/
  index.html              - Main event listing (view in browser)
  events.json             - API with all event data
  assets/site.*.css       - Styled CSS

data/
  http-cache/             - Cached API responses (auto-managed)
  request-audit.json      - Request log (for debugging)
  build-report.html       - Build metrics and stats
```

**View the site:** Open http://localhost:8080 in your browser

## Deploy to Web Hosting

### For NearlyFreeSpeech.NET (FreeBSD)

```bash
# One-time setup: Configure credentials with direnv
cp .envrc.local.example .envrc.local
# Edit .envrc.local with your NFSN credentials
direnv allow

# Deploy (builds, uploads, and regenerates site)
just deploy
```

**Automatic deployment:** GitHub Actions deploys automatically on push to `main`.

**Complete deployment guide:** See [`docs/deployment.md`](docs/deployment.md) for:
- SSH key setup for local and GitHub Actions deployment
- Required GitHub Secrets configuration
- Cron job setup on NFSN
- Troubleshooting tips

## How It Works

**Simple 3-step process:**

1. **Fetch Events**
   - Gets cultural events from datos.madrid.es (JSON/XML/CSV)
   - Gets city events from esmadrid.com (XML)
   - Has fallbacks if one source fails

2. **Filter to Area**
   - Keeps only events near Plaza de España (350m radius)
   - Also filters by districts: CENTRO, MONCLOA-ARAVACA
   - Removes past events

3. **Generate Site**
   - Creates HTML page with all events
   - Creates JSON API for programmatic access
   - Saves snapshot for offline fallback

**Why it's robust:**
- ✅ Works even if Madrid APIs are slow or down
- ✅ Multiple data sources (JSON, XML, CSV)
- ✅ Caches data to avoid repeated API calls
- ✅ Respects upstream servers (won't get blocked)

**Technical details:** See [`docs/design.md`](docs/design.md)

## Why Builds Might Be Slow (and That's Good!)

**TL;DR: The site is intentionally respectful to Madrid's servers.**

### What You'll Notice

When you run `just dev`, you might see:
```
[Pipeline] Fetching JSON from datos.madrid.es...
[Pipeline] JSON: 1055 events, 0 errors
[Pipeline] Waiting 5s before fetching next format (respectful delay)...
```

**This is intentional!** Here's why:

### The Problem We're Solving

During development, you might run `just dev` 10-20 times per hour to test changes. Without delays:
- ❌ Madrid's servers see 60+ rapid requests from the same IP
- ❌ Looks like an attack or bot scraping
- ❌ You could get IP banned

### The Solution: Smart Caching + Delays

**For Development (your local testing):**
- ✅ Caches data for **1 hour** (super fast subsequent builds!)
- ✅ Waits **5 seconds** between requests (polite to servers)
- ✅ Shows clear logging so you know what's happening

**First build:** ~15 seconds (fetches fresh data)
**Subsequent builds:** ~instant (uses cached data)

**For Production (deployed to web):**
- ✅ Caches data for **30 minutes** (fresher data)
- ✅ Waits **2 seconds** between requests
- ✅ Runs once per hour via cron (not rapid-fire)

### How to Use It

**For local development:**
```bash
just dev
# Automatically uses development mode
# Caches for 1 hour - perfect for testing!
```

**For production deployment:**
```bash
# In your cron job, add this flag:
/home/bin/buildsite -config /home/config.toml -fetch-mode production
```

### Want Fresh Data Right Now?

```bash
just clean   # Delete cache
just dev     # Fetch fresh data (takes ~15 seconds)
```

### The Bottom Line

- 🚀 **First build each hour:** Takes 15 seconds (fetching from Madrid)
- ⚡ **Subsequent builds:** Instant (uses cache)
- 🤝 **We stay respectful:** Madrid's servers stay happy, you don't get blocked

**This design lets you test rapidly without being a bad internet citizen!**

## Troubleshooting

### Build is Slow

**Expected!** First build takes ~15 seconds to fetch data from Madrid. Subsequent builds within the same hour are instant (uses cache).

**Want it faster?** You don't need to rebuild every time - just edit HTML/CSS and refresh your browser!

### Server Won't Start

**Error:** `Address already in use`

**Solution:**
```bash
just kill   # Stop any running server
just dev    # Start fresh
```

### Need Fresh Data

Cache is stale? Clear it:
```bash
just clean   # Remove all cached data
just dev     # Rebuild from scratch
```

### Tests Taking Forever

Tests take ~60 seconds because they verify the 5-second delays work correctly. **This is expected!**

Fast test run (skips delay tests):
```bash
go test ./internal/fetch -short
```

### Can't Connect to Madrid APIs

If you see errors fetching data:
1. **Check your internet connection**
2. **Try again in a few minutes** (APIs might be temporarily down)
3. **Site will use cached data** if available (no build failure!)

### Port 8080 Already in Use

Change the port in `config.toml`:
```toml
[server]
port = 3000  # Or any other port
```

Then:
```bash
just dev  # Now uses port 3000
```

## Contributing

Found a bug? Have a feature idea?

1. **Open an issue** describing the problem/idea
2. **Fork the repo** and make your changes
3. **Run tests:** `just test`
4. **Submit a pull request**

**Before submitting:**
- ✅ Run `just test` (all tests must pass)
- ✅ Run `just fmt` (format code)
- ✅ Run `just lint` (check for issues)

## License

Data source: [Ayuntamiento de Madrid – datos.madrid.es](https://datos.madrid.es)
Attribution required per Madrid's open data license.
