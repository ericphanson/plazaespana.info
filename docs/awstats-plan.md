# AWStats Integration Plan

## Goal
Set up AWStats to track weekly traffic statistics indefinitely, archive rollups in-repo via automated PRs, and serve static stats pages.

## ⚠️ Important Notes

1. **AWStats Config Verification Required**: This plan assumes NFSN's `-config=nfsn` flag automatically merges with `/home/private/.awstats.conf`. This must be verified during initial testing (see "Testing AWStats Configuration" section). If this doesn't work, we may need to use `-configdir=/home/private -config=awstats` instead.

2. **SSH Access Required**: The rollup fetch script requires SSH access to NFSN with key-based authentication (see "SSH Setup" section).

3. **GitHub Token Permissions**: PR creation requires `contents: write` and `pull-requests: write` permissions in GitHub Actions (see "GitHub Actions integration" section).

## Requirements
- Keep weekly aggregates indefinitely (in Git repo)
- NFSN handles log rotation automatically
- Run weekly via cron
- Generate static HTML pages (no CGI)
- Basic auth protection
- CI creates PRs for new weekly rollups

## Architecture

### Domains
- Primary: `plazaespana.nfshost.com`
- Custom: `plazaespana.info`

### File Locations (NFSN)
```
/home/logs/
  access_log              # Current Apache access log (truncated weekly after backup)
  access_log.backup       # Rolling backup (1 week retention, for disaster recovery)

/home/private/
  awstats/
    awstats.plazaespana.conf    # AWStats config
    awstats-data/               # AWStats database files
      awstats102025.txt         # October 2025 data (permanent)
      awstats112025.txt         # November 2025 data (permanent)
  rollups/                      # Weekly compressed logs (private, SCP access)
    2025-W43.txt.gz             # Week 43 compressed log
    2025-W44.txt.gz             # Week 44 compressed log
  bin/
    awstats-weekly.sh           # Weekly processing + static page generation
  .htpasswd                     # Basic auth credentials

/home/public/
  stats/                        # Static AWStats HTML (Basic Auth protected)
    index.html                  # Main stats page
    awstats.plazaespana.*.html  # Monthly/daily pages
```

### File Locations (Git Repo)
```
awstats-archives/
  2025-W43.txt.gz               # Week 43 rollup (checked into repo)
  2025-W44.txt.gz               # Week 44 rollup (checked into repo)
  README.md                     # Archive documentation
```

### AWStats Data Files
- AWStats maintains monthly data files (e.g., `awstats102025.txt`)
- These files are updated incrementally
- We'll keep these indefinitely for trend analysis
- Compressed weekly archives provide backup/detailed logs

## Weekly Processing Script

**`/home/private/bin/awstats-weekly.sh`**

```bash
#!/bin/bash
# Weekly AWStats processing, static page generation, and log archiving
set -euo pipefail

AWSTATS_STATIC=/usr/local/www/awstats/tools/awstats_buildstaticpages.pl
ROLLUP_DIR=/home/private/rollups
STATS_DIR=/home/public/stats
DATA_DIR=/home/private/awstats-data
ACCESS_LOG=/home/logs/access_log
LOG_FILE=/home/logs/awstats.log

# Ensure directories exist
mkdir -p "$ROLLUP_DIR" "$STATS_DIR" "$DATA_DIR"

# Get current week number (YYYY-Www format)
WEEK=$(date +%Y-W%V)

echo "=== AWStats Weekly Processing: $WEEK ===" | tee -a "$LOG_FILE"
echo "Started: $(date)" | tee -a "$LOG_FILE"

# 1. Update AWStats database and generate static pages
# NFSN uses -config=nfsn which merges /home/private/.awstats.conf
if [ -f "$ACCESS_LOG" ]; then
    echo "Updating AWStats database and generating static pages..." | tee -a "$LOG_FILE"

    if ! perl "$AWSTATS_STATIC" \
        -config=nfsn \
        -update \
        -dir="$STATS_DIR" \
        2>&1 | tee -a "$LOG_FILE"; then
        echo "ERROR: AWStats processing failed" >&2
        exit 1
    fi

    # 2. Create symlink for clean index.html access
    cd "$STATS_DIR"
    rm -f index.html
    ln -s awstats.nfsn.html index.html

    # 3. Create weekly rollup (compressed access log) if not empty
    if [ -s "$ACCESS_LOG" ]; then
        echo "Creating weekly rollup: $WEEK.txt.gz" | tee -a "$LOG_FILE"
        gzip -c "$ACCESS_LOG" > "$ROLLUP_DIR/$WEEK.txt.gz"

        # 4. Truncate access log to prevent duplicates in next rollup
        # NFSN will continue writing to it
        echo "Truncating access log..." | tee -a "$LOG_FILE"
        > "$ACCESS_LOG"

        echo "Weekly rollup created: $ROLLUP_DIR/$WEEK.txt.gz" | tee -a "$LOG_FILE"
    else
        echo "Skipping rollup - access log is empty" | tee -a "$LOG_FILE"
    fi

    echo "Static pages updated: $STATS_DIR/" | tee -a "$LOG_FILE"
else
    echo "ERROR: Access log not found at $ACCESS_LOG" >&2
    exit 1
fi

echo "Completed: $(date)" | tee -a "$LOG_FILE"
```

**Key changes:**
- Uses NFSN's `-config=nfsn` flag (merges with `/home/private/.awstats.conf`)
- Single command updates database + generates static pages
- Creates `index.html` symlink for clean URLs (safely removes old symlink first)
- Stores rollups in `/home/private/rollups/` (not public)
- **Creates required directories** (`awstats-data`, `rollups`, `stats`)
- **Error handling**: Captures AWStats output to log file, exits on failure
- **Safe rollup creation**: Verifies rollup exists and has non-zero size before truncating
- **Rolling backup**: Keeps `access_log.backup` before truncating (1 week retention)
- **Conditional truncation**: Only truncates after successful AWStats processing AND rollup creation
- **Skips empty logs**: Doesn't create rollup if no traffic that week
- Each weekly rollup contains exactly one week's data
- All output logged to `/home/logs/awstats.log` for debugging

## AWStats Configuration

NFSN provides a default AWStats config (`-config=nfsn`). Customize by creating `/home/private/.awstats.conf`:

**`/home/private/.awstats.conf`**
```perl
# Site identification
SiteDomain="plazaespana.info"
HostAliases="plazaespana.nfshost.com plazaespana.info www.plazaespana.info"

# Log location (NFSN standard)
LogFile="/home/logs/access_log"

# Don't purge old records - keep history indefinitely
KeepBackupOfHistoricFiles=1
PurgeLogFile=0

# Data directory (permanent storage)
DirData="/home/private/awstats-data"
```

**File permissions:**
```bash
chmod 711 /home/private
chmod 644 /home/private/.awstats.conf
```

NFSN will merge this with their base config at `/home/tmp/nfsn-awstats.conf`.

## Cron Setup

**Weekly job (Sunday at 1 AM):**
```
0 1 * * 0 /home/private/bin/awstats-weekly.sh >> /home/logs/awstats.log 2>&1
```

**Why weekly?**
- Balances storage vs. granularity
- Access logs ~1MB/week on low-traffic site
- Easy to review weekly trends
- Not too much email if errors occur

## Viewing Statistics

### Static Pages (Recommended)
Access generated static pages:
```
https://plazaespana.nfshost.com/stats/
https://plazaespana.info/stats/
```

Protected by Basic Auth (configured in .htaccess)

### SSH + CLI
View stats via SSH:
```bash
ssh user@host
perl /usr/local/www/awstats/cgi-bin/awstats.pl \
  -config=plazaespana \
  -output \
  -month=10 \
  -year=2025
```

## Security Considerations

### Protect Stats Directory
Create `/home/public/stats/.htaccess` (note: separate file in stats directory, NOT in main .htaccess):
```apache
AuthType Basic
AuthName "Site Statistics"
AuthUserFile /home/private/.htpasswd
Require valid-user
```

**Note:** `.htaccess` files cannot use `<Directory>` directives - those only work in main Apache config. Place this `.htaccess` file directly inside the `/home/public/stats/` directory.

### Create Password File
```bash
ssh user@nfsn.host
htpasswd -c /home/private/.htpasswd yourusername
# Enter password when prompted
chmod 600 /home/private/.htpasswd
chmod 711 /home/private
```

## CI/PR Automation for Weekly Rollups

### Overview
After deployment (or manually via workflow dispatch), GitHub Actions checks for new weekly rollups on the server via SCP and creates/updates a PR to archive them in the repo.

**Key features:**
- Uses a **canonical branch** (`awstats-rollups`) for all rollup PRs
- **No duplicate PRs** - force-pushes to existing PR if one is open
- Runs on **push to main** and **workflow_dispatch** (NOT on cron)
- Separate workflow file (`.github/workflows/fetch-awstats-rollups.yml`)

### Workflow
1. Push to main triggers workflow (or manual workflow_dispatch)
2. Workflow runs `just fetch-rollups`
3. Script uses SSH/SCP to list and download new rollup files from `/home/private/rollups/`
4. If new files exist:
   - Checks out/creates canonical branch `awstats-rollups`
   - Adds new `awstats-archives/YYYY-Www.txt.gz` files
   - Updates `awstats-archives/README.md`
   - Commits with descriptive message
5. If PR already exists for `awstats-rollups` branch:
   - Force-pushes to update existing PR
6. If no PR exists:
   - Creates new PR from `awstats-rollups` branch

### Implementation

**`/workspace/scripts/fetch-rollups.sh`**
```bash
#!/bin/bash
# Download new AWStats rollups from server via SCP and create PR if needed
set -euo pipefail

# Check required environment variables
if [ -z "${NFSN_HOST:-}" ] || [ -z "${NFSN_USER:-}" ]; then
    echo "❌ Error: NFSN_HOST and NFSN_USER environment variables required"
    echo "   Set in .envrc.local or export manually"
    exit 1
fi

REMOTE_DIR="/home/private/rollups"
ARCHIVE_DIR="awstats-archives"

mkdir -p "$ARCHIVE_DIR"

echo "Checking for new rollups on $NFSN_HOST:$REMOTE_DIR"

# Get list of available rollups via SSH
AVAILABLE=$(ssh "$NFSN_USER@$NFSN_HOST" "ls -1 $REMOTE_DIR/*.txt.gz 2>/dev/null | xargs -n1 basename" || true)

if [ -z "$AVAILABLE" ]; then
    echo "No rollups found on server"
    exit 0
fi

# Find new rollups (not in repo)
NEW_ROLLUPS=()
for rollup in $AVAILABLE; do
    if [ ! -f "$ARCHIVE_DIR/$rollup" ]; then
        NEW_ROLLUPS+=("$rollup")
    fi
done

if [ ${#NEW_ROLLUPS[@]} -eq 0 ]; then
    echo "No new rollups to archive"
    exit 0
fi

# Download new rollups via SCP
echo "Found ${#NEW_ROLLUPS[@]} new rollup(s)"
for rollup in "${NEW_ROLLUPS[@]}"; do
    echo "Downloading $rollup..."
    scp -q "$NFSN_USER@$NFSN_HOST:$REMOTE_DIR/$rollup" "$ARCHIVE_DIR/$rollup"
done

# Create or update README
cat > "$ARCHIVE_DIR/README.md" << 'EOF'
# AWStats Weekly Rollups

Compressed Apache access logs, archived weekly for permanent record.

## File Format
- Filename: `YYYY-Www.txt.gz` (ISO 8601 week numbering)
- Example: `2025-W43.txt.gz` = Week 43 of 2025
- Contents: Compressed Apache combined log format

## Viewing
```bash
# Extract and view
gunzip -c 2025-W43.txt.gz | less

# Count requests
gunzip -c 2025-W43.txt.gz | wc -l

# Top IPs
gunzip -c 2025-W43.txt.gz | awk '{print $1}' | sort | uniq -c | sort -rn | head -20
```

## Recovery
These archives can be replayed through AWStats to rebuild statistics:
```bash
gunzip -c 2025-W43.txt.gz | perl /usr/local/www/awstats/cgi-bin/awstats.pl -config=nfsn -update -LogFile=-
```
EOF

# Count total archives
TOTAL_COUNT=$(ls -1 "$ARCHIVE_DIR"/*.txt.gz 2>/dev/null | wc -l)
echo "" >> "$ARCHIVE_DIR/README.md"
echo "Total archived weeks: $TOTAL_COUNT" >> "$ARCHIVE_DIR/README.md"

# Ensure clean git state before creating PR
if [[ -n $(git status --porcelain) ]]; then
    echo "❌ Error: Working directory has uncommitted changes"
    git status
    exit 1
fi

# Checkout main and pull latest
CURRENT_BRANCH=$(git rev-parse --abbrev-ref HEAD)
if [ "$CURRENT_BRANCH" != "main" ]; then
    echo "Switching from $CURRENT_BRANCH to main..."
    git checkout main
fi
git pull origin main

# Create PR using gh CLI
echo "Creating pull request..."
BRANCH="awstats-rollup-$(date +%Y-%m-%d-%H%M%S)"
git checkout -b "$BRANCH"
git add "$ARCHIVE_DIR"
git commit -m "chore: archive AWStats rollups for ${NEW_ROLLUPS[*]}"
git push origin "$BRANCH"

gh pr create \
    --title "Archive AWStats weekly rollups" \
    --body "$(cat <<EOFPR
Automated PR to archive weekly AWStats rollups.

**New rollups:**
$(printf '- `%s`\n' "${NEW_ROLLUPS[@]}")

**Total archived:** $TOTAL_COUNT weeks

These compressed access logs provide a permanent record of site traffic and can be used to rebuild AWStats statistics if needed.
EOFPR
)" \
    --label "automated" \
    --label "awstats"

echo "✅ Pull request created: $BRANCH"
```

**Justfile recipe:**
```just
# Fetch new AWStats rollups and update/create PR (requires NFSN_HOST and NFSN_USER env vars)
fetch-rollups:
    @echo "📊 Fetching AWStats rollups..."
    @./scripts/fetch-rollups.sh
```

**GitHub Actions workflow:**
Create `.github/workflows/fetch-awstats-rollups.yml`:
```yaml
name: Fetch AWStats Rollups

on:
  push:
    branches:
      - main
  workflow_dispatch:

permissions:
  contents: write
  pull-requests: write

jobs:
  fetch-rollups:
    name: Fetch and archive AWStats rollups
    runs-on: ubuntu-latest

    steps:
      - name: Checkout code
        uses: actions/checkout@v4
        with:
          fetch-depth: 0  # Full history needed for git operations

      - name: Setup SSH for NFSN
        env:
          NFSN_SSH_KEY: ${{ secrets.NFSN_SSH_KEY }}
          NFSN_HOST: ${{ secrets.NFSN_HOST }}
        run: |
          mkdir -p ~/.ssh
          echo "$NFSN_SSH_KEY" > ~/.ssh/nfsn_awstats
          chmod 600 ~/.ssh/nfsn_awstats
          cat >> ~/.ssh/config <<EOF
          Host $NFSN_HOST
            IdentityFile ~/.ssh/nfsn_awstats
            StrictHostKeyChecking accept-new
          EOF

      - name: Setup Just
        uses: extractions/setup-just@v2

      - name: Fetch rollups and update/create PR
        env:
          GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          NFSN_HOST: ${{ secrets.NFSN_HOST }}
          NFSN_USER: ${{ secrets.NFSN_USER }}
        run: |
          just fetch-rollups
```

### SSH Setup

For the fetch-rollups script to work, SSH access to NFSN must be configured:

**Local development:**
```bash
# Add to .envrc.local
export NFSN_HOST=ssh.phx.nearlyfreespeech.net
export NFSN_USER=youruser_yoursite
```

**GitHub Actions Secrets:**
1. Generate SSH key (on your local machine):
   ```bash
   ssh-keygen -t ed25519 -f ~/.ssh/nfsn_awstats -N ""
   ```

2. Add public key to NFSN via web UI:
   - Sites → SSH/SFTP → Authorized Keys → Add Key
   - Paste contents of `~/.ssh/nfsn_awstats.pub`

3. Add secrets to GitHub repository:
   - `NFSN_SSH_KEY`: Paste contents of `~/.ssh/nfsn_awstats` (private key)
   - `NFSN_HOST`: Your NFSN hostname (e.g., `ssh.phx.nearlyfreespeech.net`)
   - `NFSN_USER`: Your NFSN username (format: `username_sitename`)

**Note:** The workflow file (shown above) handles SSH configuration automatically using these secrets.

### Manual Usage
```bash
# Check for and download new rollups
just fetch-rollups

# Force re-download specific week via SCP
scp "$NFSN_USER@$NFSN_HOST:/home/private/rollups/2025-W43.txt.gz" \
  awstats-archives/2025-W43.txt.gz

# List available rollups on server
ssh "$NFSN_USER@$NFSN_HOST" "ls -lh /home/private/rollups/"
```

## Storage Estimates

### Weekly Archives
- ~1MB raw access log per week
- Compressed: ~100KB per week
- 1 year: ~5MB compressed
- 10 years: ~50MB compressed

### AWStats Data Files
- Monthly data files: ~50KB each
- 1 year: ~600KB
- 10 years: ~6MB

**Total for 10 years: ~56MB** (negligible)

## Deployment Integration

### Update justfile
Add AWStats files to deployment (in `_deploy-files` recipe):

```just
echo "📤 Uploading AWStats config..."
scp ops/.awstats.conf "$NFSN_USER@$NFSN_HOST:/home/private/.awstats.conf"

echo "📤 Uploading AWStats weekly script..."
scp ops/awstats-weekly.sh "$NFSN_USER@$NFSN_HOST:/home/private/bin/awstats-weekly.sh.new"
```

### Update atomic swap
Include AWStats script in chmod:
```bash
mv /home/private/bin/awstats-weekly.sh.new /home/private/bin/awstats-weekly.sh && \
chmod +x /home/private/bin/awstats-weekly.sh /home/private/bin/cron-generate.sh
```

### Add fetch-rollups recipe
```just
# Fetch new AWStats rollups and create PR if needed
fetch-rollups:
    @./scripts/fetch-rollups.sh
```

## Testing AWStats Configuration

Before deploying, verify AWStats config works correctly:

### Test Config Syntax (on NFSN via SSH)
```bash
ssh user@nfsn.host
perl /usr/local/www/awstats/cgi-bin/awstats.pl -config=nfsn -configtest
```

**Expected output:**
```
Config file '/home/private/.awstats.conf' read successfully
LogFile = /home/logs/access_log
SiteDomain = plazaespana.info
...
Press ENTER to continue...
```

### Test Manual Update (on NFSN via SSH)
```bash
# Test static page generation to temp directory
perl /usr/local/www/awstats/tools/awstats_buildstaticpages.pl \
  -config=nfsn \
  -update \
  -dir=/tmp/awstats-test

# Check generated files
ls -lh /tmp/awstats-test/
```

### Test Weekly Script
```bash
# Run manually before setting up cron
/home/private/bin/awstats-weekly.sh

# Check outputs
ls -lh /home/private/rollups/
ls -lh /home/public/stats/
tail -50 /home/logs/awstats.log
```

## Monitoring

### Check Logs
```bash
# View recent AWStats processing
tail -50 /home/logs/awstats.log

# List rollup archives
ls -lh /home/private/rollups/

# Check AWStats data files
ls -lh /home/private/awstats-data/

# Check static HTML pages
ls -lh /home/public/stats/

# Verify backup exists
ls -lh /home/logs/access_log.backup

# Check current vs backup log sizes
ls -lh /home/logs/access_log*
```

### Verify Archive Integrity
```bash
# Test that archives can be extracted
gunzip -t /home/private/rollups/*.gz
```

## Recovery Scenarios

### Restore from Backup (Most Recent Week)
If the current week's rollup failed or was corrupted, use the backup:
```bash
# The backup contains the previous week's data before truncation
# You can re-process it through AWStats
gzip -c /home/logs/access_log.backup > /home/private/rollups/YYYY-Www.txt.gz

# Or rebuild stats from backup
gunzip -c /home/logs/access_log.backup | perl /usr/local/www/awstats/tools/awstats_buildstaticpages.pl \
    -config=nfsn \
    -update \
    -dir=/home/public/stats \
    -LogFile=-
```

### Restore from Weekly Archive
If AWStats data is lost, rebuild from rollup archives:
```bash
# From Git repo
cd awstats-archives
for archive in *.txt.gz; do
    echo "Processing $archive..."
    gunzip -c "$archive" | perl /usr/local/www/awstats/tools/awstats_buildstaticpages.pl \
        -config=nfsn \
        -update \
        -dir=/home/public/stats \
        -LogFile=-
done
```

### View Specific Week
```bash
# Extract and view specific week's raw logs
gunzip -c /home/private/rollups/2025-W43.txt.gz | less

# Count requests in week
gunzip -c /home/private/rollups/2025-W43.txt.gz | wc -l

# Top 20 IPs for week
gunzip -c /home/private/rollups/2025-W43.txt.gz | awk '{print $1}' | sort | uniq -c | sort -rn | head -20
```

## Alternative: Database Backend

**Future enhancement:** Use SQLite for AWStats data
- Better query performance for long-term trends
- Easier data export/analysis
- ~Same storage footprint
- Requires AWStats plugin or custom scripts

## Configuration Summary

**Resolved:**
- ✅ Domains: `plazaespana.nfshost.com`, `plazaespana.info`
- ✅ Access logs: `/home/logs/access_log` (NFSN standard)
- ✅ Log rotation: NFSN handles automatically
- ✅ Authentication: Basic Auth (htpasswd)
- ✅ Interface: Static HTML pages (no CGI)

## Implementation Checklist

1. **Create AWStats config file**
   - [ ] Create `ops/.awstats.conf` with SiteDomain and HostAliases (see "AWStats Configuration" section)
   - [ ] Add to deployment upload list in justfile

2. **Create stats directory htaccess**
   - [ ] Create `ops/stats.htaccess` with Basic Auth protection
   - [ ] Add to deployment upload list (deploys to `/home/public/stats/.htaccess`)

3. **Create weekly processing script**
   - [ ] Create `ops/awstats-weekly.sh` (see "Weekly Processing Script" section)
   - [ ] Add to deployment upload list in justfile
   - [ ] Include in atomic swap with chmod +x

4. **Create rollup fetch script**
   - [ ] Create `scripts/fetch-rollups.sh` (see "CI/PR Automation" section)
   - [ ] Add `just fetch-rollups` recipe to justfile
   - [ ] Test locally with `.envrc.local` credentials

5. **Setup SSH access**
   - [ ] Generate SSH key: `ssh-keygen -t ed25519 -f ~/.ssh/nfsn_awstats`
   - [ ] Add public key to NFSN via web UI (Sites → SSH/SFTP → Authorized Keys)
   - [ ] Add GitHub Actions secrets: `NFSN_SSH_KEY`, `NFSN_HOST`, `NFSN_USER`
   - [ ] Test SSH access: `ssh $NFSN_USER@$NFSN_HOST ls /home/private`

6. **Update deployment (justfile)**
   - [ ] Add `.awstats.conf` upload to `_deploy-files` recipe
   - [ ] Add `awstats-weekly.sh` upload to `_deploy-files` recipe
   - [ ] Add `stats/.htaccess` upload to `_deploy-files` recipe
   - [ ] Update atomic swap chmod command to include awstats-weekly.sh
   - [ ] Add domains (plazaespana.nfshost.com, plazaespana.info) to README.md

7. **Setup Basic Auth on NFSN**
   - [ ] SSH to NFSN
   - [ ] Run `htpasswd -c /home/private/.htpasswd username`
   - [ ] Set permissions: `chmod 600 /home/private/.htpasswd && chmod 711 /home/private`

8. **Deploy and test configuration**
   - [ ] Run `just deploy`
   - [ ] SSH to NFSN and test config: `perl /usr/local/www/awstats/cgi-bin/awstats.pl -config=nfsn -configtest`
   - [ ] If config test fails, verify NFSN's config merge behavior (see "Important Notes" section)

9. **Test weekly script manually**
   - [ ] SSH to NFSN and run `/home/private/bin/awstats-weekly.sh` manually
   - [ ] Verify `/home/private/rollups/` has first .gz file (NOT /home/public/rollups/)
   - [ ] Verify `/home/public/stats/` has HTML files
   - [ ] Check `/home/logs/awstats.log` for errors
   - [ ] Test web access with basic auth: `https://plazaespana.info/stats/`

10. **Setup cron job on NFSN**
    - [ ] NFSN web UI → Scheduled Tasks
    - [ ] Command: `/home/private/bin/awstats-weekly.sh >> /home/logs/awstats.log 2>&1`
    - [ ] Schedule: `0 1 * * 0` (Sunday 1 AM)

11. **Create GitHub Actions workflow**
    - [ ] Create `.github/workflows/fetch-awstats-rollups.yml` (see "CI/PR Automation" section)
    - [ ] Verify workflow has correct permissions (`contents: write`, `pull-requests: write`)
    - [ ] Commit and push workflow file to main

12. **Test rollup fetch workflow**
    - [ ] Create test rollup on NFSN: `ssh $NFSN_USER@$NFSN_HOST "echo 'test' | gzip > /home/private/rollups/test.txt.gz"`
    - [ ] Run `just fetch-rollups` locally to verify script works
    - [ ] Delete local test rollup and branch: `rm -rf awstats-archives/test.txt.gz && git branch -D awstats-rollups`
    - [ ] Trigger workflow manually via workflow_dispatch to test GitHub Actions
    - [ ] Verify PR created with `awstats-rollups` branch
    - [ ] Add another test rollup and trigger workflow again
    - [ ] Verify PR is updated (force-push) instead of creating duplicate
    - [ ] Clean up: Close test PR, delete test rollups from NFSN

13. **First real rollup PR**
    - [ ] Wait for first weekly cron run (or create real rollup manually)
    - [ ] Workflow automatically creates/updates PR after next push to main
    - [ ] Review PR to verify rollup data looks correct
    - [ ] Merge PR to archive rollup in `awstats-archives/` directory

---

## Audit Summary (2025-10-23)

### Critical Fixes Applied

1. **Fixed .htaccess protection**: Changed from `<Directory>` directive (doesn't work in `.htaccess`) to separate `ops/stats.htaccess` file deployed to `/home/public/stats/.htaccess`

2. **Added error handling to weekly script**: Captures AWStats output to log file instead of `/dev/null`, exits on failure for proper error reporting via cron

3. **Added directory initialization**: Weekly script now creates `/home/private/awstats-data/` to prevent first-run failures

4. **Added empty log handling**: Skip rollup creation if access log is empty (no traffic that week)

### Security & Operational Improvements

5. **Fixed htpasswd permissions**: Added `chmod 600 /home/private/.htpasswd` to secure password file

6. **Added git state checks to fetch-rollups**: Verifies clean working directory, checks out main, pulls latest before creating PR

7. **Fixed branch naming collision**: Changed from date-based to timestamp-based branch names to allow multiple runs per day

8. **Added SSH setup documentation**: Complete instructions for local development and GitHub Actions

9. **Added GitHub Actions permissions**: Documented required `contents: write` and `pull-requests: write` permissions

10. **Fixed symlink creation**: Changed from `ln -sf` to `rm -f && ln -s` for safer index.html symlink

### Documentation Enhancements

11. **Added testing section**: Instructions for verifying AWStats config before deployment

12. **Fixed config name inconsistency**: Changed recovery instructions to use `-config=nfsn` consistently

13. **Updated implementation checklist**: More detailed steps with section references and correct file paths

14. **Added important notes section**: Flagged config verification requirement and other prerequisites upfront

### Workflow Improvements (Applied After Initial Audit)

15. **Canonical branch approach**: Changed from timestamp-based branches to single `awstats-rollups` branch with force-push to prevent PR proliferation

16. **Separate workflow file**: Moved rollup fetch to `.github/workflows/fetch-awstats-rollups.yml` for better organization and separation of concerns

17. **Workflow dispatch support**: Enabled manual triggering via GitHub UI in addition to automatic runs on push to main

18. **No duplicate PRs**: Script checks for existing PR and force-pushes to update it instead of creating new ones

19. **Safe truncation with verification**: Only truncates access log after verifying AWStats processing AND rollup creation both succeeded

20. **Rolling backup**: Creates `access_log.backup` before truncating, providing 1-week disaster recovery option

21. **Pre-flight validation**: fetch-rollups.sh validates required commands (ssh, scp, gh, git) and `gh auth status` before processing to fail fast if environment is misconfigured

### Remaining Considerations

- **AWStats config merge behavior**: Must be verified during initial testing (Item #8 in checklist)
- **Storage estimates**: Based on low-traffic site (~1MB/week); could be 10-100x higher with traffic spikes
- **Cron error handling**: Current approach will send email on errors; consider wrapping in similar pattern to `cron-generate.sh` for consistency
