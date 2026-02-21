#!/usr/bin/env python3
"""Tests for HLL encode/decode/count logic in log-analyzer-publish.py."""

from __future__ import annotations

import importlib.util
from pathlib import Path
import shutil
import subprocess
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


if __name__ == "__main__":
    unittest.main()
