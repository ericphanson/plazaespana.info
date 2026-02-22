#!/usr/bin/env python3
"""Fail if published analytics report/manifest is older than allowed."""

from __future__ import annotations

import argparse
import datetime as dt
import json
import re
from urllib.request import urlopen


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Check analytics freshness.")
    parser.add_argument(
        "--manifest-url",
        default="https://plazaespana.info/analytics_report.html",
        help="Public analytics URL (report HTML or manifest JSON).",
    )
    parser.add_argument(
        "--max-age-days",
        type=int,
        default=3,
        help="Maximum allowed staleness in days",
    )
    return parser.parse_args()


def main() -> int:
    args = parse_args()
    with urlopen(args.manifest_url, timeout=20) as resp:
        payload = resp.read()
    text = payload.decode("utf-8", errors="replace")

    published_at: str | None = None
    source = "report_html"
    try:
        doc = json.loads(text)
    except json.JSONDecodeError:
        doc = None

    if isinstance(doc, dict):
        candidate = doc.get("published_at")
        if isinstance(candidate, str) and candidate:
            published_at = candidate
            source = "manifest_json"
    else:
        match = re.search(r"Generated:\s*([0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9:]{8}Z)", text)
        if match:
            published_at = match.group(1)

    if not published_at:
        raise RuntimeError("unable to find generated/published timestamp in analytics payload")

    published = dt.datetime.fromisoformat(published_at.replace("Z", "+00:00"))
    now = dt.datetime.now(dt.timezone.utc)
    age = now - published
    max_age = dt.timedelta(days=args.max_age_days)

    print(f"manifest_url={args.manifest_url}")
    print(f"source={source}")
    print(f"published_at={published_at}")
    print(f"age_hours={age.total_seconds() / 3600:.2f}")
    print(f"max_age_days={args.max_age_days}")

    if age > max_age:
        raise RuntimeError(
            f"analytics manifest is stale: age={age}, threshold={max_age}, "
            f"published_at={published_at}"
        )

    print("analytics manifest freshness check passed")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
