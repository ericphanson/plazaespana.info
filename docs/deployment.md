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

build/buildsite                      → /home/private/bin/buildsite
ops/cron-generate.sh                 → /home/private/bin/cron-generate.sh
config.toml                          → /home/private/config.toml
templates/index-grouped.tmpl.html    → /home/private/templates/index-grouped.tmpl.html
public/assets/site.*.css             → /home/public/assets/
public/assets/build-report.*.css     → /home/public/assets/
public/assets/*.hash                 → /home/public/assets/
ops/htaccess                         → /home/public/.htaccess
```

After upload, binary runs to generate:
- `/home/public/index.html` - Event listing (web-accessible)
- `/home/public/events.json` - JSON API (web-accessible)
- `/home/private/data/` - Cache & audit logs (not web-accessible)

## NFSN Directory Structure

```
/home/
  private/              # ❌ Not web-accessible
    bin/buildsite       # Binary
    config.toml         # Config
    templates/          # HTML templates
    data/               # Cache, audit logs (auto-created)

  public/               # ✅ Web root (served via HTTP)
    index.html          # Generated event listing
    events.json         # Generated JSON API
    assets/             # CSS
    .htaccess           # Apache config
```

Only `public/` contents are accessible via HTTP. All internal files (binary, config, templates, cache) stay in `private/`.

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

## Security

- Never commit private keys to the repository
- Private keys belong in `~/.ssh/` (local) and GitHub Secrets (CI)
- Public keys are safe to share (uploaded to NFSN)
- Use `ed25519` keys (more secure than RSA)

## Deployment Checklist

**Before deploying:**
- [ ] Tests pass (`just test`)
- [ ] Binary builds (`just freebsd`)
- [ ] SSH key added to NFSN
- [ ] Credentials configured (direnv or secrets)

**After deploying:**
- [ ] Visit NFSN site URL to verify
- [ ] Check events are showing
- [ ] Configure cron job (if first deployment)
- [ ] Check `/home/private/data/request-audit.json` for errors (via SSH)
