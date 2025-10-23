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
| `AEMET_API_KEY`    | AEMET weather API key        | Your AEMET OpenData API key         | Optional |

⚠️ Use the **private key** that matches the public key uploaded to NFSN.

**Note:** `AEMET_API_KEY` is optional. If not provided, the site builds without weather data (graceful degradation).

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

The site integrates weather forecasts from AEMET (Spanish Meteorological Agency). This is optional - the site works fine without it, but weather data enhances event cards.

### Get an API Key

1. Register at: https://opendata.aemet.es/centrodedescargas/altaUsuario
2. Wait for email with API key (usually instant)
3. API keys have indefinite validity (no expiration)

### Configure for Production (NFSN)

Add the API key as an environment variable in your scheduled task:

1. NFSN web interface → Sites → your_site → Scheduled Tasks
2. Edit your hourly site generation task
3. Update the command to include the API key:

```bash
export AEMET_API_KEY=your_aemet_api_key_here && /home/private/bin/cron-generate.sh
```

The cron wrapper script will pass this environment variable to the buildsite binary.

**To update the key:**
- Just edit the scheduled task and replace the value
- No need to redeploy or SSH to the server

### Configure for GitHub Actions

For automated deployments and PR previews, add the API key to GitHub repository secrets:

1. Repository Settings → Secrets and variables → Actions → New repository secret
2. Name: `AEMET_API_KEY`
3. Value: (paste your AEMET API key)

The GitHub Actions workflows are already configured to use this secret.

### Graceful Degradation

**If API key is missing or invalid:**
- Site generates successfully without weather data
- Build report shows weather fetch errors
- Events render normally (weather is an optional enhancement)
- No impact on existing cultural/city events

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
templates/index-grouped.tmpl.html    → /home/private/templates/index-grouped.tmpl.html

# Static assets
public/assets/site.*.css             → /home/public/assets/
public/assets/build-report.*.css     → /home/public/assets/
public/assets/*.hash                 → /home/public/assets/
public/assets/weather-icons/*.png    → /home/public/assets/weather-icons/
ops/htaccess                         → /home/public/.htaccess

# AWStats
ops/awstats.conf                     → /home/private/awstats.conf
ops/awstats-weekly.sh                → /home/private/bin/awstats-weekly.sh
ops/stats.htaccess                   → /home/public/stats/.htaccess
```

After upload, binary runs to generate:
- `/home/public/index.html` - Event listing (web-accessible)
- `/home/public/events.json` - JSON API (web-accessible)
- `/home/private/data/` - Cache & audit logs (not web-accessible)

AWStats generates (via weekly cron):
- `/home/public/stats/` - AWStats HTML pages (Basic Auth protected)
- `/home/private/awstats-data/` - AWStats database files (synced to git via GitHub Actions)

## NFSN Directory Structure

```
/home/
  private/              # ❌ Not web-accessible
    bin/
      buildsite         # Site generator binary
      cron-generate.sh  # Site generation wrapper (hourly cron)
      awstats-weekly.sh # AWStats processor (weekly cron)
    config.toml         # Site generator config
    awstats.conf        # AWStats config
    templates/          # HTML templates
    data/               # Site generator cache, audit logs (auto-created)
    awstats-data/       # AWStats database files (synced to git)

  protected/            # 🔒 Apache-readable only (not web-accessible)
    .htpasswd           # Basic Auth passwords

  public/               # ✅ Web root (served via HTTP)
    index.html          # Generated event listing
    events.json         # Generated JSON API
    assets/             # CSS files and weather icons
      site.*.css        # Hashed main site CSS
      build-report.*.css # Hashed build report CSS
      weather-icons/    # AEMET weather icons (PNG)
    stats/              # AWStats HTML (Basic Auth protected)
      .htaccess         # Basic Auth config for stats
      index.html        # AWStats main page
    .htaccess           # Apache config (caching, security headers)

  logs/                 # Log files
    access_log          # Apache access log (NFSN rotates automatically)
    generate.log        # Site generation log
    awstats.log         # AWStats processing log
```

**Access control:**
- Only `/home/public/` is web-accessible via HTTP/HTTPS
- `/home/public/stats/` requires Basic Auth (username/password)
- `/home/protected/` is readable by Apache (for .htpasswd) but not web-accessible
- All other files (`/home/private/`, `/home/logs/`) are SSH-only

## AWStats Setup (One-Time Configuration)

After first deployment with AWStats files, complete these one-time setup steps on NFSN.

### 1. Create Basic Auth Password

Protect the `/stats/` directory with a password:

```bash
# SSH to NFSN
ssh $NFSN_USER@$NFSN_HOST

# Create protected directory if it doesn't exist
mkdir -p /home/protected

# Create htpasswd file (username: awstats)
htpasswd -c /home/protected/.htpasswd awstats
# Enter password when prompted

# Set secure permissions
chmod 600 /home/protected/.htpasswd
chmod 755 /home/protected

# Exit SSH
exit
```

**Important:** Remember this username/password - you'll need it to access `https://plazaespana.info/stats/`

### 2. Verify AWStats Config

Test that AWStats can read the config:

```bash
ssh $NFSN_USER@$NFSN_HOST

# Test config syntax
perl /usr/local/www/awstats/cgi-bin/awstats.pl -configdir=/home/private -config=awstats -configtest

# Expected output should show:
# Config file '/home/private/awstats.conf' read successfully
# LogFile = /home/logs/access_log
# SiteDomain = plazaespana.info
```

### 3. Run Initial AWStats Processing

Generate the first stats manually:

```bash
# Still on NFSN via SSH

# IMPORTANT: If you previously ran AWStats without privacy settings,
# clean existing database files to remove stored IP addresses
rm -f /home/private/awstats-data/*.txt

# Run initial processing with privacy-focused config
/home/private/bin/awstats-weekly.sh

# Check for errors
tail -50 /home/logs/awstats.log

# Verify files were created
ls -lh /home/public/stats/
ls -lh /home/private/awstats-data/
```

**Expected output:**
- `/home/public/stats/index.html` and other AWStats HTML files
- `/home/private/awstats-data/*.txt` - AWStats database files (aggregate statistics only, no IPs)

### 4. Test Web Access

Visit `https://plazaespana.info/stats/` in your browser:
- Should prompt for username/password (Basic Auth)
- After login, should show AWStats statistics page
- If you see Apache directory listing or 403 error, check `.htaccess` deployment

### 5. Setup AWStats Cron Job

Add weekly AWStats processing to NFSN scheduled tasks:

1. NFSN web interface → Sites → your_site → Scheduled Tasks
2. Add task:
   - **Command:** `/home/private/bin/awstats-weekly.sh`
   - **Schedule:** `0 1 * * 0` (Sunday at 1 AM)
   - **Tag:** `awstats-weekly` (optional, for identification)

The wrapper script:
- Logs all output to `/home/logs/awstats.log` with timestamps
- Only sends email on processing failures (non-zero exit code)
- Includes full log in error emails for complete debugging context

**Why Sunday 1 AM?**
- Low traffic time
- After weekend events (captures full week)
- Weekly processing schedule

**View logs:**
```bash
ssh your_username@ssh.phx.nearlyfreespeech.net
tail -f /home/logs/awstats.log
```

### 6. Setup AWStats Database Sync (GitHub Actions)

For automated PR creation when statistics database is updated:

1. **Generate dedicated SSH key for database sync:**
   ```bash
   ssh-keygen -t ed25519 -f ~/.ssh/nfsn_awstats -N ""
   ```

2. **Add public key to NFSN:**
   - NFSN web interface → Sites → your_site → SSH/SFTP → Authorized Keys
   - Upload `~/.ssh/nfsn_awstats.pub`

3. **Add GitHub Secrets:**
   - Repository Settings → Secrets and variables → Actions
   - Add secrets:
     - `NFSN_SSH_KEY`: Paste contents of `~/.ssh/nfsn_awstats` (private key)
     - `NFSN_HOST`: Your NFSN hostname (e.g., `ssh.phx.nearlyfreespeech.net`)
     - `NFSN_USER`: Your NFSN username (format: `username_sitename`)

4. **Test workflow:**
   - GitHub → Actions → "Fetch AWStats Archives" → Run workflow
   - Should create/update PR with AWStats database files from `/home/private/awstats-data/`

**What gets synced:**
- Monthly statistics files (`awstatsMMYYYY.awstats.txt`)
- DNS cache and other state files
- **Privacy:** Only aggregate statistics (no IPs or individual requests)

**Note:** The workflow runs automatically after each push to `main`, or manually via workflow_dispatch.

### 7. Configure NFSN Log Rotation

Since AWStats tracks its position in the log file, NFSN's automatic log rotation won't cause issues. However, you should verify rotation is configured:

1. **NFSN web interface → Sites → your_site → Site Information**
2. Look for "Log Rotation" settings
3. **Recommended:** Daily or weekly rotation with compression

**Why this matters:**
- Without rotation, `access_log` grows indefinitely
- AWStats uses `KeepBackupOfHistoricFiles=1` to track position across rotations
- After rotation, AWStats automatically detects the new log file and continues processing
- Historical data is preserved in AWStats database files (synced to git)

**Example rotation schedule:**
- Rotate: Weekly (recommended)
- Keep: 4 weeks of compressed logs
- Compression: gzip

## Troubleshooting

### Permission denied (SSH)

Add your SSH public key: NFSN web interface → Profile → SSH/SFTP Keys

### Host key verification failed

```bash
ssh-keygen -R ssh.phx.nearlyfreespeech.net
ssh-keyscan -H ssh.phx.nearlyfreespeech.net >> ~/.ssh/known_hosts
```

### GitHub Actions deployment fails

1. Verify all three secrets are set in repository settings
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
/home/private/bin/buildsite -config /home/private/config.toml -out-dir /home/public -data-dir /home/private/data -template-path /home/private/templates/index-grouped.tmpl.html -fetch-mode production
```

### AWStats shows "No data available"

**Causes:**
1. Access log is empty (no traffic yet)
2. AWStats hasn't processed any logs
3. Config pointing to wrong log file

**Fix:**
```bash
# Check if logs exist
ls -lh /home/logs/access_log

# Check if AWStats data exists
ls -lh /home/private/awstats-data/

# Manually process current log
/home/private/bin/awstats-weekly.sh
```

### Basic Auth not working for /stats/

**Symptom:** Can access `/stats/` without password, or get 500 error

**Fix:**
```bash
# Verify .htaccess was deployed
ssh $NFSN_USER@$NFSN_HOST ls -la /home/public/stats/.htaccess

# Verify .htpasswd exists
ssh $NFSN_USER@$NFSN_HOST ls -la /home/protected/.htpasswd

# Check permissions
ssh $NFSN_USER@$NFSN_HOST "stat -f '%A %N' /home/protected/.htpasswd /home/protected"
# Should show: 600 /home/protected/.htpasswd and 755 /home/protected
```

### Database sync workflow fails

**Common causes:**
1. SSH key not added to NFSN
2. `NFSN_SSH_KEY` secret has wrong key
3. `gh` authentication failed

**Fix:**
```bash
# Test SSH access locally
ssh -i ~/.ssh/nfsn_awstats $NFSN_USER@$NFSN_HOST ls /home/private/awstats-data/

# Test SCP access
scp "$NFSN_USER@$NFSN_HOST:/home/private/awstats-data/*.txt" /tmp/test-awstats/

# Verify GitHub CLI auth
gh auth status

# Check workflow logs for specific error
```

## Security

- Never commit private keys to the repository
- Private keys belong in `~/.ssh/` (local) and GitHub Secrets (CI)
- Public keys are safe to share (uploaded to NFSN)
- Use `ed25519` keys (more secure than RSA)
- Keep separate SSH keys for deployment vs. database sync (better security isolation)

## Deployment Checklist

**Before first deployment:**
- [ ] Tests pass (`just test`)
- [ ] Binary builds (`just freebsd`)
- [ ] SSH key added to NFSN
- [ ] Credentials configured (direnv or secrets)
- [ ] (Optional) AEMET API key obtained and configured

**After first deployment:**
- [ ] Visit NFSN site URL to verify site works
- [ ] Check events are showing
- [ ] Configure site generation cron job (hourly)
- [ ] (Optional) Add AEMET_API_KEY to cron command if using weather
- [ ] Check `/home/private/data/request-audit.json` for errors (via SSH)
- [ ] (Optional) View `build-report.html` to verify weather data fetching

**AWStats setup (after first deployment):**
- [ ] Create protected directory: `mkdir -p /home/protected`
- [ ] Create Basic Auth password: `htpasswd -c /home/protected/.htpasswd awstats`
- [ ] Set permissions: `chmod 600 /home/protected/.htpasswd && chmod 755 /home/protected`
- [ ] Verify AWStats config: `perl /usr/local/www/awstats/cgi-bin/awstats.pl -configdir=/home/private -config=awstats -configtest`
- [ ] Run initial processing: `/home/private/bin/awstats-weekly.sh`
- [ ] Test web access: Visit `https://plazaespana.info/stats/` (should prompt for password)
- [ ] Configure NFSN log rotation: Weekly with compression (NFSN web UI → Site Information)
- [ ] Configure AWStats cron job: `0 1 * * 0` (Sunday 1 AM)
- [ ] Setup database sync SSH key and GitHub secrets
- [ ] Test database sync workflow: GitHub Actions → "Fetch AWStats Archives" → Run workflow

**After each deployment:**
- [ ] Verify site updates with new content
- [ ] Check `/home/logs/generate.log` for build errors
- [ ] If AWStats enabled, check `/home/logs/awstats.log` for stats errors
