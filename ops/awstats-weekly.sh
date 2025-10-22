#!/bin/bash
# Weekly AWStats processing and static page generation
set -euo pipefail

AWSTATS_STATIC=/usr/local/www/awstats/tools/awstats_buildstaticpages.pl
STATS_DIR=/home/public/stats
DATA_DIR=/home/private/awstats-data
ACCESS_LOG=/home/logs/access_log
LOG_FILE=/home/logs/awstats.log

# Ensure directories exist
mkdir -p "$STATS_DIR" "$DATA_DIR"

echo "=== AWStats Weekly Processing ===" | tee -a "$LOG_FILE"
echo "Started: $(date)" | tee -a "$LOG_FILE"

# 1. Update AWStats database and generate static pages
# Use explicit config path to avoid NFSN merge issues
if [ -f "$ACCESS_LOG" ]; then
    echo "Updating AWStats database and generating static pages..." | tee -a "$LOG_FILE"

    if ! perl "$AWSTATS_STATIC" \
        -configdir=/home/private \
        -config=awstats \
        -update \
        -dir="$STATS_DIR" \
        2>&1 | tee -a "$LOG_FILE"; then
        echo "ERROR: AWStats processing failed" >&2
        exit 1
    fi

    # 2. Create symlink for clean index.html access
    cd "$STATS_DIR"
    rm -f index.html
    ln -s awstats.awstats.html index.html

    echo "Static pages updated: $STATS_DIR/" | tee -a "$LOG_FILE"
    echo "Database files in $DATA_DIR will be synced to git via GitHub Actions" | tee -a "$LOG_FILE"
else
    echo "ERROR: Access log not found at $ACCESS_LOG" >&2
    exit 1
fi

echo "Completed: $(date)" | tee -a "$LOG_FILE"
