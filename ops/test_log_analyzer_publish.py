#!/usr/bin/env python3
"""Tests for HLL encode/decode/count logic in log-analyzer-publish.py."""

from __future__ import annotations

import importlib.util
import json
from pathlib import Path
import shutil
import subprocess
import tempfile
import textwrap
import unittest


OPS_DIR = Path(__file__).resolve().parent
REPO_ROOT = OPS_DIR.parent
PUBLISH_PATH = OPS_DIR / "log-analyzer-publish.py"


def load_publish_module():
    spec = importlib.util.spec_from_file_location("log_analyzer_publish", PUBLISH_PATH)
    if spec is None or spec.loader is None:
        raise RuntimeError(f"Unable to load module spec from {PUBLISH_PATH}")
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


publish = load_publish_module()


class HLLInteropTests(unittest.TestCase):
    def _require_janet(self) -> None:
        if shutil.which("janet") is None:
            self.skipTest("janet not found on PATH")

    def _build_janet_hll(self, body: str) -> tuple[str, int]:
        self._require_janet()
        program = textwrap.dedent(
            f"""
            (import ./log-analyzer/src/hll)
            (def h (hll/new))
            {body}
            (print (hll/to-base64 h))
            (print (math/round (hll/count-estimate h)))
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
        return lines[0], int(lines[1])

    def test_sparse_roundtrip(self) -> None:
        buckets = [0] * publish.HLL_NUM_BUCKETS
        for i in range(0, 800, 9):
            buckets[i] = (i % 31) + 1

        encoded = publish.encode_hll_b64(buckets)
        decoded = publish.decode_hll_b64(encoded)
        self.assertEqual(decoded, buckets)

    def test_dense_roundtrip(self) -> None:
        buckets = [((i % 31) + 1) for i in range(publish.HLL_NUM_BUCKETS)]
        encoded = publish.encode_hll_b64(buckets)
        decoded = publish.decode_hll_b64(encoded)
        self.assertEqual(decoded, buckets)

    def test_python_count_matches_janet_count(self) -> None:
        b64, janet_count = self._build_janet_hll(
            """
            (for i 0 5000
              (hll/add h (string "ip-" i)))
            # Add overlap and duplicates to mimic real log traffic.
            (for i 0 2000
              (hll/add h (string "ip-" i)))
            (for i 0 1200
              (hll/add h (string "returning-" (% i 150))))
            """
        )

        buckets = publish.decode_hll_b64(b64)
        python_count = publish.round_half_up(publish.hll_count_estimate(buckets))
        self.assertLessEqual(abs(python_count - janet_count), 1)

    def test_python_reencode_matches_janet_sparse_and_dense(self) -> None:
        sparse_b64, _ = self._build_janet_hll(
            """
            (for i 0 160
              (hll/add h (string "sparse-" i)))
            """
        )
        dense_b64, _ = self._build_janet_hll(
            """
            (for i 0 12000
              (hll/add h (string "dense-" i)))
            """
        )

        self.assertEqual(publish.encode_hll_b64(publish.decode_hll_b64(sparse_b64)), sparse_b64)
        self.assertEqual(publish.encode_hll_b64(publish.decode_hll_b64(dense_b64)), dense_b64)


class PublishPipelineTests(unittest.TestCase):
    def _write_month_file(self, path: Path, month: str, requests: int, hll_b64: str | None = None) -> None:
        if hll_b64 is None:
            hll_b64 = publish.encode_hll_b64([0] * publish.HLL_NUM_BUCKETS)
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

            publish.rebuild_lifetime_and_manifest(merged_dir, generated_lifetime, "nfs")

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
