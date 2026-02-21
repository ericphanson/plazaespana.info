#!/bin/bash
# Publish log-analyzer outputs:
# 1) validate privacy constraints
# 2) merge with persisted month history
# 3) rebuild lifetime.json from persisted month files
# 4) publish to Apache-served path
# 5) mirror JSON backup to Bunny with immutable-month safeguards
set -euo pipefail
export LC_ALL=C

GENERATED_DIR="${1:-}"
DATA_DIR="${2:-/home/private/log-analyzer-data}"
PUBLIC_ANALYTICS_DIR="${3:-/home/public/analytics}"
LOG_FILE="${LOG_FILE:-/home/logs/log-analyzer.log}"

if [ -z "$GENERATED_DIR" ]; then
    echo "Usage: $0 <generated-dir> [data-dir] [public-analytics-dir]" >&2
    exit 1
fi

if [ ! -d "$GENERATED_DIR" ]; then
    echo "ERROR: generated output directory not found: $GENERATED_DIR" >&2
    exit 1
fi

for cmd in python3 curl hostname; do
    if ! command -v "$cmd" >/dev/null 2>&1; then
        echo "ERROR: required command not found: $cmd" >&2
        exit 1
    fi
done

GRACE_DAYS="${MONTH_CLOSE_GRACE_DAYS:-7}"
BUNNY_BACKUP_REQUIRED="${BUNNY_BACKUP_REQUIRED:-1}"
BUNNY_BACKUP_REPORT="${BUNNY_BACKUP_REPORT:-0}"
BUNNY_KEY_FILE="/home/private/bunny-storage-key.txt"
BUNNY_ZONE_FILE="/home/private/bunny-storage-zone.txt"
BUNNY_ENDPOINT_FILE="/home/private/bunny-storage-endpoint.txt"
BUNNY_BASE_PATH_FILE="/home/private/bunny-base-path.txt"

mkdir -p "$(dirname "$LOG_FILE")"

log() {
    echo "$*" | tee -a "$LOG_FILE"
}

fail() {
    log "ERROR: $*"
    exit 1
}

is_month_filename() {
    [[ "$1" =~ ^[0-9]{4}-[0-9]{2}\.json$ ]]
}

month_is_mutable() {
    local month="$1"
    python3 - "$month" "$GRACE_DAYS" <<'PY'
import datetime
import sys

month = sys.argv[1]
grace_days = int(sys.argv[2])
today = datetime.datetime.now(datetime.timezone.utc).date()
current = today.strftime("%Y-%m")
if month == current:
    sys.exit(0)
first_this_month = today.replace(day=1)
last_prev = first_this_month - datetime.timedelta(days=1)
prev = last_prev.strftime("%Y-%m")
if month == prev and today.day <= grace_days:
    sys.exit(0)
sys.exit(1)
PY
}

json_equal_ignoring_generated_at() {
    local lhs="$1"
    local rhs="$2"
    python3 - "$lhs" "$rhs" <<'PY'
import json
import sys

lhs_path = sys.argv[1]
rhs_path = sys.argv[2]

def normalize(obj):
    if isinstance(obj, dict):
        return {k: normalize(v) for k, v in obj.items() if k != "generated_at"}
    if isinstance(obj, list):
        return [normalize(v) for v in obj]
    return obj

with open(lhs_path, "r", encoding="utf-8") as f:
    lhs = normalize(json.load(f))
with open(rhs_path, "r", encoding="utf-8") as f:
    rhs = normalize(json.load(f))

sys.exit(0 if lhs == rhs else 1)
PY
}

privacy_check_dir() {
    local dir="$1"
    local include_report="$2"
    python3 - "$dir" "$include_report" <<'PY'
import ipaddress
import json
import pathlib
import re
import sys

root = pathlib.Path(sys.argv[1])
include_report = sys.argv[2] == "1"

json_files = sorted(root.glob("*.json"))
if not json_files:
    print(f"No JSON files found under {root}", file=sys.stderr)
    sys.exit(1)

files_to_scan = list(json_files)
report_path = root / "report.html"
if include_report and report_path.exists():
    files_to_scan.append(report_path)

ipv4_re = re.compile(r"(?<![\d.])(?:\d{1,3}\.){3}\d{1,3}(?![\d.])")
ipv6_candidate_re = re.compile(r"(?i)(?<![0-9a-f:])[0-9a-f:]{2,39}(?![0-9a-f:])")

def find_ip_match(text):
    for m in ipv4_re.finditer(text):
        cand = m.group(0)
        try:
            ip = ipaddress.ip_address(cand)
        except ValueError:
            continue
        if isinstance(ip, ipaddress.IPv4Address):
            return f"IPv4:{cand}"
    for m in ipv6_candidate_re.finditer(text):
        cand = m.group(0)
        if ":" not in cand:
            continue
        try:
            ip = ipaddress.ip_address(cand)
        except ValueError:
            continue
        if isinstance(ip, ipaddress.IPv6Address):
            return f"IPv6:{cand}"
    return None

def scan_paths(node, path):
    if isinstance(node, dict):
        for k, v in node.items():
            if k == "path" and isinstance(v, str) and "?" in v:
                raise ValueError(f"{path}: path contains query string: {v}")
            scan_paths(v, path)
    elif isinstance(node, list):
        for item in node:
            scan_paths(item, path)

for fp in files_to_scan:
    text = fp.read_text(encoding="utf-8", errors="replace")
    ip_match = find_ip_match(text)
    if ip_match:
        raise ValueError(f"{fp}: potential IP address found: {ip_match}")

for fp in json_files:
    try:
        data = json.loads(fp.read_text(encoding="utf-8"))
    except json.JSONDecodeError as e:
        raise ValueError(f"{fp}: invalid JSON: {e}") from e
    scan_paths(data, fp)

PY
}

bunny_sync() {
    local merged_dir="$1"

    if [ ! -f "$BUNNY_KEY_FILE" ] || [ ! -f "$BUNNY_ZONE_FILE" ] || [ ! -f "$BUNNY_ENDPOINT_FILE" ]; then
        if [ "$BUNNY_BACKUP_REQUIRED" = "1" ]; then
            fail "Bunny backup required, but one or more Bunny config files are missing in /home/private"
        fi
        log "Bunny backup not configured; skipping backup mirror."
        return 0
    fi

    local bunny_key bunny_zone bunny_endpoint bunny_base_path
    bunny_key="$(tr -d '\r\n' < "$BUNNY_KEY_FILE")"
    bunny_zone="$(tr -d '\r\n' < "$BUNNY_ZONE_FILE")"
    bunny_endpoint="$(tr -d '\r\n' < "$BUNNY_ENDPOINT_FILE")"
    if [ -f "$BUNNY_BASE_PATH_FILE" ]; then
        bunny_base_path="$(tr -d '\r\n' < "$BUNNY_BASE_PATH_FILE")"
    else
        bunny_base_path="analytics-backup/current"
    fi

    [ -n "$bunny_key" ] || fail "Bunny storage key file is empty"
    [ -n "$bunny_zone" ] || fail "Bunny storage zone file is empty"
    [ -n "$bunny_endpoint" ] || fail "Bunny storage endpoint file is empty"
    [ -n "$bunny_base_path" ] || fail "Bunny base path is empty"

    bunny_endpoint="${bunny_endpoint#http://}"
    bunny_endpoint="${bunny_endpoint#https://}"
    bunny_endpoint="${bunny_endpoint%/}"
    bunny_base_path="${bunny_base_path#/}"
    bunny_base_path="${bunny_base_path%/}"

    local remote_tmp
    remote_tmp="$(mktemp -d /tmp/log-analyzer-bunny.XXXXXX)"

    bunny_url_for() {
        local rel="$1"
        printf 'https://%s/%s/%s' "$bunny_endpoint" "$bunny_zone" "$rel"
    }

    bunny_download_if_exists() {
        local rel="$1"
        local out="$2"
        local url code
        url="$(bunny_url_for "$rel")"
        code="$(curl -sS -o "$out" -w "%{http_code}" -H "AccessKey: $bunny_key" "$url")"
        if [ "$code" = "200" ]; then
            return 0
        fi
        rm -f "$out"
        if [ "$code" = "404" ]; then
            return 1
        fi
        fail "Bunny download failed for $rel (HTTP $code)"
    }

    bunny_upload() {
        local local_file="$1"
        local rel="$2"
        local url code
        url="$(bunny_url_for "$rel")"
        code="$(curl -sS -o /dev/null -w "%{http_code}" -X PUT -H "AccessKey: $bunny_key" --data-binary @"$local_file" "$url")"
        if [ "$code" != "200" ] && [ "$code" != "201" ]; then
            fail "Bunny upload failed for $rel (HTTP $code)"
        fi
    }

    local json_files=("$merged_dir"/*.json)
    local report_file="$merged_dir/report.html"
    local mirrored=0
    local immutable_verified=0

    for local_file in "${json_files[@]}"; do
        local name rel month mutable remote_copy
        name="$(basename "$local_file")"
        rel="$bunny_base_path/$name"
        mutable=1
        if is_month_filename "$name"; then
            month="${name%.json}"
            if month_is_mutable "$month"; then
                mutable=1
            else
                mutable=0
            fi
        fi

        remote_copy="$remote_tmp/$name"
        if [ "$mutable" -eq 0 ]; then
            if bunny_download_if_exists "$rel" "$remote_copy"; then
                if ! json_equal_ignoring_generated_at "$local_file" "$remote_copy"; then
                    fail "Immutable month differs from existing Bunny backup: $name"
                fi
                immutable_verified=$((immutable_verified + 1))
                continue
            fi
        fi

        bunny_upload "$local_file" "$rel"
        mirrored=$((mirrored + 1))
    done

    if [ "$BUNNY_BACKUP_REPORT" = "1" ] && [ -f "$report_file" ]; then
        bunny_upload "$report_file" "$bunny_base_path/report.html"
        mirrored=$((mirrored + 1))
    fi

    log "Bunny backup complete: uploaded/updated=$mirrored immutable-verified=$immutable_verified base=$bunny_base_path"
    rm -rf "$remote_tmp"
}

build_lifetime_and_manifest() {
    local merged_dir="$1"
    local generated_lifetime="$2"
    local source_host="$3"

    python3 - "$merged_dir" "$generated_lifetime" "$source_host" <<'PY'
import base64
import datetime as dt
import glob
import hashlib
import json
import math
import os
import pathlib
import sys

merged_dir = pathlib.Path(sys.argv[1])
generated_lifetime = pathlib.Path(sys.argv[2])
source_host = sys.argv[3]

hll_precision = 14
hll_num_buckets = 1 << hll_precision
hll_max_zeros = 32 - hll_precision
hll_version = 1
hll_magic = b"HLL"
sparse_threshold = 3412

def decode_hll_b64(s):
    raw = base64.b64decode(s.encode("ascii"))
    if len(raw) < 6:
        raise ValueError("HLL buffer too small")
    if raw[:3] != hll_magic:
        raise ValueError("Invalid HLL magic")
    if raw[3] != hll_version:
        raise ValueError("Unsupported HLL version")
    precision = raw[4]
    if precision != hll_precision:
        raise ValueError(f"Unexpected HLL precision {precision}")
    fmt = raw[5]
    buckets = [0] * hll_num_buckets
    if fmt == 1:
        if len(raw) < 8:
            raise ValueError("Sparse HLL buffer too small")
        count = raw[6] | (raw[7] << 8)
        pos = 8
        for _ in range(count):
            if pos + 2 >= len(raw):
                raise ValueError("Sparse HLL truncated")
            idx = raw[pos] | (raw[pos + 1] << 8)
            val = raw[pos + 2] & 0x1F
            if idx >= hll_num_buckets:
                raise ValueError("Sparse HLL index out of range")
            buckets[idx] = val
            pos += 3
    elif fmt == 0:
        bit_pos = 0
        bucket_idx = 0
        offset = 6
        while bucket_idx < hll_num_buckets:
            byte_pos = offset + (bit_pos // 8)
            bit_offset = bit_pos % 8
            bits_in_first = 8 - bit_offset
            if byte_pos >= len(raw):
                raise ValueError("Dense HLL truncated")
            if bits_in_first >= 5:
                value = (raw[byte_pos] >> (bits_in_first - 5)) & 0x1F
            else:
                if byte_pos + 1 >= len(raw):
                    raise ValueError("Dense HLL truncated")
                first_bits = raw[byte_pos] & ((1 << bits_in_first) - 1)
                second_bits = raw[byte_pos + 1] >> (8 + bits_in_first - 5)
                value = (first_bits << (5 - bits_in_first)) | second_bits
            buckets[bucket_idx] = value & 0x1F
            bit_pos += 5
            bucket_idx += 1
    else:
        raise ValueError(f"Unknown HLL format marker: {fmt}")
    return buckets

def encode_hll_b64(buckets):
    non_zero = sum(1 for v in buckets if v != 0)
    out = bytearray()
    out.extend(hll_magic)
    out.append(hll_version)
    out.append(hll_precision)
    if non_zero < sparse_threshold:
        out.append(1)
        entries = [(i, v) for i, v in enumerate(buckets) if v != 0]
        out.append(len(entries) & 0xFF)
        out.append((len(entries) >> 8) & 0xFF)
        for idx, val in entries:
            out.append(idx & 0xFF)
            out.append((idx >> 8) & 0xFF)
            out.append(val & 0x1F)
    else:
        out.append(0)
        bit_pos = 0
        current_byte = 0
        for v in buckets:
            v5 = v & 0x1F
            bits_remaining = 8 - (bit_pos % 8)
            if bits_remaining >= 5:
                current_byte |= (v5 << (bits_remaining - 5))
                bit_pos += 5
                if bit_pos % 8 == 0:
                    out.append(current_byte & 0xFF)
                    current_byte = 0
            else:
                current_byte |= (v5 >> (5 - bits_remaining))
                out.append(current_byte & 0xFF)
                current_byte = ((v5 & ((1 << (5 - bits_remaining)) - 1)) << (8 + bits_remaining - 5))
                bit_pos += 5
        if bit_pos % 8 != 0:
            out.append(current_byte & 0xFF)
    return base64.b64encode(bytes(out)).decode("ascii")

def hll_count_estimate(buckets):
    m = float(hll_num_buckets)
    alpha = 0.7213 / (1.0 + 1.079 / m)
    sum_pow = 0.0
    zero_buckets = 0
    for v in buckets:
        sum_pow += 2.0 ** (-v)
        if v == 0:
            zero_buckets += 1
    raw = alpha * m * m / sum_pow
    if raw <= 2.5 * m and zero_buckets > 0:
        return m * math.log(m / zero_buckets)
    if raw > (2 ** 32) / 30.0:
        return -1.0 * (2 ** 32) * math.log(1.0 - raw / (2 ** 32))
    return raw

def round_half_up(x):
    return int(math.floor(x + 0.5))

month_paths = sorted(glob.glob(str(merged_dir / "[0-9][0-9][0-9][0-9]-[0-9][0-9].json")))
if not month_paths:
    raise RuntimeError("No month JSON files found when rebuilding lifetime.json")

months = []
total_requests = 0
merged_buckets = [0] * hll_num_buckets

for month_path in month_paths:
    with open(month_path, "r", encoding="utf-8") as f:
        doc = json.load(f)
    summary = doc.get("summary")
    if not isinstance(summary, dict):
        raise RuntimeError(f"{month_path}: missing summary object")
    month = summary.get("month")
    if not isinstance(month, str):
        month = os.path.basename(month_path)[:7]
        summary["month"] = month
    req = int(summary.get("requests", 0))
    total_requests += req
    hll_b64 = summary.get("unique_visitors_hll")
    if not isinstance(hll_b64, str) or not hll_b64:
        raise RuntimeError(f"{month_path}: missing summary.unique_visitors_hll")
    buckets = decode_hll_b64(hll_b64)
    merged_buckets = [max(a, b) for a, b in zip(merged_buckets, buckets)]
    months.append(summary)

months.sort(key=lambda m: m.get("month", ""))
generated_at = dt.datetime.now(dt.timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")

log_files_processed = []
if generated_lifetime.exists():
    with open(generated_lifetime, "r", encoding="utf-8") as f:
        src_lifetime = json.load(f)
    if isinstance(src_lifetime.get("log_files_processed"), list):
        log_files_processed = src_lifetime["log_files_processed"]

total_hll_b64 = encode_hll_b64(merged_buckets)
total_unique = round_half_up(hll_count_estimate(merged_buckets))

lifetime = {
    "generated_at": generated_at,
    "log_files_processed": log_files_processed,
    "total_requests": total_requests,
    "total_unique_visitors_estimate": total_unique,
    "total_unique_visitors_hll": total_hll_b64,
    "months": months,
}

lifetime_path = merged_dir / "lifetime.json"
with open(lifetime_path, "w", encoding="utf-8") as f:
    json.dump(lifetime, f, indent=2, sort_keys=True)
    f.write("\n")

checksums = {}
for fp in sorted(merged_dir.glob("*.json")):
    if fp.name == "manifest.json":
        continue
    digest = hashlib.sha256(fp.read_bytes()).hexdigest()
    checksums[fp.name] = digest

manifest = {
    "generated_at": generated_at,
    "published_at": generated_at,
    "source_host": source_host,
    "month_count": len(month_paths),
    "months": [m.get("month") for m in months],
    "file_checksums_sha256": checksums,
}

manifest_path = merged_dir / "manifest.json"
with open(manifest_path, "w", encoding="utf-8") as f:
    json.dump(manifest, f, indent=2, sort_keys=True)
    f.write("\n")
PY
}

atomic_replace_from_source() {
    local source_dir="$1"
    local target_dir="$2"
    local stage_dir="${target_dir}.new"
    local backup_dir="${target_dir}.old"
    local had_old=0

    rm -rf "$stage_dir"
    mkdir -p "$stage_dir"
    cp -a "$source_dir"/. "$stage_dir"/

    rm -rf "$backup_dir"
    if [ -d "$target_dir" ]; then
        mv "$target_dir" "$backup_dir"
        had_old=1
    fi

    if mv "$stage_dir" "$target_dir"; then
        rm -rf "$backup_dir"
        return 0
    fi

    if [ "$had_old" -eq 1 ] && [ -d "$backup_dir" ]; then
        mv "$backup_dir" "$target_dir" || true
    fi
    fail "Atomic replace failed for target: $target_dir"
}

generated_months=("$GENERATED_DIR"/????-??.json)
if [ "${generated_months[0]}" = "$GENERATED_DIR/????-??.json" ]; then
    fail "No generated month files found in $GENERATED_DIR"
fi
[ -f "$GENERATED_DIR/lifetime.json" ] || fail "Missing generated lifetime.json in $GENERATED_DIR"
[ -f "$GENERATED_DIR/report.html" ] || fail "Missing generated report.html in $GENERATED_DIR"

log "Running privacy checks on generated outputs..."
privacy_check_dir "$GENERATED_DIR" "1" || fail "Privacy check failed on generated outputs"

WORK_DIR="$(mktemp -d /tmp/log-analyzer-publish.XXXXXX)"
cleanup() {
    rm -rf "$WORK_DIR"
}
trap cleanup EXIT

MERGED_DATA_DIR="$WORK_DIR/merged-data"
PUBLIC_STAGE_DIR="$WORK_DIR/public-analytics"
mkdir -p "$MERGED_DATA_DIR" "$PUBLIC_STAGE_DIR/data"

# Start with existing JSONs to preserve history across log rotation.
if [ -d "$DATA_DIR" ]; then
    cp "$DATA_DIR"/*.json "$MERGED_DATA_DIR"/ 2>/dev/null || true
fi

for gen_file in "${generated_months[@]}"; do
    file_name="$(basename "$gen_file")"
    month="${file_name%.json}"
    merged_path="$MERGED_DATA_DIR/$file_name"

    if [ -f "$merged_path" ] && ! month_is_mutable "$month"; then
        if ! json_equal_ignoring_generated_at "$gen_file" "$merged_path"; then
            log "Immutable month differs from regenerated output (expected with rotation); preserving existing $file_name"
        fi
        continue
    fi

    cp "$gen_file" "$merged_path"
done

# report.html is always regenerated from the latest run.
cp "$GENERATED_DIR/report.html" "$MERGED_DATA_DIR/report.html"

log "Rebuilding lifetime.json and manifest.json from persisted month files..."
build_lifetime_and_manifest "$MERGED_DATA_DIR" "$GENERATED_DIR/lifetime.json" "$(hostname -s)"

log "Running privacy checks on merged outputs..."
privacy_check_dir "$MERGED_DATA_DIR" "1" || fail "Privacy check failed on merged outputs"

log "Publishing canonical private analytics store..."
atomic_replace_from_source "$MERGED_DATA_DIR" "$DATA_DIR"

log "Mirroring JSON backup to Bunny..."
bunny_sync "$DATA_DIR"

log "Publishing Apache analytics report and JSON..."
cp "$DATA_DIR"/report.html "$PUBLIC_STAGE_DIR/report.html"
cp "$DATA_DIR"/*.json "$PUBLIC_STAGE_DIR/data"/
atomic_replace_from_source "$PUBLIC_STAGE_DIR" "$PUBLIC_ANALYTICS_DIR"

log "Publish completed: data_dir=$DATA_DIR public_dir=$PUBLIC_ANALYTICS_DIR"
