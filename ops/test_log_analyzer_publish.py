#!/usr/bin/env python3
"""Tests for log-analyzer-publish.py pipeline behavior."""

from __future__ import annotations

from contextlib import contextmanager
import importlib.util
import json
from pathlib import Path
import shutil
import subprocess
import tempfile
import textwrap
import unittest
from unittest.mock import patch


OPS_DIR = Path(__file__).resolve().parent
REPO_ROOT = OPS_DIR.parent
PUBLISH_PATH = OPS_DIR / "log-analyzer-publish.py"
EMPTY_HLL_B64 = "SExMAQ4BAAA="


def load_publish_module():
    spec = importlib.util.spec_from_file_location("log_analyzer_publish", PUBLISH_PATH)
    if spec is None or spec.loader is None:
        raise RuntimeError(f"Unable to load module spec from {PUBLISH_PATH}")
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


publish = load_publish_module()


@contextmanager
def suppress_logger_stdout():
    """Prevent Logger.log from printing to terminal during tests."""
    with patch("sys.stdout.isatty", return_value=False):
        yield


class RebuildInteropTests(unittest.TestCase):
    def _require_janet(self) -> None:
        if shutil.which("janet") is None:
            self.skipTest("janet not found on PATH")

    def _build_janet_hll(self, body: str) -> str:
        self._require_janet()
        program = textwrap.dedent(
            f"""
            (import ./log-analyzer/src/hll)
            (def h (hll/new))
            {body}
            (print (hll/to-base64 h))
            """
        ).strip()

        proc = subprocess.run(
            ["janet", "-e", program],
            cwd=REPO_ROOT,
            text=True,
            capture_output=True,
            check=True,
        )
        lines = [line.strip() for line in proc.stdout.splitlines() if line.strip()]
        if len(lines) < 1:
            self.fail(f"Unexpected Janet output: {proc.stdout!r}")
        return lines[0]

    def _janet_merged_stats(self, hll_a: str, hll_b: str) -> tuple[int, str]:
        self._require_janet()
        program = textwrap.dedent(
            f"""
            (import ./log-analyzer/src/hll)
            (def merged (hll/merge (hll/from-base64 \"{hll_a}\") (hll/from-base64 \"{hll_b}\")))
            (print (math/floor (+ (hll/count-estimate merged) 0.5)))
            (print (hll/to-base64 merged))
            """
        ).strip()
        proc = subprocess.run(
            ["janet", "-e", program],
            cwd=REPO_ROOT,
            text=True,
            capture_output=True,
            check=True,
        )
        lines = [line.strip() for line in proc.stdout.splitlines() if line.strip()]
        if len(lines) < 2:
            self.fail(f"Unexpected Janet output: {proc.stdout!r}")
        return int(lines[0]), lines[1]

    def _janet_main_wrapper(self, directory: Path) -> Path:
        self._require_janet()
        wrapper_path = directory / "log-analyzer-wrapper.sh"
        wrapper_path.write_text(
            textwrap.dedent(
                f"""\
                #!/bin/sh
                exec janet "{REPO_ROOT / 'log-analyzer' / 'src' / 'main.janet'}" "$@"
                """
            ),
            encoding="utf-8",
        )
        wrapper_path.chmod(0o755)
        return wrapper_path

    def _write_month_file(self, path: Path, month: str, requests: int, hll_b64: str) -> None:
        doc = {
            "generated_at": "2026-02-21T00:00:00Z",
            "month": month,
            "summary": {
                "month": month,
                "requests": requests,
                "unique_visitors_exact": 0,
                "unique_visitors_estimate": 0,
                "unique_visitors_hll": hll_b64,
            },
            "hourly": [],
        }
        path.write_text(json.dumps(doc, indent=2, sort_keys=True) + "\n", encoding="utf-8")

    def test_rebuild_mode_via_janet_main_script(self) -> None:
        with tempfile.TemporaryDirectory() as td:
            base = Path(td)
            merged_dir = base / "merged"
            merged_dir.mkdir()

            hll_a = self._build_janet_hll(
                """
                (for i 0 2200
                  (hll/add h (string "returning-" i)))
                """
            )
            hll_b = self._build_janet_hll(
                """
                (for i 1200 3600
                  (hll/add h (string "returning-" i)))
                """
            )
            expected_total, expected_hll = self._janet_merged_stats(hll_a, hll_b)

            self._write_month_file(merged_dir / "2026-01.json", "2026-01", requests=10, hll_b64=hll_a)
            self._write_month_file(merged_dir / "2026-02.json", "2026-02", requests=20, hll_b64=hll_b)

            generated_lifetime = base / "generated-lifetime.json"
            generated_lifetime.write_text(
                json.dumps({"log_files_processed": ["access_log", "access_log.1"]}),
                encoding="utf-8",
            )
            wrapper_bin = self._janet_main_wrapper(base)
            log = publish.Logger(base / "publish.log")

            with suppress_logger_stdout():
                publish.rebuild_lifetime_and_manifest(merged_dir, generated_lifetime, "nfs", wrapper_bin, log)

            lifetime = json.loads((merged_dir / "lifetime.json").read_text(encoding="utf-8"))
            manifest = json.loads((merged_dir / "manifest.json").read_text(encoding="utf-8"))

            self.assertEqual(lifetime["total_requests"], 30)
            self.assertEqual(lifetime["total_unique_visitors_estimate"], expected_total)
            self.assertEqual(lifetime["total_unique_visitors_hll"], expected_hll)
            self.assertEqual(lifetime["log_files_processed"], ["access_log", "access_log.1"])
            self.assertEqual(manifest["months"], ["2026-01", "2026-02"])
            self.assertEqual(manifest["month_count"], 2)
            self.assertIn("lifetime.json", manifest["file_checksums_sha256"])


class PublishPipelineTests(unittest.TestCase):
    def _write_fake_rebuild_bin(self, directory: Path) -> Path:
        fake_bin = directory / "fake-log-analyzer-rebuild.py"
        fake_bin.write_text(
            textwrap.dedent(
                f"""\
                #!/usr/bin/env python3
                import argparse
                import json
                import sys
                from pathlib import Path

                parser = argparse.ArgumentParser()
                parser.add_argument("--mode", required=True)
                parser.add_argument("--json-dir", required=True)
                parser.add_argument("--lifetime-source")
                args = parser.parse_args()

                if args.mode != "rebuild":
                    print("expected --mode rebuild", file=sys.stderr)
                    raise SystemExit(2)

                json_dir = Path(args.json_dir)
                month_paths = sorted(json_dir.glob("[0-9][0-9][0-9][0-9]-[0-9][0-9].json"))
                if not month_paths:
                    print("no month files", file=sys.stderr)
                    raise SystemExit(3)

                months = []
                total_requests = 0
                for month_path in month_paths:
                    doc = json.loads(month_path.read_text(encoding="utf-8"))
                    summary = doc.get("summary")
                    if not isinstance(summary, dict):
                        print(f"missing summary object in {{month_path}}", file=sys.stderr)
                        raise SystemExit(4)
                    if "month" not in summary:
                        summary["month"] = month_path.name[:7]
                    if "unique_visitors_hll" not in summary:
                        print(f"missing summary.unique_visitors_hll in {{month_path}}", file=sys.stderr)
                        raise SystemExit(5)
                    total_requests += int(summary.get("requests", 0))
                    months.append(summary)

                months.sort(key=lambda m: str(m.get("month", "")))
                log_files = []
                if args.lifetime_source:
                    source_path = Path(args.lifetime_source)
                    if source_path.exists():
                        source = json.loads(source_path.read_text(encoding="utf-8"))
                        if isinstance(source.get("log_files_processed"), list):
                            log_files = source["log_files_processed"]

                lifetime = {{
                    "generated_at": "2026-02-21T12:00:00Z",
                    "log_files_processed": log_files,
                    "total_requests": total_requests,
                    "total_unique_visitors_estimate": 0,
                    "total_unique_visitors_hll": "{EMPTY_HLL_B64}",
                    "months": months,
                }}
                (json_dir / "lifetime.json").write_text(
                    json.dumps(lifetime, indent=2, sort_keys=True) + "\\n",
                    encoding="utf-8",
                )
                """
            ),
            encoding="utf-8",
        )
        fake_bin.chmod(0o755)
        return fake_bin

    def _write_month_file(self, path: Path, month: str, requests: int, hll_b64: str | None = None) -> None:
        if hll_b64 is None:
            hll_b64 = EMPTY_HLL_B64
        doc = {
            "generated_at": "2026-02-21T00:00:00Z",
            "month": month,
            "summary": {
                "month": month,
                "requests": requests,
                "unique_visitors_exact": 0,
                "unique_visitors_estimate": 0,
                "unique_visitors_hll": hll_b64,
            },
            "hourly": [],
        }
        path.write_text(json.dumps(doc, indent=2, sort_keys=True) + "\n", encoding="utf-8")

    def test_rebuild_lifetime_fails_when_log_analyzer_bin_missing(self) -> None:
        with tempfile.TemporaryDirectory() as td:
            base = Path(td)
            merged_dir = base / "merged"
            merged_dir.mkdir()
            self._write_month_file(merged_dir / "2026-01.json", "2026-01", requests=1)
            generated_lifetime = base / "generated-lifetime.json"
            generated_lifetime.write_text("{}", encoding="utf-8")
            log = publish.Logger(base / "publish.log")
            with suppress_logger_stdout():
                with self.assertRaisesRegex(RuntimeError, "log-analyzer binary not found"):
                    publish.rebuild_lifetime_and_manifest(
                        merged_dir,
                        generated_lifetime,
                        "nfs",
                        base / "missing-log-analyzer",
                        log,
                    )

    def test_privacy_check_rejects_ip_address(self) -> None:
        with tempfile.TemporaryDirectory() as td:
            root = Path(td)
            payload = {
                "summary": {"month": "2026-02", "requests": 1},
                "hourly": [{"note": "saw 203.0.113.42 in a debug field"}],
            }
            (root / "2026-02.json").write_text(json.dumps(payload), encoding="utf-8")
            with self.assertRaisesRegex(RuntimeError, "potential IP address found"):
                publish.privacy_check_dir(root, include_report=False)

    def test_privacy_check_rejects_query_string_paths(self) -> None:
        with tempfile.TemporaryDirectory() as td:
            root = Path(td)
            payload = {
                "summary": {"month": "2026-02", "requests": 1},
                "hourly": [{"top_paths": [{"path": "/events?utm_source=test", "requests": 1}]}],
            }
            (root / "2026-02.json").write_text(json.dumps(payload), encoding="utf-8")
            with self.assertRaisesRegex(RuntimeError, "path contains query string"):
                publish.privacy_check_dir(root, include_report=False)

    def test_merge_generated_months_preserves_immutable_existing_data(self) -> None:
        with tempfile.TemporaryDirectory() as td:
            base = Path(td)
            generated_dir = base / "generated"
            merged_dir = base / "merged"
            generated_dir.mkdir()
            merged_dir.mkdir()

            month = "2020-01"
            self._write_month_file(generated_dir / f"{month}.json", month, requests=999)
            self._write_month_file(merged_dir / f"{month}.json", month, requests=123)

            log = publish.Logger(base / "publish.log")
            with suppress_logger_stdout():
                publish.merge_generated_months(generated_dir, merged_dir, grace_days=0, log=log)

            with (merged_dir / f"{month}.json").open("r", encoding="utf-8") as f:
                merged_doc = json.load(f)
            self.assertEqual(merged_doc["summary"]["requests"], 123)

    def test_merge_generated_months_updates_current_month(self) -> None:
        with tempfile.TemporaryDirectory() as td:
            base = Path(td)
            generated_dir = base / "generated"
            merged_dir = base / "merged"
            generated_dir.mkdir()
            merged_dir.mkdir()

            month = publish.dt.datetime.now(publish.dt.timezone.utc).strftime("%Y-%m")
            self._write_month_file(generated_dir / f"{month}.json", month, requests=222)
            self._write_month_file(merged_dir / f"{month}.json", month, requests=111)

            log = publish.Logger(base / "publish.log")
            with suppress_logger_stdout():
                publish.merge_generated_months(generated_dir, merged_dir, grace_days=7, log=log)

            with (merged_dir / f"{month}.json").open("r", encoding="utf-8") as f:
                merged_doc = json.load(f)
            self.assertEqual(merged_doc["summary"]["requests"], 222)

    def test_rebuild_lifetime_and_manifest_from_month_shards(self) -> None:
        with tempfile.TemporaryDirectory() as td:
            base = Path(td)
            merged_dir = base / "merged"
            merged_dir.mkdir()

            self._write_month_file(merged_dir / "2025-12.json", "2025-12", requests=10)
            self._write_month_file(merged_dir / "2026-01.json", "2026-01", requests=20)

            generated_lifetime = base / "generated-lifetime.json"
            generated_lifetime.write_text(
                json.dumps({"log_files_processed": ["access_log", "access_log.1"]}),
                encoding="utf-8",
            )

            fake_bin = self._write_fake_rebuild_bin(base)
            log = publish.Logger(base / "publish.log")
            with suppress_logger_stdout():
                publish.rebuild_lifetime_and_manifest(merged_dir, generated_lifetime, "nfs", fake_bin, log)

            lifetime = json.loads((merged_dir / "lifetime.json").read_text(encoding="utf-8"))
            manifest = json.loads((merged_dir / "manifest.json").read_text(encoding="utf-8"))

            self.assertEqual(lifetime["total_requests"], 30)
            self.assertEqual(lifetime["log_files_processed"], ["access_log", "access_log.1"])
            self.assertEqual(manifest["month_count"], 2)
            self.assertEqual(manifest["months"], ["2025-12", "2026-01"])
            self.assertIn("published_at", manifest)
            self.assertIn("2025-12.json", manifest["file_checksums_sha256"])
            self.assertIn("2026-01.json", manifest["file_checksums_sha256"])
            self.assertIn("lifetime.json", manifest["file_checksums_sha256"])


if __name__ == "__main__":
    unittest.main()
