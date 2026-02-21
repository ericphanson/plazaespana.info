#!/usr/bin/env python3
"""Publish log-analyzer outputs.

Pipeline:
1) validate privacy constraints
2) merge with persisted month history
3) rebuild lifetime.json + manifest.json from persisted month files
4) generate report.html from persisted JSON files (phase 2)
5) publish canonical private data
6) mirror JSON backup to Bunny with immutable-month safeguards
7) publish Apache analytics endpoints
"""

from __future__ import annotations

import argparse
import base64
import datetime as dt
import glob
import hashlib
import ipaddress
import json
import math
import os
from pathlib import Path
import re
import shutil
import socket
import subprocess
import sys
import tempfile
from typing import Any


HLL_PRECISION = 14
HLL_NUM_BUCKETS = 1 << HLL_PRECISION
HLL_VERSION = 1
HLL_MAGIC = b"HLL"
SPARSE_THRESHOLD = 3412
MONTH_JSON_RE = re.compile(r"^\d{4}-\d{2}\.json$")
IPV4_RE = re.compile(r"(?<![\d.])(?:\d{1,3}\.){3}\d{1,3}(?![\d.])")
IPV6_CANDIDATE_RE = re.compile(r"(?i)(?<![0-9a-f:])[0-9a-f:]{2,39}(?![0-9a-f:])")


def env_flag(name: str, default: bool) -> bool:
    raw = os.environ.get(name)
    if raw is None:
        return default
    return raw.strip().lower() in {"1", "true", "yes", "on"}


class Logger:
    def __init__(self, log_file: Path) -> None:
        self.log_file = log_file
        self.log_file.parent.mkdir(parents=True, exist_ok=True)

    def log(self, msg: str) -> None:
        if sys.stdout.isatty():
            print(msg)
        with self.log_file.open("a", encoding="utf-8") as f:
            f.write(msg + "\n")


def fail(log: Logger, msg: str) -> None:
    log.log(f"ERROR: {msg}")
    raise RuntimeError(msg)


def month_is_mutable(month: str, grace_days: int) -> bool:
    today = dt.datetime.now(dt.timezone.utc).date()
    current = today.strftime("%Y-%m")
    if month == current:
        return True
    first_this_month = today.replace(day=1)
    prev = (first_this_month - dt.timedelta(days=1)).strftime("%Y-%m")
    return month == prev and today.day <= grace_days


def normalize_without_generated_at(node: Any) -> Any:
    if isinstance(node, dict):
        return {k: normalize_without_generated_at(v) for k, v in node.items() if k != "generated_at"}
    if isinstance(node, list):
        return [normalize_without_generated_at(v) for v in node]
    return node


def json_equal_ignoring_generated_at(lhs: Path, rhs: Path) -> bool:
    with lhs.open("r", encoding="utf-8") as f:
        l = normalize_without_generated_at(json.load(f))
    with rhs.open("r", encoding="utf-8") as f:
        r = normalize_without_generated_at(json.load(f))
    return l == r


def find_ip_match(text: str) -> str | None:
    for m in IPV4_RE.finditer(text):
        cand = m.group(0)
        try:
            ip = ipaddress.ip_address(cand)
        except ValueError:
            continue
        if isinstance(ip, ipaddress.IPv4Address):
            return f"IPv4:{cand}"
    for m in IPV6_CANDIDATE_RE.finditer(text):
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


def check_paths_no_query(node: Any, path: Path) -> None:
    if isinstance(node, dict):
        for k, v in node.items():
            if k == "path" and isinstance(v, str) and "?" in v:
                raise RuntimeError(f"{path}: path contains query string: {v}")
            check_paths_no_query(v, path)
    elif isinstance(node, list):
        for item in node:
            check_paths_no_query(item, path)


def privacy_check_dir(root: Path, include_report: bool) -> None:
    json_files = sorted(root.glob("*.json"))
    if not json_files:
        raise RuntimeError(f"No JSON files found under {root}")

    files_to_scan = list(json_files)
    report = root / "report.html"
    if include_report and report.exists():
        files_to_scan.append(report)

    for fp in files_to_scan:
        text = fp.read_text(encoding="utf-8", errors="replace")
        ip_match = find_ip_match(text)
        if ip_match:
            raise RuntimeError(f"{fp}: potential IP address found: {ip_match}")

    for fp in json_files:
        try:
            doc = json.loads(fp.read_text(encoding="utf-8"))
        except json.JSONDecodeError as e:
            raise RuntimeError(f"{fp}: invalid JSON: {e}") from e
        check_paths_no_query(doc, fp)


def decode_hll_b64(s: str) -> list[int]:
    raw = base64.b64decode(s.encode("ascii"))
    if len(raw) < 6:
        raise RuntimeError("HLL buffer too small")
    if raw[:3] != HLL_MAGIC:
        raise RuntimeError("Invalid HLL magic")
    if raw[3] != HLL_VERSION:
        raise RuntimeError("Unsupported HLL version")
    if raw[4] != HLL_PRECISION:
        raise RuntimeError(f"Unexpected HLL precision {raw[4]}")

    fmt = raw[5]
    buckets = [0] * HLL_NUM_BUCKETS
    if fmt == 1:
        if len(raw) < 8:
            raise RuntimeError("Sparse HLL buffer too small")
        count = raw[6] | (raw[7] << 8)
        pos = 8
        for _ in range(count):
            if pos + 2 >= len(raw):
                raise RuntimeError("Sparse HLL truncated")
            idx = raw[pos] | (raw[pos + 1] << 8)
            val = raw[pos + 2] & 0x1F
            if idx >= HLL_NUM_BUCKETS:
                raise RuntimeError("Sparse HLL index out of range")
            buckets[idx] = val
            pos += 3
    elif fmt == 0:
        bit_pos = 0
        bucket_idx = 0
        offset = 6
        while bucket_idx < HLL_NUM_BUCKETS:
            byte_pos = offset + (bit_pos // 8)
            bit_offset = bit_pos % 8
            bits_in_first = 8 - bit_offset
            if byte_pos >= len(raw):
                raise RuntimeError("Dense HLL truncated")
            if bits_in_first >= 5:
                value = (raw[byte_pos] >> (bits_in_first - 5)) & 0x1F
            else:
                if byte_pos + 1 >= len(raw):
                    raise RuntimeError("Dense HLL truncated")
                first_bits = raw[byte_pos] & ((1 << bits_in_first) - 1)
                second_bits = raw[byte_pos + 1] >> (8 + bits_in_first - 5)
                value = (first_bits << (5 - bits_in_first)) | second_bits
            buckets[bucket_idx] = value & 0x1F
            bit_pos += 5
            bucket_idx += 1
    else:
        raise RuntimeError(f"Unknown HLL format marker: {fmt}")
    return buckets


def encode_hll_b64(buckets: list[int]) -> str:
    non_zero = sum(1 for v in buckets if v != 0)
    out = bytearray()
    out.extend(HLL_MAGIC)
    out.append(HLL_VERSION)
    out.append(HLL_PRECISION)
    if non_zero < SPARSE_THRESHOLD:
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
                current_byte |= v5 << (bits_remaining - 5)
                bit_pos += 5
                if bit_pos % 8 == 0:
                    out.append(current_byte & 0xFF)
                    current_byte = 0
            else:
                current_byte |= v5 >> (5 - bits_remaining)
                out.append(current_byte & 0xFF)
                current_byte = (v5 & ((1 << (5 - bits_remaining)) - 1)) << (8 + bits_remaining - 5)
                bit_pos += 5
        if bit_pos % 8 != 0:
            out.append(current_byte & 0xFF)
    return base64.b64encode(bytes(out)).decode("ascii")


def hll_count_estimate(buckets: list[int]) -> float:
    m = float(HLL_NUM_BUCKETS)
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
    if raw > (2**32) / 30.0:
        return -1.0 * (2**32) * math.log(1.0 - raw / (2**32))
    return raw


def round_half_up(x: float) -> int:
    return int(math.floor(x + 0.5))


def rebuild_lifetime_and_manifest(merged_dir: Path, generated_lifetime: Path, source_host: str) -> None:
    month_paths = sorted(merged_dir.glob("[0-9][0-9][0-9][0-9]-[0-9][0-9].json"))
    if not month_paths:
        raise RuntimeError("No month JSON files found when rebuilding lifetime.json")

    months: list[dict[str, Any]] = []
    total_requests = 0
    merged_buckets = [0] * HLL_NUM_BUCKETS

    for month_path in month_paths:
        with month_path.open("r", encoding="utf-8") as f:
            doc = json.load(f)
        summary = doc.get("summary")
        if not isinstance(summary, dict):
            raise RuntimeError(f"{month_path}: missing summary object")
        month = summary.get("month")
        if not isinstance(month, str):
            month = month_path.name[:7]
            summary["month"] = month

        req = int(summary.get("requests", 0))
        total_requests += req
        hll_b64 = summary.get("unique_visitors_hll")
        if not isinstance(hll_b64, str) or not hll_b64:
            raise RuntimeError(f"{month_path}: missing summary.unique_visitors_hll")
        buckets = decode_hll_b64(hll_b64)
        merged_buckets = [max(a, b) for a, b in zip(merged_buckets, buckets)]
        months.append(summary)

    months.sort(key=lambda m: str(m.get("month", "")))
    generated_at = dt.datetime.now(dt.timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")

    log_files_processed: list[str] = []
    if generated_lifetime.exists():
        with generated_lifetime.open("r", encoding="utf-8") as f:
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
    with (merged_dir / "lifetime.json").open("w", encoding="utf-8") as f:
        json.dump(lifetime, f, indent=2, sort_keys=True)
        f.write("\n")

    checksums: dict[str, str] = {}
    for fp in sorted(merged_dir.glob("*.json")):
        if fp.name == "manifest.json":
            continue
        checksums[fp.name] = hashlib.sha256(fp.read_bytes()).hexdigest()

    manifest = {
        "generated_at": generated_at,
        "published_at": generated_at,
        "source_host": source_host,
        "month_count": len(month_paths),
        "months": [m.get("month") for m in months],
        "file_checksums_sha256": checksums,
    }
    with (merged_dir / "manifest.json").open("w", encoding="utf-8") as f:
        json.dump(manifest, f, indent=2, sort_keys=True)
        f.write("\n")


def atomic_replace_from_source(source_dir: Path, target_dir: Path) -> None:
    stage_dir = Path(str(target_dir) + ".new")
    backup_dir = Path(str(target_dir) + ".old")

    if stage_dir.exists():
        shutil.rmtree(stage_dir)
    shutil.copytree(source_dir, stage_dir)

    if backup_dir.exists():
        shutil.rmtree(backup_dir)
    had_old = False
    if target_dir.exists():
        target_dir.rename(backup_dir)
        had_old = True

    try:
        stage_dir.rename(target_dir)
        if backup_dir.exists():
            shutil.rmtree(backup_dir)
    except Exception:
        if had_old and backup_dir.exists():
            backup_dir.rename(target_dir)
        raise


def run_checked(cmd: list[str], log: Logger, context: str) -> subprocess.CompletedProcess[str]:
    proc = subprocess.run(cmd, text=True, capture_output=True)
    if proc.returncode != 0:
        fail(log, f"{context} failed (exit {proc.returncode}): {' '.join(cmd)}\n{proc.stderr.strip()}")
    return proc


def generate_report_from_json(log_analyzer_bin: Path, json_dir: Path, report_path: Path, log: Logger) -> None:
    if not log_analyzer_bin.exists():
        fail(log, f"log-analyzer binary not found: {log_analyzer_bin}")
    cmd = [
        str(log_analyzer_bin),
        "--mode",
        "report",
        "--json-dir",
        str(json_dir),
        "--report-path",
        str(report_path),
    ]
    proc = run_checked(cmd, log, "Report generation")
    if proc.stdout.strip():
        log.log(proc.stdout.strip())
    if proc.stderr.strip():
        log.log(proc.stderr.strip())


def bunny_sync(
    merged_dir: Path,
    grace_days: int,
    backup_required: bool,
    backup_report: bool,
    log: Logger,
) -> None:
    key_file = Path("/home/private/bunny-storage-key.txt")
    zone_file = Path("/home/private/bunny-storage-zone.txt")
    endpoint_file = Path("/home/private/bunny-storage-endpoint.txt")
    base_file = Path("/home/private/bunny-base-path.txt")

    if not (key_file.exists() and zone_file.exists() and endpoint_file.exists()):
        if backup_required:
            fail(log, "Bunny backup required, but one or more Bunny config files are missing in /home/private")
        log.log("Bunny backup not configured; skipping backup mirror.")
        return

    bunny_key = key_file.read_text(encoding="utf-8").strip()
    bunny_zone = zone_file.read_text(encoding="utf-8").strip()
    bunny_endpoint = endpoint_file.read_text(encoding="utf-8").strip()
    bunny_base_path = base_file.read_text(encoding="utf-8").strip() if base_file.exists() else "analytics-backup/current"

    if not bunny_key or not bunny_zone or not bunny_endpoint or not bunny_base_path:
        fail(log, "Bunny configuration files must be non-empty")

    bunny_endpoint = bunny_endpoint.removeprefix("http://").removeprefix("https://").rstrip("/")
    bunny_base_path = bunny_base_path.lstrip("/").rstrip("/")

    def bunny_url(rel: str) -> str:
        return f"https://{bunny_endpoint}/{bunny_zone}/{rel}"

    def bunny_download_if_exists(rel: str, out: Path) -> bool:
        cmd = [
            "curl",
            "-sS",
            "-o",
            str(out),
            "-w",
            "%{http_code}",
            "-H",
            f"AccessKey: {bunny_key}",
            bunny_url(rel),
        ]
        proc = run_checked(cmd, log, "Bunny download")
        code = proc.stdout.strip()
        if code == "200":
            return True
        if out.exists():
            out.unlink()
        if code == "404":
            return False
        fail(log, f"Bunny download failed for {rel} (HTTP {code})")
        return False

    def bunny_upload(local_file: Path, rel: str) -> None:
        cmd = [
            "curl",
            "-sS",
            "-o",
            "/dev/null",
            "-w",
            "%{http_code}",
            "-X",
            "PUT",
            "-H",
            f"AccessKey: {bunny_key}",
            "--data-binary",
            f"@{local_file}",
            bunny_url(rel),
        ]
        proc = run_checked(cmd, log, "Bunny upload")
        code = proc.stdout.strip()
        if code not in {"200", "201"}:
            fail(log, f"Bunny upload failed for {rel} (HTTP {code})")

    mirrored = 0
    immutable_verified = 0
    mirrored_verified = 0
    with tempfile.TemporaryDirectory(prefix="log-analyzer-bunny-") as td:
        tmp_dir = Path(td)
        for local_file in sorted(merged_dir.glob("*.json")):
            name = local_file.name
            rel = f"{bunny_base_path}/{name}"
            mutable = True
            if MONTH_JSON_RE.fullmatch(name):
                month = name[:7]
                mutable = month_is_mutable(month, grace_days)

            remote_copy = tmp_dir / name
            if not mutable:
                if bunny_download_if_exists(rel, remote_copy):
                    if not json_equal_ignoring_generated_at(local_file, remote_copy):
                        fail(log, f"Immutable month differs from existing Bunny backup: {name}")
                    immutable_verified += 1
                    continue
            bunny_upload(local_file, rel)
            mirrored += 1

        report_file = merged_dir / "report.html"
        if backup_report and report_file.exists():
            bunny_upload(report_file, f"{bunny_base_path}/report.html")
            mirrored += 1

        expected_files: list[tuple[Path, str, bool, bool]] = []
        for local_file in sorted(merged_dir.glob("*.json")):
            name = local_file.name
            rel = f"{bunny_base_path}/{name}"
            mutable = True
            if MONTH_JSON_RE.fullmatch(name):
                mutable = month_is_mutable(name[:7], grace_days)
            expected_files.append((local_file, rel, mutable, True))
        if backup_report and report_file.exists():
            expected_files.append((report_file, f"{bunny_base_path}/report.html", True, False))

        for idx, (local_file, rel, mutable, is_json) in enumerate(expected_files):
            remote_copy = tmp_dir / f"verify-{idx}-{local_file.name}"
            if not bunny_download_if_exists(rel, remote_copy):
                fail(log, f"Bunny verify failed: missing mirrored file {rel}")
            if is_json and MONTH_JSON_RE.fullmatch(local_file.name) and not mutable:
                if not json_equal_ignoring_generated_at(local_file, remote_copy):
                    fail(log, f"Bunny verify failed for immutable month {local_file.name}")
            else:
                local_sha = hashlib.sha256(local_file.read_bytes()).hexdigest()
                remote_sha = hashlib.sha256(remote_copy.read_bytes()).hexdigest()
                if local_sha != remote_sha:
                    fail(log, f"Bunny verify checksum mismatch for {local_file.name}")
            mirrored_verified += 1

    log.log(
        f"Bunny backup complete: uploaded/updated={mirrored} "
        f"immutable-verified={immutable_verified} mirrored-verified={mirrored_verified} "
        f"base={bunny_base_path}"
    )


def merge_generated_months(
    generated_dir: Path,
    merged_dir: Path,
    grace_days: int,
    log: Logger,
) -> None:
    for gen_file in sorted(generated_dir.glob("[0-9][0-9][0-9][0-9]-[0-9][0-9].json")):
        name = gen_file.name
        month = name[:7]
        merged_path = merged_dir / name

        if merged_path.exists() and not month_is_mutable(month, grace_days):
            if not json_equal_ignoring_generated_at(gen_file, merged_path):
                log.log(
                    f"Immutable month differs from regenerated output "
                    f"(expected with rotation); preserving existing {name}"
                )
            continue
        shutil.copy2(gen_file, merged_path)


def main() -> int:
    parser = argparse.ArgumentParser(
        description="Publish log-analyzer outputs with privacy checks and Bunny backup.",
    )
    parser.add_argument("generated_dir")
    parser.add_argument("data_dir", nargs="?", default="/home/private/log-analyzer-data")
    parser.add_argument("public_analytics_dir", nargs="?", default="/home/public/analytics")
    args = parser.parse_args()

    generated_dir = Path(args.generated_dir)
    data_dir = Path(args.data_dir)
    public_dir = Path(args.public_analytics_dir)
    log_file = Path(os.environ.get("LOG_FILE", "/home/logs/log-analyzer.log"))
    log = Logger(log_file)

    grace_days = int(os.environ.get("MONTH_CLOSE_GRACE_DAYS", "7"))
    bunny_backup_required = env_flag("BUNNY_BACKUP_REQUIRED", True)
    bunny_backup_report = env_flag("BUNNY_BACKUP_REPORT", False)
    log_analyzer_bin = Path(os.environ.get("LOG_ANALYZER_BIN", "/home/private/bin/log-analyzer"))

    try:
        if not generated_dir.is_dir():
            fail(log, f"generated output directory not found: {generated_dir}")

        generated_months = sorted(generated_dir.glob("[0-9][0-9][0-9][0-9]-[0-9][0-9].json"))
        if not generated_months:
            fail(log, f"No generated month files found in {generated_dir}")
        generated_lifetime = generated_dir / "lifetime.json"
        if not generated_lifetime.exists():
            fail(log, f"Missing generated lifetime.json in {generated_dir}")

        log.log("Running privacy checks on generated outputs...")
        privacy_check_dir(generated_dir, include_report=False)

        with tempfile.TemporaryDirectory(prefix="log-analyzer-publish-") as td:
            work_dir = Path(td)
            merged_data_dir = work_dir / "merged-data"
            public_stage_dir = work_dir / "public-analytics"
            merged_data_dir.mkdir(parents=True, exist_ok=True)
            (public_stage_dir / "data").mkdir(parents=True, exist_ok=True)

            if data_dir.exists():
                # Persist only canonical month shards from prior runs.
                # lifetime.json/manifest.json are rebuilt each run.
                for fp in data_dir.glob("[0-9][0-9][0-9][0-9]-[0-9][0-9].json"):
                    shutil.copy2(fp, merged_data_dir / fp.name)

            merge_generated_months(generated_dir, merged_data_dir, grace_days, log)

            log.log("Rebuilding lifetime.json and manifest.json from persisted month files...")
            rebuild_lifetime_and_manifest(merged_data_dir, generated_lifetime, socket.gethostname().split(".")[0])

            log.log("Generating report.html from persisted JSON...")
            generate_report_from_json(log_analyzer_bin, merged_data_dir, merged_data_dir / "report.html", log)

            log.log("Running privacy checks on merged outputs...")
            privacy_check_dir(merged_data_dir, include_report=True)

            log.log("Publishing canonical private analytics store...")
            atomic_replace_from_source(merged_data_dir, data_dir)

            log.log("Mirroring JSON backup to Bunny...")
            bunny_sync(
                data_dir,
                grace_days=grace_days,
                backup_required=bunny_backup_required,
                backup_report=bunny_backup_report,
                log=log,
            )

            log.log("Publishing Apache analytics report and JSON...")
            shutil.copy2(data_dir / "report.html", public_stage_dir / "report.html")
            for fp in data_dir.glob("*.json"):
                shutil.copy2(fp, public_stage_dir / "data" / fp.name)
            atomic_replace_from_source(public_stage_dir, public_dir)

        log.log(f"Publish completed: data_dir={data_dir} public_dir={public_dir}")
        return 0
    except Exception as e:  # noqa: BLE001
        log.log(f"ERROR: {e}")
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
