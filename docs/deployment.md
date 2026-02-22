# Deployment Guide

> **⚠️ SECURITY NOTICE**
>
> This deployment guide is specific to the original author's hosting environment (NearlyFreeSpeech.NET) and serves as an **example implementation**. Do not blindly copy these configurations to production environments.
>
> **Before deploying:**
> - Adapt paths, hostnames, and security settings to your infrastructure
> - Review all credentials and secrets management practices
> - Consider this a learning resource, not production-ready configuration
> - Implement additional security hardening appropriate for your threat model
>
> **Privacy Note:** This project is designed for public event listings only. No user authentication, personal data collection, or sensitive information handling is implemented. If you adapt this for other purposes, ensure appropriate privacy and security controls.

This guide covers deploying the Madrid Events site to NearlyFreeSpeech.NET (NFSN).

## Prerequisites

- SSH access to your NFSN account
- SSH key pair for authentication
- **direnv** (recommended) - [Install direnv](https://direnv.net/docs/installation.html)
- NFS host has `python3` and `curl` available (required by analytics publish/backup script)

## Quick Start

### 1. Setup Credentials (One Time)

**Using direnv (recommended):**

```bash
# Copy and edit credentials
cp .envrc.local.example .envrc.local
# Edit .envrc.local with your NFSN_HOST and NFSN_USER

# Enable direnv
direnv allow
```

Variables will auto-load when you `cd` into the project. Why direnv? Gitignored credentials, no shell pollution, per-project config.
The repo's `.envrc` loads `.envrc.local`.

**Alternative (manual):**
```bash
export NFSN_HOST=ssh.phx.nearlyfreespeech.net
export NFSN_USER=your_username
```

### 2. Setup SSH Key

**Generate key (if needed):**
```bash
ssh-keygen -t ed25519 -C "your_email@example.com"
```

**Add public key to NFSN:**
- NFSN web interface → Profile → SSH/SFTP Keys
- Upload `~/.ssh/id_ed25519.pub`

**Test connection:**
```bash
ssh your_username@ssh.phx.nearlyfreespeech.net
```

### 3. Deploy

**Local:**
```bash
just deploy
```

`just deploy` now fails early with actionable errors if common prerequisites are missing:
- `build/buildsite` missing (`just freebsd`)
- `log-analyzer/build/log-analyzer-freebsd` missing (`just log-analyzer-freebsd`)
- hashed CSS artifacts missing (`just hash-css`)

**Automatic:** GitHub Actions deploys on push to `main` (after tests pass).

### 4. Run Initial Analytics Snapshot (One Time)

Run this once after the first successful deploy so analytics files exist before cron:

```bash
ssh $NFSN_USER@$NFSN_HOST
/home/private/bin/log-analyzer-daily.sh

# Verify outputs
ls -lh /home/private/log-analyzer-data/
tail -50 /home/logs/log-analyzer.log
```

Expected output:
- `/home/private/log-analyzer-data/lifetime.json`
- `/home/private/log-analyzer-data/YYYY-MM.json` files
- `/home/private/log-analyzer-data/report.html`

## GitHub Actions Setup

Add these secrets in repository Settings → Secrets and variables → Actions:

| Secret Name               | Description                          | Value                               | Required |
|---------------------------|--------------------------------------|-------------------------------------|----------|
| `NFSN_SSH_KEY`            | Private SSH key                      | Contents of `~/.ssh/id_ed25519`     | Yes      |
| `NFSN_HOST`               | NFSN SSH hostname                    | SSH hostname from site information  | Yes      |
| `NFSN_USER`               | NFSN username                        | `your_username`                     | Yes      |
| `NFSN_KNOWN_HOST`         | SSH host key (preview/cleanup jobs)  | Output from `ssh-keyscan -H`        | Yes      |
| `AEMET_API_KEY`           | AEMET weather API key                | Your AEMET OpenData API key         | Yes      |
| `BUNNY_STORAGE_KEY`       | Bunny storage API key                | Bunny storage key                   | Recommended |
| `BUNNY_STORAGE_ZONE`      | Bunny storage zone                   | e.g. `my-analytics-zone`            | Recommended |
| `BUNNY_STORAGE_ENDPOINT`  | Bunny storage endpoint host          | e.g. `storage.bunnycdn.com`         | Recommended |
| `BUNNY_BASE_PATH`         | Bunny backup path prefix             | default: `analytics-backup/current` | Optional |

⚠️ Use the **private key** that matches the public key uploaded to NFSN.

**Note:** `AEMET_API_KEY` is required. Without it, site builds will fail.

## Cron Setup on NFSN (Site + Analytics)

After first deployment, configure both scheduled tasks:

1. NFSN web interface → Sites → your_site → Scheduled Tasks
2. Add site generation task:
   - **Command:** `/home/private/bin/cron-generate.sh`
   - **Schedule:** Every hour (or `0 * * * *`)
3. Add analytics task:
   - **Command:** `/home/private/bin/log-analyzer-daily.sh`
   - **Schedule:** `15 1 * * *` (daily at 01:15)
   - **Tag:** `log-analyzer-daily` (optional)

Site wrapper (`cron-generate.sh`):
- Logs all output to `/home/logs/generate.log` with timestamps
- Sends cron email only on failures with a concise error summary
- Includes fallback log tail and full logfile path for debugging

Analytics wrapper (`log-analyzer-daily.sh`):
- Reads `/home/logs/access_log*`
- Writes snapshots to `/home/private/log-analyzer-data`
- Publishes `/home/public/analytics_report.html`
- Logs to `/home/logs/log-analyzer.log`
- Uses a lock directory to prevent overlapping runs and purges stale temp dirs
- Deployment also creates `/home/private/bin/log-analyzer-weekly.sh` as a compatibility symlink to the daily script

Configure Bunny backup (recommended):
- Set `BUNNY_STORAGE_KEY`, `BUNNY_STORAGE_ZONE`, `BUNNY_STORAGE_ENDPOINT`
- Optional `BUNNY_BASE_PATH` defaults to `analytics-backup/current`
- Run `just deploy` after setting them (deployment writes `/home/private/bunny-*.txt`, mode `600`)
- Validate credentials before deploy with `just bunny-check` (performs a write/read/delete smoke test under `.../_healthchecks/`)

Runtime knobs for analytics job:
- `MONTH_CLOSE_GRACE_DAYS` (default `7`)
- `BUNNY_BACKUP_REQUIRED` (default `1`)
- `LOG_ANALYZER_TMP_DIR_BASE` (default `/tmp`)
- `LOG_ANALYZER_KEEP_TMP_ON_FAILURE` (`1` preserves temp dirs for debugging)
- `BUNNY_CURL_CONNECT_TIMEOUT_SEC` (default `10`)
- `BUNNY_CURL_MAX_TIME_SEC` (default `120`)
- `BUNNY_CURL_RETRIES` (default `3`)
- `BUNNY_CURL_RETRY_DELAY_SEC` (default `2`)
- `BUNNY_CURL_RETRY_MAX_TIME_SEC` (default `180`)

Configure NFSN log rotation for analytics quality:
- Rotate weekly
- Keep at least 4-8 weeks
- Enable compression

Stale-data alerting:
- Workflow: `.github/workflows/check-analytics-stale.yml`
- Checks `https://plazaespana.info/analytics_report.html` and enforces report `Generated:` age <= 3 days
- Local check: `just check-analytics-stale`

**View logs:**
```bash
ssh your_username@ssh.phx.nearlyfreespeech.net
tail -f /home/logs/generate.log
tail -f /home/logs/log-analyzer.log
```

## AEMET Weather API Setup

The site integrates weather forecasts from AEMET (Spanish Meteorological Agency). Weather data is required for site builds.

### Get an API Key

1. Register at: https://opendata.aemet.es/centrodedescargas/altaUsuario
2. Wait for email with API key (usually instant)
3. API keys have indefinite validity (no expiration)

### Configure for Production (NFSN)

The AEMET API key is stored in a secure file on NFSN (`/home/private/aemet-api-key.txt`).

**Setup (one-time):**

1. Set the `AEMET_API_KEY` environment variable locally:
   ```bash
   export AEMET_API_KEY=your_aemet_api_key_here
   ```

2. Deploy the site:
   ```bash
   just deploy
   ```

   The deployment automatically:
   - Reads `AEMET_API_KEY` from your local environment
   - Writes it to a temporary file (`build/aemet-api-key.txt`)
   - Uploads the file to `/home/private/aemet-api-key.txt` on NFSN
   - Sets secure permissions (600)
   - Cleans up the local temporary file

3. Your cron job will automatically read the key from the file (no changes needed)

**To update the key:**
1. Update your local `AEMET_API_KEY` environment variable
2. Run `just deploy` again
3. The new key is automatically uploaded and used on next site generation

**Security benefits:**
- API key stored in `/home/private` (not web-accessible)
- Not visible in NFSN cron command interface
- File permissions set to 600 (owner read/write only)
- No need to edit cron jobs to change the key

### Configure for GitHub Actions

For automated deployments and PR previews, add the API key to GitHub repository secrets:

1. Repository Settings → Secrets and variables → Actions → New repository secret
2. Name: `AEMET_API_KEY`
3. Value: (paste your AEMET API key)

The GitHub Actions workflows are already configured to use this secret.

### Error Handling

**If API key is missing or invalid:**
- Build fails with non-zero exit code
- Full API response dumped to stderr for debugging
- Site continues to serve last successful build

**Check weather status:**
- View `/home/public/build-report.html` for weather fetch details
- Check `/home/private/data/request-audit.json` for AEMET API requests

## What Gets Deployed

Files uploaded to NFSN:

```
Local → Remote

# Site generation
build/buildsite                      → /home/private/bin/buildsite
ops/cron-generate.sh                 → /home/private/bin/cron-generate.sh
config.toml                          → /home/private/config.toml
$AEMET_API_KEY (env)                 → /home/private/aemet-api-key.txt (if set)
generator/templates/index.tmpl.html  → /home/private/templates/index.tmpl.html

# Static assets
public/assets/site.*.css             → /home/public/assets/
public/assets/build-report.*.css     → /home/public/assets/
public/assets/*.hash                 → /home/public/assets/
public/assets/weather-icons/*.png    → /home/public/assets/weather-icons/
ops/htaccess                         → /home/public/.htaccess

# Log analyzer
log-analyzer/build/log-analyzer-freebsd → /home/private/bin/log-analyzer
ops/log-analyzer-daily.sh               → /home/private/bin/log-analyzer-daily.sh
ops/log-analyzer-publish.py             → /home/private/bin/log-analyzer-publish.py
$BUNNY_* (env, optional)                → /home/private/bunny-*.txt
```

After upload, binary runs to generate:
- `/home/public/index.html` - Event listing (web-accessible)
- `/home/public/events.json` - JSON API (web-accessible)
- `/home/private/data/` - Cache & audit logs (not web-accessible)

Log analyzer generates (via cron):
- `/home/private/log-analyzer-data/` - canonical aggregate monthly JSON + lifetime JSON + report HTML
- `/home/public/analytics_report.html` - public analytics report
- Bunny mirror at `analytics-backup/current/` (JSON backup; immutable old month files by content)

Notes:
- `lifetime.json` is rebuilt from persisted month files and remains stable across log rotation.
- `report.html` is generated from persisted JSON files, so historical months remain represented after log rotation.

## NFSN Directory Structure

```
/home/
  private/              # ❌ Not web-accessible
    bin/
      buildsite         # Site generator binary
      cron-generate.sh  # Site generation wrapper (hourly cron)
      log-analyzer      # Log analyzer binary (FreeBSD)
      log-analyzer-daily.sh # Aggregate analytics snapshot job (cron)
      log-analyzer-publish.py # Publish + privacy + backup wrapper
    config.toml         # Site generator config
    aemet-api-key.txt   # AEMET API key (optional, mode 600)
    bunny-storage-key.txt      # Bunny storage key (optional, mode 600)
    bunny-storage-zone.txt     # Bunny storage zone (optional, mode 600)
    bunny-storage-endpoint.txt # Bunny endpoint host (optional, mode 600)
    bunny-base-path.txt        # Bunny base path (optional, mode 600)
    templates/          # HTML templates
    data/               # Site generator cache, audit logs (auto-created)
    log-analyzer-data/  # Canonical aggregate analytics snapshots

  public/               # ✅ Web root (served via HTTP)
    index.html          # Generated event listing
    events.json         # Generated JSON API
    analytics_report.html # Analytics report
    assets/             # CSS files and weather icons
      site.*.css        # Hashed main site CSS
      build-report.*.css # Hashed build report CSS
      weather-icons/    # AEMET weather icons (PNG)
    .htaccess           # Apache config (caching, security headers)

  logs/                 # Log files
    access_log          # Apache access log (NFSN rotates automatically)
    generate.log        # Site generation log
    log-analyzer.log    # Log analyzer processing log
```

**Access control:**
- Only `/home/public/` is web-accessible via HTTP/HTTPS
- All other files (`/home/private/`, `/home/logs/`) are SSH-only

## Troubleshooting

### Permission denied (SSH)

Add your SSH public key: NFSN web interface → Profile → SSH/SFTP Keys

### Host key verification failed

```bash
ssh-keygen -R ssh.phx.nearlyfreespeech.net
ssh-keyscan -H ssh.phx.nearlyfreespeech.net >> ~/.ssh/known_hosts
```

### GitHub Actions deployment fails

1. Verify required secrets are set in repository settings (`NFSN_SSH_KEY`, `NFSN_HOST`, `NFSN_USER`, `AEMET_API_KEY`)
2. Ensure `NFSN_SSH_KEY` private key matches public key on NFSN
3. Check GitHub Actions logs for specific errors

### Site not updating

**Check the logs:**
```bash
ssh your_username@ssh.phx.nearlyfreespeech.net
tail -100 /home/logs/generate.log
```

**Run manually to debug:**
```bash
ssh your_username@ssh.phx.nearlyfreespeech.net
/home/private/bin/buildsite -config /home/private/config.toml -out-dir /home/public -data-dir /home/private/data -template-path /home/private/templates/index.tmpl.html -fetch-mode production
```

### Log analyzer output missing or stale

**Causes:**
1. Access log is empty (no traffic yet)
2. Log-analyzer cron hasn't run yet
3. Wrapper script failed

**Fix:**
```bash
# Check if logs exist
ls -lh /home/logs/access_log

# Check if aggregate output exists
ls -lh /home/private/log-analyzer-data/

# Manually process current logs
/home/private/bin/log-analyzer-daily.sh

# Inspect logs
tail -100 /home/logs/log-analyzer.log
```

### Bunny backup sync fails

**Common causes:**
1. Missing `/home/private/bunny-*.txt` files
2. Invalid Bunny storage key / zone / endpoint
3. Immutable month mismatch (historical JSON changed unexpectedly)
4. Missing immutable month object on Bunny (script will re-upload canonical local month file)

**Fix:**
```bash
# Check Bunny config on NFS
ssh "$NFSN_USER@$NFSN_HOST" 'ls -l /home/private/bunny-*.txt'

# Run analytics job manually and inspect logs
ssh "$NFSN_USER@$NFSN_HOST" '/home/private/bin/log-analyzer-daily.sh'
ssh "$NFSN_USER@$NFSN_HOST" 'tail -100 /home/logs/log-analyzer.log'
```

## Security

- Never commit private keys to the repository
- Private keys belong in `~/.ssh/` (local) and GitHub Secrets (CI)
- Public keys are safe to share (uploaded to NFSN)
- Use `ed25519` keys (more secure than RSA)
- Keep Bunny credentials in `/home/private` with mode `600` and rotate periodically

## Deployment Checklist

**Before first deployment:**
- [ ] Tests pass (`just test`)
- [ ] Binary builds (`just freebsd`)
- [ ] SSH key added to NFSN
- [ ] Credentials configured (direnv or secrets)
- [ ] AEMET API key obtained and configured (required)

**After first deployment:**
- [ ] Visit NFSN site URL to verify site works
- [ ] Check events are showing
- [ ] Configure site generation cron job (hourly)
- [ ] Ensure AEMET_API_KEY is configured (required for builds)
- [ ] Check `/home/private/data/request-audit.json` for errors (via SSH)
- [ ] View `build-report.html` to verify weather data fetching

**After first deployment (analytics):**
- [ ] Run initial analytics processing: `/home/private/bin/log-analyzer-daily.sh`
- [ ] Verify analytics output files in `/home/private/log-analyzer-data/`
- [ ] Verify public analytics report endpoint (`/analytics_report.html`)
- [ ] Configure analytics cron job: `15 1 * * *` (daily at 01:15)
- [ ] Configure Bunny backup env vars (`BUNNY_STORAGE_*`) and redeploy
- [ ] Configure NFSN log rotation: weekly with compression (NFSN web UI → Site Information)
- [ ] Verify Bunny mirror was updated (check `/home/logs/log-analyzer.log`)
- [ ] Verify stale-data workflow is green (`Check Analytics Freshness`)

**After each deployment:**
- [ ] Verify site updates with new content
- [ ] Check `/home/logs/generate.log` for build errors
- [ ] Check `/home/logs/log-analyzer.log` for analytics errors
