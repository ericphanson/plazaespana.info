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

        # Create compressed rollup
        if ! gzip -c "$ACCESS_LOG" > "$ROLLUP_DIR/$WEEK.txt.gz"; then
            echo "ERROR: Failed to create rollup archive" >&2
            exit 1
        fi

        # Verify rollup was created successfully
        if [ ! -s "$ROLLUP_DIR/$WEEK.txt.gz" ]; then
            echo "ERROR: Rollup archive is empty or missing" >&2
            exit 1
        fi

        echo "Weekly rollup created: $ROLLUP_DIR/$WEEK.txt.gz ($(stat -f%z "$ROLLUP_DIR/$WEEK.txt.gz" 2>/dev/null || stat -c%s "$ROLLUP_DIR/$WEEK.txt.gz" 2>/dev/null) bytes)" | tee -a "$LOG_FILE"

        # 4. Backup and truncate access log to prevent duplicates in next rollup
        # Keep rolling backup (overwrite previous backup)
        echo "Creating backup and truncating access log..." | tee -a "$LOG_FILE"
        cp "$ACCESS_LOG" "$ACCESS_LOG.backup"

        # Truncate log (NFSN will continue writing to it)
        : > "$ACCESS_LOG"

        echo "Access log backed up and truncated (backup: $ACCESS_LOG.backup)" | tee -a "$LOG_FILE"
    else
        echo "Skipping rollup - access log is empty" | tee -a "$LOG_FILE"
    fi

    echo "Static pages updated: $STATS_DIR/" | tee -a "$LOG_FILE"
else
    echo "ERROR: Access log not found at $ACCESS_LOG" >&2
    exit 1
fi

echo "Completed: $(date)" | tee -a "$LOG_FILE"
