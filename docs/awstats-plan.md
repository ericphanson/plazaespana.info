# AWStats Integration Plan

## Goal
Set up AWStats to track weekly traffic statistics indefinitely, archive rollups in-repo via automated PRs, and serve static stats pages.

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
  access_log              # Current Apache access log (we truncate weekly)

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
ACCESS_LOG=/home/logs/access_log

# Ensure directories exist
mkdir -p "$ROLLUP_DIR" "$STATS_DIR"

# Get current week number (YYYY-Www format)
WEEK=$(date +%Y-W%V)

echo "=== AWStats Weekly Processing: $WEEK ==="
echo "Started: $(date)"

# 1. Update AWStats database and generate static pages
# NFSN uses -config=nfsn which merges /home/private/.awstats.conf
if [ -f "$ACCESS_LOG" ]; then
    echo "Updating AWStats database and generating static pages..."
    perl "$AWSTATS_STATIC" \
        -config=nfsn \
        -update \
        -dir="$STATS_DIR" \
        > /dev/null 2>&1

    # 2. Create symlink for clean index.html access
    cd "$STATS_DIR"
    ln -sf awstats.nfsn.html index.html

    # 3. Create weekly rollup (compressed access log)
    echo "Creating weekly rollup: $WEEK.txt.gz"
    gzip -c "$ACCESS_LOG" > "$ROLLUP_DIR/$WEEK.txt.gz"

    # 4. Truncate access log to prevent duplicates in next rollup
    # NFSN will continue writing to it
    echo "Truncating access log..."
    > "$ACCESS_LOG"

    echo "Weekly rollup created: $ROLLUP_DIR/$WEEK.txt.gz"
    echo "Static pages updated: $STATS_DIR/"
else
    echo "Warning: Access log not found at $ACCESS_LOG"
fi

echo "Completed: $(date)"
```

**Key changes:**
- Uses NFSN's `-config=nfsn` flag (merges with `/home/private/.awstats.conf`)
- Single command updates database + generates static pages
- Creates `index.html` symlink for clean URLs
- Stores rollups in `/home/private/rollups/` (not public)
- **Truncates access_log after archiving** to prevent duplicates
- Each weekly rollup contains exactly one week's data

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
In `/home/public/.htaccess`:
```apache
# Protect /stats/ directory with basic auth
<Directory /home/public/stats>
  AuthType Basic
  AuthName "Site Statistics"
  AuthUserFile /home/private/.htpasswd
  Require valid-user
</Directory>
```

### Create Password File
```bash
ssh user@nfsn.host
htpasswd -c /home/private/.htpasswd yourusername
# Enter password when prompted
```

## CI/PR Automation for Weekly Rollups

### Overview
After successful deployment, CI checks for new weekly rollups on the server via SCP and creates a PR to archive them in the repo.

### Workflow
1. Deploy succeeds (GitHub Actions or manual)
2. CI runs `just fetch-rollups` (or automatic step in CI)
3. Script uses SCP to list and download new rollup files from `/home/private/rollups/`
4. If new files exist, creates PR with:
   - New `awstats-archives/YYYY-Www.txt.gz` files
   - Updated `awstats-archives/README.md`
   - Descriptive commit message

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
gunzip -c 2025-W43.txt.gz | perl awstats.pl -config=plazaespana -update -LogFile=-
```
EOF

# Count total archives
TOTAL_COUNT=$(ls -1 "$ARCHIVE_DIR"/*.txt.gz 2>/dev/null | wc -l)
echo "" >> "$ARCHIVE_DIR/README.md"
echo "Total archived weeks: $TOTAL_COUNT" >> "$ARCHIVE_DIR/README.md"

# Create PR using gh CLI
echo "Creating pull request..."
BRANCH="awstats-rollup-$(date +%Y-%m-%d)"
git checkout -b "$BRANCH"
git add "$ARCHIVE_DIR"
git commit -m "chore: archive AWStats rollups for ${NEW_ROLLUPS[*]}"

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

echo "Pull request created: $BRANCH"
```

**Justfile recipe:**
```just
# Fetch new AWStats rollups and create PR if needed
fetch-rollups:
    @./scripts/fetch-rollups.sh
```

**GitHub Actions integration:**
Add to `.github/workflows/ci.yml` deploy job:
```yaml
- name: Check for new AWStats rollups
  if: github.event_name == 'push' && github.ref == 'refs/heads/main'
  run: |
    just fetch-rollups
  env:
    GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

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
```

### Verify Archive Integrity
```bash
# Test that archives can be extracted
gunzip -t /home/private/rollups/*.gz
```

## Recovery Scenarios

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
   - [ ] Create `ops/.awstats.conf` with SiteDomain and HostAliases
   - [ ] Add to deployment upload list

2. **Create weekly processing script**
   - [ ] Create `ops/awstats-weekly.sh`
   - [ ] Add to deployment upload list
   - [ ] Include in atomic swap with chmod +x

3. **Create rollup fetch script**
   - [ ] Create `scripts/fetch-rollups.sh`
   - [ ] Add `just fetch-rollups` recipe
   - [ ] Test locally

4. **Update deployment**
   - [ ] Add .awstats.conf upload to justfile
   - [ ] Add awstats-weekly.sh upload to justfile
   - [ ] Update chmod command for both scripts
   - [ ] Add domains to README.md

5. **Update .htaccess**
   - [ ] Add `/stats/` directory protection (Basic Auth)
   - [ ] Deploy .htaccess changes

6. **Setup Basic Auth on NFSN**
   - [ ] SSH to NFSN
   - [ ] Run `htpasswd -c /home/private/.htpasswd username`
   - [ ] Set permissions: `chmod 711 /home/private`

7. **Deploy and test**
   - [ ] Run `just deploy`
   - [ ] SSH to NFSN and run `/home/private/bin/awstats-weekly.sh` manually
   - [ ] Verify `/home/public/rollups/` has first .gz file
   - [ ] Verify `/home/public/stats/` has HTML files
   - [ ] Test web access with basic auth

8. **Setup cron job on NFSN**
   - [ ] NFSN web UI → Scheduled Tasks
   - [ ] Command: `/home/private/bin/awstats-weekly.sh >> /home/logs/awstats.log 2>&1`
   - [ ] Schedule: `0 1 * * 0` (Sunday 1 AM)

9. **Setup CI automation**
   - [ ] Add `fetch-rollups` step to GitHub Actions
   - [ ] Test PR creation with mock rollup

10. **First rollup PR**
    - [ ] Wait for first weekly cron run
    - [ ] Run `just fetch-rollups` to create PR
    - [ ] Review and merge PR
    - [ ] Verify rollup in `awstats-archives/`
