# AWStats Database Files

Synchronized copy of AWStats aggregate statistics database.

## Privacy

**Important:** These files are configured to contain **aggregate statistics only**:
- ✅ Total page views, unique visitors (counts)
- ✅ Referrer statistics (which sites link to us, aggregated)
- ✅ Browser, OS, country statistics (aggregated percentages)
- ✅ Monthly trends and historical data
- ❌ **No individual IP addresses** (configured with `MaxNbOfHostsShown=0`)
- ❌ **No individual requests**
- ❌ **No personal information**

This is enforced via AWStats configuration (`ops/awstats.conf`):
```perl
MaxNbOfHostsShown=0      # Don't store individual IPs
MaxNbOfLoginShown=0      # Don't track individual users
MaxNbOfRobotShown=0      # Don't track individual bots
DNSLookup=0              # Don't do DNS lookups
ShowAuthenticatedUsers=0 # Don't track logged-in users
```

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
