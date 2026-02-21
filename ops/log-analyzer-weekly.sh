#!/bin/bash
# Wrapper script for NFSN cron - runs log-analyzer and writes aggregate stats
set -euo pipefail

BIN=/home/private/bin/log-analyzer
PUBLISH_BIN=/home/private/bin/log-analyzer-publish.py
SOURCE_LOG_DIR=/home/logs
DATA_DIR=/home/private/log-analyzer-data
TMP_DIR_BASE=/tmp
LOG_DIR=/home/logs
LOG_FILE=$LOG_DIR/log-analyzer.log
PUBLIC_ANALYTICS_DIR=/home/public/analytics
LOCK_DIR=$TMP_DIR_BASE/log-analyzer-weekly.lock
LOCK_OWNED=0

mkdir -p "$LOG_DIR" "$DATA_DIR"

echo "=== log-analyzer run started: $(date '+%Y-%m-%d %H:%M:%S %Z') ===" >> "$LOG_FILE"

acquire_lock() {
    if mkdir "$LOCK_DIR" 2>/dev/null; then
        printf '%s\n' "$$" > "$LOCK_DIR/pid"
        LOCK_OWNED=1
        return
    fi

    if [ -f "$LOCK_DIR/pid" ]; then
        old_pid="$(cat "$LOCK_DIR/pid" 2>/dev/null || true)"
        if [ -n "$old_pid" ] && ! kill -0 "$old_pid" 2>/dev/null; then
            rm -rf "$LOCK_DIR"
            mkdir "$LOCK_DIR"
            printf '%s\n' "$$" > "$LOCK_DIR/pid"
            LOCK_OWNED=1
            return
        fi
    fi

    echo "ERROR: another log-analyzer-weekly run appears active (lock: $LOCK_DIR)" >&2
    exit 1
}

if [ ! -x "$BIN" ]; then
    echo "ERROR: log-analyzer binary missing or not executable at $BIN" >&2
    exit 1
fi

if [ ! -x "$PUBLISH_BIN" ]; then
    echo "ERROR: publish script missing or not executable at $PUBLISH_BIN" >&2
    exit 1
fi

if ! command -v python3 >/dev/null 2>&1; then
    echo "ERROR: python3 is required by publish script" >&2
    exit 1
fi

if [ ! -d "$SOURCE_LOG_DIR" ]; then
    echo "ERROR: log directory not found at $SOURCE_LOG_DIR" >&2
    exit 1
fi

TMP_OUT="$(mktemp -d "$TMP_DIR_BASE/log-analyzer.XXXXXX")"
TMP_LOG_DIR="$(mktemp -d "$TMP_DIR_BASE/log-analyzer-logs.XXXXXX")"

unique_dest() {
    local base_path="$1"
    local candidate="$base_path"
    local n=1
    while [ -e "$candidate" ]; do
        candidate="${base_path}.${n}"
        n=$((n + 1))
    done
    printf '%s\n' "$candidate"
}

cleanup() {
    rm -rf "$TMP_OUT" "$TMP_LOG_DIR"
    if [ "$LOCK_OWNED" -eq 1 ]; then
        rm -rf "$LOCK_DIR"
    fi
}
trap cleanup EXIT

acquire_lock

HAVE_LOGS=0
HAVE_GZ=0
for src in "$SOURCE_LOG_DIR"/access_log*; do
    if [ ! -e "$src" ]; then
        continue
    fi
    HAVE_LOGS=1
    base="$(basename "$src")"
    if [[ "$base" == *.gz ]]; then
        HAVE_GZ=1
        if ! command -v gzip >/dev/null 2>&1; then
            echo "ERROR: found .gz logs but gzip is not available" >&2
            exit 1
        fi
        dest="$(unique_dest "$TMP_LOG_DIR/${base%.gz}")"
        if ! gzip -cd "$src" > "$dest"; then
            echo "ERROR: failed to decompress $src" >&2
            exit 1
        fi
    else
        dest="$(unique_dest "$TMP_LOG_DIR/$base")"
        cp "$src" "$dest"
    fi
done

if [ "$HAVE_LOGS" -eq 0 ]; then
    echo "ERROR: no access_log* files found in $SOURCE_LOG_DIR" >&2
    exit 1
fi

if [ "$HAVE_GZ" -eq 1 ]; then
    echo "Detected .gz log files; expanded into $TMP_LOG_DIR" >> "$LOG_FILE"
fi

if ! "$BIN" --mode analyze --out-dir "$TMP_OUT" "$TMP_LOG_DIR" >> "$LOG_FILE" 2>&1; then
    echo "ERROR: log-analyzer failed at $(date '+%Y-%m-%d %H:%M:%S')" >&2
    echo "==================== LOG TAIL ====================" >&2
    tail -200 "$LOG_FILE" >&2 || true
    echo "==================================================" >&2
    exit 1
fi

if ! python3 "$PUBLISH_BIN" "$TMP_OUT" "$DATA_DIR" "$PUBLIC_ANALYTICS_DIR" >> "$LOG_FILE" 2>&1; then
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
