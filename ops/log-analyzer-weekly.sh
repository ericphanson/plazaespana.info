#!/bin/bash
# Wrapper script for NFSN cron - runs log-analyzer and writes aggregate stats
set -euo pipefail

BIN=/home/private/bin/log-analyzer
SOURCE_LOG_DIR=/home/logs
DATA_DIR=/home/private/log-analyzer-data
TMP_DIR_BASE=/tmp
LOG_DIR=/home/logs
LOG_FILE=$LOG_DIR/log-analyzer.log

mkdir -p "$LOG_DIR" "$DATA_DIR"

echo "=== log-analyzer run started: $(date '+%Y-%m-%d %H:%M:%S %Z') ===" >> "$LOG_FILE"

if [ ! -x "$BIN" ]; then
    echo "ERROR: log-analyzer binary missing or not executable at $BIN" >&2
    exit 1
fi

if [ ! -d "$SOURCE_LOG_DIR" ]; then
    echo "ERROR: log directory not found at $SOURCE_LOG_DIR" >&2
    exit 1
fi

TMP_OUT="$(mktemp -d "$TMP_DIR_BASE/log-analyzer.XXXXXX")"
STAGE_DIR="${DATA_DIR}.new"
BACKUP_DIR="${DATA_DIR}.old"

cleanup() {
    rm -rf "$TMP_OUT" "$STAGE_DIR"
}
trap cleanup EXIT

if ! "$BIN" --out-dir "$TMP_OUT" "$SOURCE_LOG_DIR" >> "$LOG_FILE" 2>&1; then
    echo "ERROR: log-analyzer failed at $(date '+%Y-%m-%d %H:%M:%S')" >&2
    echo "==================== FULL LOG ====================" >&2
    cat "$LOG_FILE" >&2
    echo "==================================================" >&2
    exit 1
fi

mkdir -p "$STAGE_DIR"
cp "$TMP_OUT"/*.json "$STAGE_DIR"/
cp "$TMP_OUT"/report.html "$STAGE_DIR"/

rm -rf "$BACKUP_DIR"
if [ -d "$DATA_DIR" ]; then
    mv "$DATA_DIR" "$BACKUP_DIR"
fi
mv "$STAGE_DIR" "$DATA_DIR"
rm -rf "$BACKUP_DIR"

echo "Wrote aggregate stats to: $DATA_DIR" >> "$LOG_FILE"
echo "=== log-analyzer run completed: $(date '+%Y-%m-%d %H:%M:%S %Z') ===" >> "$LOG_FILE"
echo "" >> "$LOG_FILE"

exit 0
