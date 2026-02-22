#!/usr/bin/env python3
"""Fail if published analytics manifest is older than allowed."""

from __future__ import annotations

import argparse
import datetime as dt
import json
from urllib.request import urlopen


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Check analytics manifest freshness.")
    parser.add_argument(
        "--manifest-url",
        default="https://plazaespana.info/analytics/data/manifest.json",
        help="Public manifest.json URL",
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
    doc = json.loads(payload.decode("utf-8"))

    published_at = doc.get("published_at")
    if not isinstance(published_at, str) or not published_at:
        raise RuntimeError("manifest missing published_at")

    published = dt.datetime.fromisoformat(published_at.replace("Z", "+00:00"))
    now = dt.datetime.now(dt.timezone.utc)
    age = now - published
    max_age = dt.timedelta(days=args.max_age_days)

    print(f"manifest_url={args.manifest_url}")
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
