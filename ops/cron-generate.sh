#!/bin/bash
# Wrapper script for NFSN cron - logs all output, only emails on errors
set -euo pipefail

LOG_DIR=/home/logs
LOG_FILE=$LOG_DIR/generate.log
EMAIL_TAIL_LINES=${GENERATE_EMAIL_TAIL_LINES:-200}
EMAIL_FALLBACK_LINES=${GENERATE_EMAIL_FALLBACK_LINES:-40}

# Ensure log directory exists
mkdir -p "$LOG_DIR"

# Guard against invalid or empty tail settings.
case "$EMAIL_TAIL_LINES" in
    ''|*[!0-9]*)
        EMAIL_TAIL_LINES=200
        ;;
esac
if [ "$EMAIL_TAIL_LINES" -le 0 ]; then
    EMAIL_TAIL_LINES=200
fi

case "$EMAIL_FALLBACK_LINES" in
    ''|*[!0-9]*)
        EMAIL_FALLBACK_LINES=40
        ;;
esac
if [ "$EMAIL_FALLBACK_LINES" -le 0 ]; then
    EMAIL_FALLBACK_LINES=40
fi

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

    # Build failed - output a bounded tail to stderr to trigger email.
    # Full history remains in $LOG_FILE on disk.
    echo "=== Build failed: $(date '+%Y-%m-%d %H:%M:%S %Z') ===" >> "$LOG_FILE"
    echo "ERROR: Build failed at $(date '+%Y-%m-%d %H:%M:%S')" >&2
    log_tail="$(tail -n "$EMAIL_TAIL_LINES" "$LOG_FILE" 2>/dev/null || true)"
    error_summary="$(printf '%s\n' "$log_tail" | grep -Ei '(error|fatal|panic|traceback|exception|failed)' | tail -n 1 || true)"
    if [ -n "$error_summary" ]; then
        echo "Error summary: $error_summary" >&2
    else
        echo "No explicit error line found; showing last $EMAIL_FALLBACK_LINES log lines." >&2
        echo "==================== LOG TAIL ====================" >&2
        printf '%s\n' "$log_tail" | tail -n "$EMAIL_FALLBACK_LINES" >&2 || true
        echo "==================================================" >&2
    fi
    echo "Full log: $LOG_FILE" >&2
    exit 1
fi

# Log completion time
echo "=== Build completed: $(date '+%Y-%m-%d %H:%M:%S %Z') ===" >> "$LOG_FILE"
echo "" >> "$LOG_FILE"

# Success - no output to stderr, so no cron email
exit 0
