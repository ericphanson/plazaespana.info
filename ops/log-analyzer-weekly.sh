#!/bin/bash
# Wrapper script for NFSN cron - runs log-analyzer and writes aggregate stats
set -euo pipefail

BIN=/home/private/bin/log-analyzer
PUBLISH_BIN=/home/private/bin/log-analyzer-publish.sh
SOURCE_LOG_DIR=/home/logs
DATA_DIR=/home/private/log-analyzer-data
TMP_DIR_BASE=/tmp
LOG_DIR=/home/logs
LOG_FILE=$LOG_DIR/log-analyzer.log
PUBLIC_ANALYTICS_DIR=/home/public/analytics

mkdir -p "$LOG_DIR" "$DATA_DIR"

echo "=== log-analyzer run started: $(date '+%Y-%m-%d %H:%M:%S %Z') ===" >> "$LOG_FILE"

if [ ! -x "$BIN" ]; then
    echo "ERROR: log-analyzer binary missing or not executable at $BIN" >&2
    exit 1
fi

if [ ! -x "$PUBLISH_BIN" ]; then
    echo "ERROR: publish script missing or not executable at $PUBLISH_BIN" >&2
    exit 1
fi

if [ ! -d "$SOURCE_LOG_DIR" ]; then
    echo "ERROR: log directory not found at $SOURCE_LOG_DIR" >&2
    exit 1
fi

TMP_OUT="$(mktemp -d "$TMP_DIR_BASE/log-analyzer.XXXXXX")"

cleanup() {
    rm -rf "$TMP_OUT"
}
trap cleanup EXIT

if ! "$BIN" --out-dir "$TMP_OUT" "$SOURCE_LOG_DIR" >> "$LOG_FILE" 2>&1; then
    echo "ERROR: log-analyzer failed at $(date '+%Y-%m-%d %H:%M:%S')" >&2
    echo "==================== LOG TAIL ====================" >&2
    tail -200 "$LOG_FILE" >&2 || true
    echo "==================================================" >&2
    exit 1
fi

if ! "$PUBLISH_BIN" "$TMP_OUT" "$DATA_DIR" "$PUBLIC_ANALYTICS_DIR" >> "$LOG_FILE" 2>&1; then
    echo "ERROR: publish step failed at $(date '+%Y-%m-%d %H:%M:%S')" >&2
    echo "==================== LOG TAIL ====================" >&2
    tail -200 "$LOG_FILE" >&2 || true
    echo "==================================================" >&2
    exit 1
fi

echo "Wrote aggregate stats to: $DATA_DIR" >> "$LOG_FILE"
echo "Published analytics to: $PUBLIC_ANALYTICS_DIR" >> "$LOG_FILE"
echo "=== log-analyzer run completed: $(date '+%Y-%m-%d %H:%M:%S %Z') ===" >> "$LOG_FILE"
echo "" >> "$LOG_FILE"

exit 0
