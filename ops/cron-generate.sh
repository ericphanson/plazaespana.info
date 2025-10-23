#!/bin/bash
# Wrapper script for NFSN cron - logs all output, only emails on errors
set -euo pipefail

LOG_DIR=/home/logs
LOG_FILE=$LOG_DIR/generate.log

# Ensure log directory exists
mkdir -p "$LOG_DIR"

# Log start time
echo "=== Build started: $(date '+%Y-%m-%d %H:%M:%S %Z') ===" >> "$LOG_FILE"

# Run buildsite, capturing all output to log file
# If it fails (non-zero exit), output to stderr to trigger cron email
if ! /home/private/bin/buildsite \
  -config /home/private/config.toml \
  -out-dir /home/public \
  -data-dir /home/private/data \
  -template-path /home/private/templates/index.tmpl.html \
  -fetch-mode production >> "$LOG_FILE" 2>&1; then

    # Build failed - output full log to stderr to trigger email
    echo "ERROR: Build failed at $(date '+%Y-%m-%d %H:%M:%S')" >&2
    echo "==================== FULL LOG ====================" >&2
    cat "$LOG_FILE" >&2
    echo "==================================================" >&2
    exit 1
fi

# Log completion time
echo "=== Build completed: $(date '+%Y-%m-%d %H:%M:%S %Z') ===" >> "$LOG_FILE"
echo "" >> "$LOG_FILE"

# Success - no output to stderr, so no cron email
exit 0
