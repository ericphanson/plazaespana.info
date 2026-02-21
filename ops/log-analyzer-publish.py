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
import datetime as dt
import glob
import hashlib
import ipaddress
import json
import os
from pathlib import Path
import re
import shutil
import socket
import subprocess
import sys
import tempfile
from typing import Any


MONTH_JSON_RE = re.compile(r"^\d{4}-\d{2}\.json$")
IPV4_RE = re.compile(r"(?<![\d.])(?:\d{1,3}\.){3}\d{1,3}(?![\d.])")
IPV6_CANDIDATE_RE = re.compile(r"(?i)(?<![0-9a-f:])[0-9a-f:]{2,39}(?![0-9a-f:])")


def env_flag(name: str, default: bool) -> bool:
    raw = os.environ.get(name)
    if raw is None:
        return default
    return raw.strip().lower() in {"1", "true", "yes", "on"}


def env_int(name: str, default: int, minimum: int = 0) -> int:
    raw = os.environ.get(name)
    if raw is None:
        value = default
    else:
        try:
            value = int(raw.strip())
        except ValueError as e:
            raise RuntimeError(f"{name} must be an integer (got {raw!r})") from e
    if value < minimum:
        raise RuntimeError(f"{name} must be >= {minimum} (got {value})")
    return value


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


def rebuild_lifetime_with_log_analyzer(
    log_analyzer_bin: Path,
    merged_dir: Path,
    generated_lifetime: Path,
    log: Logger,
) -> None:
    if not log_analyzer_bin.exists():
        fail(log, f"log-analyzer binary not found: {log_analyzer_bin}")

    cmd = [
        str(log_analyzer_bin),
        "--mode",
        "rebuild",
        "--json-dir",
        str(merged_dir),
    ]
    if generated_lifetime.exists():
        cmd.extend(["--lifetime-source", str(generated_lifetime)])

    proc = run_checked(cmd, log, "Lifetime rebuild")
    if proc.stdout.strip():
        log.log(proc.stdout.strip())
    if proc.stderr.strip():
        log.log(proc.stderr.strip())


def rebuild_lifetime_and_manifest(
    merged_dir: Path,
    generated_lifetime: Path,
    source_host: str,
    log_analyzer_bin: Path,
    log: Logger,
) -> None:
    rebuild_lifetime_with_log_analyzer(log_analyzer_bin, merged_dir, generated_lifetime, log)

    month_paths = sorted(merged_dir.glob("[0-9][0-9][0-9][0-9]-[0-9][0-9].json"))
    if not month_paths:
        raise RuntimeError("No month JSON files found when rebuilding lifetime.json")

    lifetime_path = merged_dir / "lifetime.json"
    if not lifetime_path.exists():
        raise RuntimeError(f"Expected lifetime.json from rebuild step: {lifetime_path}")
    with lifetime_path.open("r", encoding="utf-8") as f:
        lifetime = json.load(f)
    generated_at = lifetime.get("generated_at")
    if not isinstance(generated_at, str) or not generated_at:
        generated_at = dt.datetime.now(dt.timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")

    month_names: list[str] = []
    months = lifetime.get("months")
    if isinstance(months, list):
        for month_entry in months:
            if isinstance(month_entry, dict):
                month = month_entry.get("month")
                if isinstance(month, str):
                    month_names.append(month)
    if not month_names:
        month_names = [month_path.name[:7] for month_path in month_paths]

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
        "months": month_names,
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

    try:
        curl_connect_timeout = env_int("BUNNY_CURL_CONNECT_TIMEOUT_SEC", 10, minimum=1)
        curl_max_time = env_int("BUNNY_CURL_MAX_TIME_SEC", 120, minimum=1)
        curl_retries = env_int("BUNNY_CURL_RETRIES", 3, minimum=0)
        curl_retry_delay = env_int("BUNNY_CURL_RETRY_DELAY_SEC", 2, minimum=0)
        curl_retry_max_time = env_int("BUNNY_CURL_RETRY_MAX_TIME_SEC", 180, minimum=1)
    except RuntimeError as e:
        fail(log, str(e))

    bunny_endpoint = bunny_endpoint.removeprefix("http://").removeprefix("https://").rstrip("/")
    bunny_base_path = bunny_base_path.lstrip("/").rstrip("/")

    curl_common_flags = [
        "--connect-timeout",
        str(curl_connect_timeout),
        "--max-time",
        str(curl_max_time),
        "--retry",
        str(curl_retries),
        "--retry-delay",
        str(curl_retry_delay),
        "--retry-max-time",
        str(curl_retry_max_time),
    ]

    def bunny_url(rel: str) -> str:
        return f"https://{bunny_endpoint}/{bunny_zone}/{rel}"

    def bunny_download_if_exists(rel: str, out: Path) -> bool:
        cmd = [
            "curl",
            "-sS",
            *curl_common_flags,
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
            *curl_common_flags,
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
            rebuild_lifetime_and_manifest(
                merged_data_dir,
                generated_lifetime,
                socket.gethostname().split(".")[0],
                log_analyzer_bin,
                log,
            )

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
