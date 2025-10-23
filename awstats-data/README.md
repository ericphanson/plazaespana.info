# AWStats Database Files

Synchronized copy of AWStats aggregate statistics database.

## Privacy

**Important:** These files contain **aggregate statistics only**. Individual visitor data is automatically stripped before committing to git.

**What's included:**
- ✅ Total page views, unique visitors (counts)
- ✅ Referrer statistics (which sites link to us, aggregated)
- ✅ Browser, OS, country statistics (aggregated percentages)
- ✅ Monthly trends and historical data (by day, by hour)
- ✅ Domain statistics (which domains visit us, aggregated)

**What's removed:**
- ❌ **Individual IP addresses** (BEGIN_VISITOR sections stripped)
- ❌ **Individual robot IPs** (BEGIN_ROBOT sections stripped)
- ❌ **Individual requests** (per-IP tracking removed)
- ❌ **Session details** (per-visitor page views stripped)

**How it works:**

1. AWStats processes logs on the server and creates full database files
2. The sync script (`scripts/fetch-stats-archives.sh`) downloads the files
3. Before committing, it strips out these sections containing personal data:
   - `BEGIN_VISITOR / END_VISITOR` - Individual visitor IPs and sessions
   - `BEGIN_ROBOT / END_ROBOT` - Individual bot/crawler IPs
   - `BEGIN_SIDER_*` - Individual search engine bot IPs
   - `BEGIN_WORMS / BEGIN_EMAILSENDER / BEGIN_EMAILRECEIVER` - Individual spam/attack IPs
4. Only aggregate statistical sections are committed to git

This ensures **no personal information** is ever stored in the repository while preserving all meaningful traffic statistics.

## File Format

- **Monthly stats:** `awstatsMMYYYY.awstats.txt` (e.g., `awstats102025.awstats.txt` for October 2025)
- **DNS cache:** `dnscachelastupdate.awstats.txt` (cached DNS lookups, if any)
- **Other files:** Various AWStats state files

## Viewing Statistics

These are AWStats internal database files (text format). To view human-readable statistics:

1. Visit the live stats page: https://plazaespana.info/stats/
2. Or regenerate HTML locally:
   ```bash
   perl /usr/local/www/awstats/tools/awstats_buildstaticpages.pl \
     -configdir=. -config=awstats -update -dir=./html-output
   ```

## Recovery

To restore from this backup (e.g., after server failure):

```bash
# Copy to production location (NFSN)
scp awstats-data/*.txt user@host:/home/private/awstats-data/

# Regenerate HTML from database
ssh user@host '/home/private/bin/awstats-weekly.sh'
```

## Automatic Sync

These files are automatically synced from the production server via GitHub Actions workflow (`.github/workflows/fetch-awstats-archives.yml`). The workflow:
- Runs after each push to `main` or via manual trigger
- Downloads `.txt` files from `/home/private/awstats-data/` on the server
- Creates/updates a PR with changes
- Uses a canonical branch (`awstats-data`) that is force-pushed

Merge the PR to preserve the latest statistics in git history.
