# Deployment Guide

This guide covers deploying the Madrid Events site to NearlyFreeSpeech.NET (NFSN).

## Prerequisites

- SSH access to your NFSN account
- SSH key pair for authentication

## Quick Start

### Local Deployment

```bash
# Set environment variables
export NFSN_HOST=ssh.phx.nearlyfreespeech.net
export NFSN_USER=your_username

# Deploy
just deploy
```

This will:
1. Build the FreeBSD binary
2. Hash the CSS for cache busting
3. Upload all files to NFSN
4. Set correct permissions
5. Regenerate the site on the server

### GitHub Actions Deployment

Deployment happens automatically on push to `main` after tests pass.

## SSH Key Setup

### For Local Deployment

1. Generate SSH key if you don't have one:
   ```bash
   ssh-keygen -t ed25519 -C "your_email@example.com"
   ```

2. Add public key to NFSN:
   - Log into NFSN web interface
   - Go to Profile → SSH/SFTP Keys
   - Add your public key (`~/.ssh/id_ed25519.pub`)

3. Test connection:
   ```bash
   ssh your_username@ssh.phx.nearlyfreespeech.net
   ```

### For GitHub Actions

1. **Add SSH private key to GitHub Secrets:**
   - Go to repository Settings → Secrets and variables → Actions
   - Click "New repository secret"
   - Name: `NFSN_SSH_KEY`
   - Value: Contents of your **private** SSH key (`~/.ssh/id_ed25519`)

   **⚠️ IMPORTANT:** This should be the private key that corresponds to the public key uploaded to NFSN.

2. **Add NFSN host:**
   - Name: `NFSN_HOST`
   - Value: `ssh.phx.nearlyfreespeech.net` (or your specific NFSN SSH host)

3. **Add NFSN username:**
   - Name: `NFSN_USER`
   - Value: Your NFSN username

## Required GitHub Secrets

| Secret Name    | Description                          | Example Value                        |
|----------------|--------------------------------------|--------------------------------------|
| `NFSN_SSH_KEY` | Private SSH key for authentication   | Contents of `~/.ssh/id_ed25519`      |
| `NFSN_HOST`    | NFSN SSH hostname                    | `ssh.phx.nearlyfreespeech.net`       |
| `NFSN_USER`    | NFSN SSH username                    | `your_username`                      |

## What Gets Deployed

The deployment process uploads:

1. **Binary:** `build/buildsite` → `/home/bin/buildsite`
2. **Config:** `config.toml` → `/home/config.toml`
3. **Template:** `templates/index-grouped.tmpl.html` → `/home/templates/index-grouped.tmpl.html`
4. **CSS:** `public/assets/site.*.css` → `/home/public/assets/`
5. **Apache config:** `ops/htaccess` → `/home/public/.htaccess`

After uploading, the binary is run to regenerate the site with fresh data.

## Deployment Flow

### GitHub Actions (Automatic)

```
Push to main
  ↓
Run tests
  ↓
Build FreeBSD binary
  ↓ (if tests pass)
Deploy to NFSN
  ↓
Regenerate site
```

### Local (Manual)

```
just deploy
  ↓
Build FreeBSD binary
  ↓
Hash CSS
  ↓
Upload via SCP
  ↓
SSH: Set permissions
  ↓
SSH: Regenerate site
```

## Cron Setup on NFSN

After deploying, set up hourly site regeneration:

1. Log into NFSN web interface
2. Go to Sites → your_site → Scheduled Tasks
3. Add new task:
   - **Command:** `/home/bin/buildsite -config /home/config.toml -fetch-mode production`
   - **Schedule:** `0 * * * *` (every hour at :00)
   - Or use the web UI to select "Every hour"

**Important:** Always use `-fetch-mode production` for cron jobs (30min cache TTL, 2s delays between requests).

## Directory Structure on NFSN

After deployment:

```
/home/
  bin/
    buildsite              # Executable binary
  config.toml              # Configuration file
  templates/
    index-grouped.tmpl.html # HTML template
  public/                  # Web root (served to visitors)
    index.html             # Generated event listing
    events.json            # Generated JSON API
    assets/
      site.<hash>.css      # Hashed CSS file
    .htaccess              # Apache configuration
  data/                    # Auto-created by binary
    http-cache/            # Cached HTTP responses
    request-audit.json     # HTTP request log
    last_success.json      # Snapshot fallback
    audit-events.json      # Event audit trail
```

## Troubleshooting

### "Permission denied" errors

Make sure your SSH key is added to NFSN:
- NFSN web interface → Profile → SSH/SFTP Keys

### "Host key verification failed"

Remove old host key and re-add:
```bash
ssh-keygen -R ssh.phx.nearlyfreespeech.net
ssh-keyscan -H ssh.phx.nearlyfreespeech.net >> ~/.ssh/known_hosts
```

### GitHub Actions deployment fails

1. Check that all secrets are set correctly in repository settings
2. Ensure the private key in `NFSN_SSH_KEY` matches the public key on NFSN
3. Check GitHub Actions logs for specific error messages

### Site not regenerating

SSH into NFSN and check logs:
```bash
ssh your_username@ssh.phx.nearlyfreespeech.net
cd /home
./bin/buildsite -config config.toml -fetch-mode production
```

Look for error messages in the output.

## Security Notes

- **Never commit private keys** to the repository
- Private keys should only be in:
  - Your local `~/.ssh/` directory
  - GitHub Secrets (for CI/CD)
- The public key is safe to share and goes on NFSN
- Use `ed25519` keys (more secure than older RSA)

## Local Environment Variables

For convenience, add to your `~/.bashrc` or `~/.zshrc`:

```bash
export NFSN_HOST=ssh.phx.nearlyfreespeech.net
export NFSN_USER=your_username
```

Then you can just run `just deploy` without setting variables each time.

## Deployment Checklist

Before deploying:

- [ ] Tests pass locally (`just test`)
- [ ] FreeBSD binary builds (`just freebsd`)
- [ ] Config is valid (`just config`)
- [ ] SSH key is set up on NFSN
- [ ] Environment variables are set (local) or secrets configured (GitHub)

After deploying:

- [ ] Visit your NFSN site URL to verify
- [ ] Check that events are showing
- [ ] Verify cron job is configured
- [ ] Check `/home/data/request-audit.json` for any fetch errors
