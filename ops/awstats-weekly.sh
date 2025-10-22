#!/bin/bash
# Wrapper script for NFSN cron - logs all output, only emails on errors
set -euo pipefail

AWSTATS_STATIC=/usr/local/www/awstats/tools/awstats_buildstaticpages.pl
STATS_DIR=/home/public/stats
DATA_DIR=/home/private/awstats-data
ACCESS_LOG=/home/logs/access_log
LOG_DIR=/home/logs
LOG_FILE=$LOG_DIR/awstats.log

# Ensure directories exist
mkdir -p "$STATS_DIR" "$DATA_DIR" "$LOG_DIR"

# Log start time
echo "=== AWStats Weekly Processing started: $(date '+%Y-%m-%d %H:%M:%S %Z') ===" >> "$LOG_FILE"

# Check if access log exists
if [ ! -f "$ACCESS_LOG" ]; then
    echo "ERROR: Access log not found at $ACCESS_LOG at $(date '+%Y-%m-%d %H:%M:%S')" >&2
    echo "==================== FULL LOG ====================" >&2
    cat "$LOG_FILE" >&2
    echo "==================================================" >&2
    exit 1
fi

# Update AWStats database and generate static pages
# Use explicit config path to avoid NFSN merge issues
# Capture all output to log file
if ! perl "$AWSTATS_STATIC" \
    -configdir=/home/private \
    -config=awstats \
    -update \
    -dir="$STATS_DIR" >> "$LOG_FILE" 2>&1; then

    # AWStats processing failed - output full log to stderr to trigger email
    echo "ERROR: AWStats processing failed at $(date '+%Y-%m-%d %H:%M:%S')" >&2
    echo "==================== FULL LOG ====================" >&2
    cat "$LOG_FILE" >&2
    echo "==================================================" >&2
    exit 1
fi

# Create symlink for clean index.html access
cd "$STATS_DIR"
rm -f index.html
ln -s awstats.awstats.html index.html

# Log completion
echo "Static pages updated: $STATS_DIR/" >> "$LOG_FILE"
echo "Database files in $DATA_DIR will be synced to git via GitHub Actions" >> "$LOG_FILE"
echo "=== AWStats Weekly Processing completed: $(date '+%Y-%m-%d %H:%M:%S %Z') ===" >> "$LOG_FILE"
echo "" >> "$LOG_FILE"

# Success - no output to stderr, so no cron email
exit 0
