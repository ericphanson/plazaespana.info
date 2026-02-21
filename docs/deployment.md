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

**Automatic:** GitHub Actions deploys on push to `main` (after tests pass).

## GitHub Actions Setup

Add these secrets in repository Settings → Secrets and variables → Actions:

| Secret Name        | Description                  | Value                               | Required |
|--------------------|------------------------------|-------------------------------------|----------|
| `NFSN_SSH_KEY`     | Private SSH key              | Contents of `~/.ssh/id_ed25519`     | Yes      |
| `NFSN_HOST`        | NFSN SSH hostname            | SSH hostname from site information  | Yes      |
| `NFSN_USER`        | NFSN username                | `your_username`                     | Yes      |
| `NFSN_KNOWN_HOST`  | SSH host key (for security)  | Output from `ssh-keyscan` command   | Yes      |
| `AEMET_API_KEY`    | AEMET weather API key        | Your AEMET OpenData API key         | Yes      |

⚠️ Use the **private key** that matches the public key uploaded to NFSN.

**Note:** `AEMET_API_KEY` is required. Without it, site builds will fail.

### How to populate NFSN_KNOWN_HOST

To securely verify the NFSN host key, run this command **from a trusted machine** (not in CI):

```bash
ssh-keyscan -H "$NFSN_HOST"
```

This will output something like:
```
|1|abc123...= ssh-ed25519 AAAA...
|1|def456...= ssh-rsa AAAA...
```

**Copy the entire output** and paste it as the value for the `NFSN_KNOWN_HOST` secret.

**Why this matters:** This prevents man-in-the-middle attacks during GitHub Actions deployments. By capturing the host key once from a trusted network and storing it as a secret, all future CI runs will verify they're connecting to the legitimate NFSN server.

## Cron Setup on NFSN

After first deployment, set up hourly regeneration:

1. NFSN web interface → Sites → your_site → Scheduled Tasks
2. Add task:
   - **Command:** `/home/private/bin/cron-generate.sh`
   - **Schedule:** Every hour (or `0 * * * *`)

The wrapper script:
- Logs all output to `/home/logs/generate.log` with timestamps
- Only sends email on build failures (non-zero exit code)
- Includes full log in error emails for complete debugging context

**View logs:**
```bash
ssh your_username@ssh.phx.nearlyfreespeech.net
tail -f /home/logs/generate.log
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
ops/log-analyzer-weekly.sh              → /home/private/bin/log-analyzer-weekly.sh
```

After upload, binary runs to generate:
- `/home/public/index.html` - Event listing (web-accessible)
- `/home/public/events.json` - JSON API (web-accessible)
- `/home/private/data/` - Cache & audit logs (not web-accessible)

Log analyzer generates (via weekly cron):
- `/home/private/log-analyzer-data/` - aggregate monthly JSON + lifetime JSON + report HTML
- Synced to git via GitHub Actions into `log-analyzer-data/`

## NFSN Directory Structure

```
/home/
  private/              # ❌ Not web-accessible
    bin/
      buildsite         # Site generator binary
      cron-generate.sh  # Site generation wrapper (hourly cron)
      log-analyzer      # Log analyzer binary (FreeBSD)
      log-analyzer-weekly.sh # Aggregate analytics snapshot job (weekly cron)
    config.toml         # Site generator config
    aemet-api-key.txt   # AEMET API key (optional, mode 600)
    templates/          # HTML templates
    data/               # Site generator cache, audit logs (auto-created)
    log-analyzer-data/  # Aggregate analytics snapshots (synced to git)

  public/               # ✅ Web root (served via HTTP)
    index.html          # Generated event listing
    events.json         # Generated JSON API
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

## Log Analyzer Setup (One-Time Configuration)

After first deployment with `log-analyzer`, complete these steps on NFSN.

### 1. Run Initial Aggregation

```bash
ssh $NFSN_USER@$NFSN_HOST

# Run initial analytics snapshot
/home/private/bin/log-analyzer-weekly.sh

# Check output and logs
ls -lh /home/private/log-analyzer-data/
tail -50 /home/logs/log-analyzer.log
```

Expected output:
- `/home/private/log-analyzer-data/lifetime.json`
- `/home/private/log-analyzer-data/YYYY-MM.json` files
- `/home/private/log-analyzer-data/report.html`

### 2. Setup NFSN Cron Job

Add weekly analytics processing:

1. NFSN web interface → Sites → your_site → Scheduled Tasks
2. Add task:
   - **Command:** `/home/private/bin/log-analyzer-weekly.sh`
   - **Schedule:** `0 1 * * 0` (Sunday at 1 AM)
   - **Tag:** `log-analyzer-weekly` (optional)

The wrapper script:
- Reads logs from `/home/logs/access_log*`
- Writes aggregate output to `/home/private/log-analyzer-data`
- Logs all activity to `/home/logs/log-analyzer.log`
- Emits stderr on failures (so cron can alert)

### 3. Setup Stats Sync (GitHub Actions)

For automated PR creation with latest aggregate analytics:

1. Generate dedicated SSH key for sync:
   ```bash
   ssh-keygen -t ed25519 -f ~/.ssh/nfsn_stats -N ""
   ```
2. Add public key to NFSN authorized keys.
3. Add/update GitHub secrets:
   - `NFSN_SSH_KEY`: private key contents
   - `NFSN_HOST`
   - `NFSN_USER`
   - `NFSN_KNOWN_HOST`
4. Run workflow: GitHub → Actions → "Fetch Log Analyzer Stats".

The workflow pulls `/home/private/log-analyzer-data`, enforces privacy checks, and opens/updates a PR against canonical branch `log-analyzer-data`.

### 4. Configure NFSN Log Rotation

`log-analyzer` is stateless and reprocesses all available `access_log*` files each run.

Recommended:
- Rotate weekly
- Keep at least 4–8 weeks
- Enable compression

This controls how much exact history is available before only HLL-based long-term unique estimates remain.

## Troubleshooting

### Permission denied (SSH)

Add your SSH public key: NFSN web interface → Profile → SSH/SFTP Keys

### Host key verification failed

```bash
ssh-keygen -R ssh.phx.nearlyfreespeech.net
ssh-keyscan -H ssh.phx.nearlyfreespeech.net >> ~/.ssh/known_hosts
```

### GitHub Actions deployment fails

1. Verify required secrets are set in repository settings (`NFSN_SSH_KEY`, `NFSN_HOST`, `NFSN_USER`, `NFSN_KNOWN_HOST`, `AEMET_API_KEY`)
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
/home/private/bin/log-analyzer-weekly.sh

# Inspect logs
tail -100 /home/logs/log-analyzer.log
```

### Database sync workflow fails

**Common causes:**
1. SSH key not added to NFSN
2. `NFSN_SSH_KEY` secret has wrong key
3. `gh` authentication failed

**Fix:**
```bash
# Test SSH access locally
ssh -i ~/.ssh/nfsn_stats $NFSN_USER@$NFSN_HOST ls /home/private/log-analyzer-data/

# Test SCP access
scp "$NFSN_USER@$NFSN_HOST:/home/private/log-analyzer-data/*.json" /tmp/test-log-analyzer/

# Verify GitHub CLI auth
gh auth status

# Check workflow logs for specific error
```

## Security

- Never commit private keys to the repository
- Private keys belong in `~/.ssh/` (local) and GitHub Secrets (CI)
- Public keys are safe to share (uploaded to NFSN)
- Use `ed25519` keys (more secure than RSA)
- Keep separate SSH keys for deployment vs. analytics sync (better security isolation)

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

**Log-analyzer setup (after first deployment):**
- [ ] Run initial processing: `/home/private/bin/log-analyzer-weekly.sh`
- [ ] Verify output files in `/home/private/log-analyzer-data/`
- [ ] Configure NFSN log rotation: Weekly with compression (NFSN web UI → Site Information)
- [ ] Configure log-analyzer cron job: `0 1 * * 0` (Sunday 1 AM)
- [ ] Setup analytics sync SSH key and GitHub secrets
- [ ] Test sync workflow: GitHub Actions → "Fetch Log Analyzer Stats" → Run workflow

**After each deployment:**
- [ ] Verify site updates with new content
- [ ] Check `/home/logs/generate.log` for build errors
- [ ] Check `/home/logs/log-analyzer.log` for analytics errors
