# Deployment to NearlyFreeSpeech.NET

**⚠️ This document is for reference only.**

**For current deployment instructions, see:** [`../docs/deployment.md`](../docs/deployment.md)

The new deployment guide covers:
- `just deploy` command for automated deployment
- GitHub Actions automatic deployment on push to `main`
- SSH key setup for both local and CI/CD
- Complete troubleshooting guide

---

## Legacy Manual Deployment Instructions

The following instructions are kept for reference but are superseded by `just deploy`.

## Initial Setup

1. **Build FreeBSD binary locally:**
   ```bash
   ./scripts/build-freebsd.sh
   ```

2. **Upload via SFTP:**
   ```bash
   sftp username@ssh.phx.nearlyfreespeech.net
   put build/buildsite /home/bin/buildsite
   put config.toml /home/config.toml
   put templates/index.tmpl.html /home/templates/index.tmpl.html
   put ops/htaccess /home/public/.htaccess
   ```

3. **Set permissions:**
   ```bash
   ssh username@ssh.phx.nearlyfreespeech.net
   chmod +x /home/bin/buildsite
   mkdir -p /home/data /home/public/assets /home/templates
   ```

4. **Configure cron (Scheduled Tasks in NFSN web UI):**
   - **Command:** `/home/private/bin/buildsite -config /home/private/config.toml -out-dir /home/public -data-dir /home/private/data -fetch-mode production`
   - **Schedule:** Every hour (or `*/10` for 10-minute intervals)
   - **Flags explained:**
     - `-config /home/private/config.toml` - Use uploaded config
     - `-out-dir /home/public` - Override output path to web root
     - `-data-dir /home/private/data` - Override data path (in private/)
     - `-fetch-mode production` - Production fetch settings (30min cache, 2s delays)

## Configuration File

The `config.toml` file contains all site configuration:

```toml
[cultural_events]
# datos.madrid.es cultural programming
json_url = "https://datos.madrid.es/egob/catalogo/300107-0-agenda-actividades-eventos.json"
xml_url = "https://datos.madrid.es/egob/catalogo/300107-0-agenda-actividades-eventos.xml"
csv_url = "https://datos.madrid.es/egob/catalogo/300107-0-agenda-actividades-eventos.csv"

[city_events]
# esmadrid.com tourism/city events
xml_url = "https://www.esmadrid.com/opendata/agenda_v1_es.xml"

[filter]
# Plaza de España coordinates
latitude = 40.42338
longitude = -3.71217
radius_km = 0.35

# Distrito filtering
distritos = ["CENTRO", "MONCLOA-ARAVACA"]

# Time filtering
past_events_weeks = 2  # Exclude events started >2 weeks ago

[output]
html_path = "public/index.html"
json_path = "public/events.json"

[snapshot]
data_dir = "data"

[fetch]
# Respectful upstream fetching
mode = "production"  # Use production mode for cron (30min cache, 2s delays)
cache_dir = "data/http-cache"
audit_path = "data/request-audit.json"

[server]
# For development only
port = 8080
```

**Validate config before deploying:**
```bash
just build
just config
```

## Respectful Upstream Fetching

The site implements comprehensive respectful fetching to prevent overwhelming upstream servers (datos.madrid.es, esmadrid.com).

### Production Mode (for Cron)

**IMPORTANT:** Always use `-fetch-mode production` in cron jobs.

**Settings:**
- **Cache TTL:** 30 minutes (fresh data)
- **Min delay:** 2 seconds between requests to same host
- **Rate limit:** 1 request per hour per URL
- **Purpose:** Standard respectful behavior for automated systems

**Cron Command:**
```bash
/home/private/bin/buildsite -config /home/private/config.toml -out-dir /home/public -data-dir /home/private/data -fetch-mode production
```

### Features

1. **HTTP Caching:**
   - Persistent cache in `/home/data/http-cache/`
   - Uses `If-Modified-Since` headers to minimize bandwidth
   - Server returns 304 Not Modified → uses cached data (no body transfer)

2. **Request Throttling:**
   - Per-host delays prevent rapid-fire requests
   - Enforced delays between JSON → XML → CSV fetches

3. **Rate Limit Detection:**
   - Detects 429/403/503 status codes
   - Clear error logging if blocked

4. **Request Auditing:**
   - All HTTP requests logged to `/home/data/request-audit.json`
   - Useful for debugging upstream issues

### Directory Structure

After first run, data directory contains:
```
/home/data/
  http-cache/           # Persistent HTTP cache (auto-created)
    <sha256>.json       # Cached responses
  request-audit.json    # HTTP request log
  last_success.json     # Snapshot fallback
  audit-events.json     # Event audit trail
  build-report.html     # Build metrics
```

**Important:** All cache and audit files are automatically managed. No manual cleanup needed.

## Updates

1. Build new binary: `./scripts/build-freebsd.sh`
2. Upload binary and config:
   ```bash
   sftp username@ssh.phx.nearlyfreespeech.net
   put build/buildsite /home/bin/buildsite
   put config.toml /home/config.toml
   ```
3. Binary will be used on next cron run
