# Deployment Guide

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

| Secret Name    | Description           | Value                               |
|----------------|-----------------------|-------------------------------------|
| `NFSN_SSH_KEY` | Private SSH key       | Contents of `~/.ssh/id_ed25519`     |
| `NFSN_HOST`    | NFSN SSH hostname     | `ssh.phx.nearlyfreespeech.net`      |
| `NFSN_USER`    | NFSN username         | `your_username`                     |

⚠️ Use the **private key** that matches the public key uploaded to NFSN.

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
- `/home/private/rollups/` - Weekly compressed access logs
- `/home/private/awstats-data/` - AWStats database files
- `/home/logs/access_log.backup` - Rolling backup of access log

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
    .htpasswd           # Basic Auth passwords (for /stats/)
    templates/          # HTML templates
    data/               # Site generator cache, audit logs (auto-created)
    awstats-data/       # AWStats database files (auto-created)
    rollups/            # Weekly access log archives (auto-created)

  public/               # ✅ Web root (served via HTTP)
    index.html          # Generated event listing
    events.json         # Generated JSON API
    assets/             # CSS files
    stats/              # AWStats HTML (Basic Auth protected)
      .htaccess         # Basic Auth config for stats
      index.html        # AWStats main page
    .htaccess           # Apache config (caching, security headers)

  logs/                 # Log files
    access_log          # Apache access log (truncated weekly)
    access_log.backup   # Previous week's log (rolling backup)
    generate.log        # Site generation log
    awstats.log         # AWStats processing log
```

**Access control:**
- Only `/home/public/` is web-accessible via HTTP/HTTPS
- `/home/public/stats/` requires Basic Auth (username/password)
- All other files (`/home/private/`, `/home/logs/`) are SSH-only

## AWStats Setup (One-Time Configuration)

After first deployment with AWStats files, complete these one-time setup steps on NFSN.

### 1. Create Basic Auth Password

Protect the `/stats/` directory with a password:

```bash
# SSH to NFSN
ssh $NFSN_USER@$NFSN_HOST

# Create htpasswd file (username: awstats)
htpasswd -c /home/private/.htpasswd awstats
# Enter password when prompted

# Set secure permissions
chmod 600 /home/private/.htpasswd
chmod 711 /home/private

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
/home/private/bin/awstats-weekly.sh

# Check for errors
tail -50 /home/logs/awstats.log

# Verify files were created
ls -lh /home/public/stats/
ls -lh /home/private/rollups/
```

**Expected output:**
- `/home/public/stats/index.html` and other AWStats HTML files
- `/home/private/rollups/YYYY-Www.txt.gz` (current week's rollup)
- `/home/logs/access_log.backup` (backup before truncation)

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

**Why Sunday 1 AM?**
- Low traffic time (minimal data loss during log truncation)
- Weekly rollups align with calendar weeks
- After weekend events (captures full week)

### 6. Setup AWStats Rollup Fetch (GitHub Actions)

For automated PR creation when new rollups are available:

1. **Generate dedicated SSH key for rollup fetch:**
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
   - GitHub → Actions → "Fetch AWStats Rollups" → Run workflow
   - Should create/update PR with new rollups from `/home/private/rollups/`

**Note:** The workflow runs automatically after each push to `main`, or manually via workflow_dispatch.

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
ssh $NFSN_USER@$NFSN_HOST ls -la /home/private/.htpasswd

# Check permissions
ssh $NFSN_USER@$NFSN_HOST "stat -f '%A %N' /home/private/.htpasswd /home/private"
# Should show: 600 /home/private/.htpasswd and 711 /home/private
```

### Rollup fetch workflow fails

**Common causes:**
1. SSH key not added to NFSN
2. `NFSN_SSH_KEY` secret has wrong key
3. `gh` authentication failed

**Fix:**
```bash
# Test SSH access locally
ssh -i ~/.ssh/nfsn_awstats $NFSN_USER@$NFSN_HOST ls /home/private/rollups/

# Verify GitHub CLI auth
gh auth status

# Check workflow logs for specific error
```

## Security

- Never commit private keys to the repository
- Private keys belong in `~/.ssh/` (local) and GitHub Secrets (CI)
- Public keys are safe to share (uploaded to NFSN)
- Use `ed25519` keys (more secure than RSA)
- Keep separate SSH keys for deployment vs. rollup fetch (better security isolation)

## Deployment Checklist

**Before first deployment:**
- [ ] Tests pass (`just test`)
- [ ] Binary builds (`just freebsd`)
- [ ] SSH key added to NFSN
- [ ] Credentials configured (direnv or secrets)

**After first deployment:**
- [ ] Visit NFSN site URL to verify site works
- [ ] Check events are showing
- [ ] Configure site generation cron job (hourly)
- [ ] Check `/home/private/data/request-audit.json` for errors (via SSH)

**AWStats setup (after first deployment):**
- [ ] Create Basic Auth password: `htpasswd -c /home/private/.htpasswd username`
- [ ] Set permissions: `chmod 600 /home/private/.htpasswd && chmod 711 /home/private`
- [ ] Verify AWStats config: `perl /usr/local/www/awstats/cgi-bin/awstats.pl -configdir=/home/private -config=awstats -configtest`
- [ ] Run initial processing: `/home/private/bin/awstats-weekly.sh`
- [ ] Test web access: Visit `https://plazaespana.info/stats/` (should prompt for password)
- [ ] Configure AWStats cron job: `0 1 * * 0` (Sunday 1 AM)
- [ ] Setup rollup fetch SSH key and GitHub secrets
- [ ] Test rollup fetch workflow via workflow_dispatch

**After each deployment:**
- [ ] Verify site updates with new content
- [ ] Check `/home/logs/generate.log` for build errors
- [ ] If AWStats enabled, check `/home/logs/awstats.log` for stats errors
