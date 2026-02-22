# Log Analyzer Design

## Scope

This document describes the current behavior of the log analyzer and the design
constraints it operates under. The source of truth is `log-analyzer/src/main.janet`.

## Goals

- Privacy: do not persist raw IPs, full User-Agent strings, or referrer URLs.
- Useful aggregates: hourly request counts, status codes, bytes, top paths.
- Re-runnable: regenerate analytics from raw logs at any time.
- Git-friendly: outputs are small, human-readable JSON.

## Non-goals

- Real-time streaming analytics.
- User/session tracking or long-lived identifiers.
- Storing raw logs or per-request events in the repo.

## Assumptions

- Logs are Apache Combined format (`access_log*`) and live on the server.
- NFSN rotates logs weekly and retains 4-8 weeks.
- The analyzer has two modes:
  - `--mode analyze` generates month JSON shards + `lifetime.json`.
  - `--mode report` renders `report.html` from persisted JSON shards.
- The NFS cron wrapper runs analyze mode, then publish logic performs
  merge/validation/atomic publish and invokes report mode.

## Current Implementation

### Inputs

- All files in the log directory with names starting with `access_log`.
- Default log directory: `/home/logs` (override via CLI arg).

### CLI Flags

```
log-analyzer [flags] [log-directory]

Flags:
  --mode <analyze|report>    Operation mode (default: analyze)
  --out-dir, -o <dir>        Analyze mode output dir for month shards + lifetime.json
  --json-dir <dir>           Report mode input dir (default: /home/private/log-analyzer-data)
  --report-path <path>       Report mode output HTML path (default: <json-dir>/report.html)
```

### Processing

For each log line, the analyzer:

- Deduplicates using FNV-1a hash of the raw line (skips already-seen lines).
- Parses IP, timestamp, request path, status code, bytes, referrer, and
  User-Agent using a PEG grammar.
- Strips query strings from request paths before aggregation/output.
- Converts timestamps to UTC using the log timezone offset.
- Buckets by hour and month (in UTC).
- Tracks hourly counts and unique IPs (exact per hour, in-memory hash set).
- Updates monthly HyperLogLog (HLL) sketches for approximate uniques.
- Updates monthly exact IP hash sets for exact uniques (while logs exist).
- Updates a total HLL across all months.
- Classifies referrers into categories (direct/search/social/internal/external).
- Classifies browser family and OS platform for non-bot traffic.
- Bot detection via substring matching on User-Agent.

### Output Schema (stdout mode)

The analyzer outputs a single JSON document to stdout:

```json
{
  "generated_at": "2026-01-11T01:00:00Z",
  "log_files_processed": ["access_log", "access_log.20260104"],
  "total_requests": 8500,
  "total_unique_visitors_estimate": 1200,
  "total_unique_visitors_hll": "SExMAQ4B...",
  "hourly": [
    {
      "hour": "2026-01-01T00:00:00Z",
      "requests": 23,
      "bytes_sent": 340000,
      "unique_ips": 12,
      "bots": 3,
      "scans": 2,
      "visitors": 18,
      "status_codes": {"200": 20, "304": 2, "404": 1},
      "status_by_type": {
        "bots": {"200": 2, "404": 1},
        "scans": {"404": 2},
        "visitors": {"200": 18}
      },
      "top_paths": [
        {"path": "/", "requests": 15},
        {"path": "/events.json", "requests": 8},
        {"path": "other", "requests": 3}
      ],
      "referrer_categories": {"direct": 10, "search": 5, "external": 3},
      "browsers": {"Chrome": 10, "Safari": 5, "Firefox": 3},
      "platforms": {"Android": 8, "iOS": 5, "Windows": 4}
    }
  ],
  "monthly": [
    {
      "month": "2026-01",
      "requests": 8500,
      "unique_visitors_exact": 1200,
      "unique_visitors_estimate": 1198,
      "unique_visitors_hll": "SExMAQ4B..."
    }
  ]
}
```

Notes:
- `top_paths` is the top 20 paths per hour, plus an `other` bucket for the rest.
- `monthly` includes both exact and HLL-estimated unique visitor counts.
- `browsers` and `platforms` are tracked for non-bot traffic only.
- `referrer_categories` classifies without storing full URLs.

### Output Schema (analyze mode with `--out-dir <dir>`)

When `--out-dir` is specified, the analyzer writes:

- `YYYY-MM.json` per month (that month's hourly data + monthly summary)
- `lifetime.json` (merged HLLs, total counts, list of months)

`report.html` is generated separately by report mode:

- `log-analyzer --mode report --json-dir <dir> --report-path <path>`

## Data Flow

```
raw logs (server only)
  -> log-analyzer --mode analyze (dedup + parse + classify + aggregate)
  -> generated month shards + lifetime.json
  -> publish pipeline (privacy checks + immutable month merge + lifetime/manifest rebuild)
  -> log-analyzer --mode report (render report.html from persisted JSON)
  -> atomic publish to /home/private/log-analyzer-data and /home/public/analytics
```

## Privacy Model

- Raw IPs are used in memory for hourly/monthly uniqueness and HLL inputs,
  then discarded.
- HLL stores hashed values (FNV-1a + Murmur finalizer) and is not reversible.
- User-Agent is used for bot detection, browser/OS classification, then
  discarded. Only category names are stored, not the full string.
- Referrer URLs are classified into categories (direct/search/social/internal/
  external) and discarded. Only category counts are stored.

## Time Handling

- The timezone offset from each log line is parsed and used to convert
  timestamps to UTC before bucketing.
- Buckets are hourly, and conversion currently applies hour-level offset
  adjustment (minute offsets are rare in server logs and effectively round to
  nearest hour boundary behavior).
- All output timestamps use the `Z` suffix and represent UTC-hour buckets.
- Hour buckets are in UTC regardless of the server's local timezone.

## Line Deduplication

- Each raw log line is hashed (FNV-1a, 32-bit) and stored in an in-memory set.
- Duplicate lines (from overlapping rotated log files) are skipped.
- Dedup stats are reported to stderr per file.
- This makes request/byte/status counts correct even with overlapping files.
- Tradeoff: 32-bit hashing keeps memory bounded, but very rare hash collisions can
  undercount requests.

## Classification

### Bot Detection

Substring-based on lowercased User-Agent. Matches: bot, crawler, spider,
scraper, curl, wget, python, go-http-client, gptbot, claudebot, bingbot,
googlebot, facebookexternalhit, twitterbot.

### Referrer Categories

- `direct`: referrer is "-" or empty
- `search`: google, bing, duckduckgo, baidu, yandex, yahoo, ecosia
- `social`: facebook, twitter/x, t.co, reddit, linkedin, instagram, mastodon
- `internal`: plazaespana.info
- `external`: everything else

### Browser Families

Order matters (check specific before general):
Edge, Opera, Samsung Internet, Firefox, Chrome, Safari, other.

### Platform Families

iPhone/iPad → iOS, Android, CrOS → ChromeOS, Windows, Macintosh → macOS,
Linux, other.

## Accuracy and Idempotency

- Line deduplication makes the analyzer idempotent for all metrics when
  processing overlapping log files.
- Hourly `unique_ips` is exact for that hour (hash set).
- Monthly `unique_visitors_exact` is exact for that month (hash set).
- Monthly `unique_visitors_estimate` is approximate (HLL, ~0.81% error).
- Total uniques are approximate (HLL across all months).
- All counts (requests, bytes, status codes) are exact.

## HTML Report

A self-contained static HTML file with:
- Summary cards (total requests, unique visitors, months, log files)
- Monthly summary table (requests, exact uniques, HLL estimates)
- Daily request bar chart (CSS-only, no JavaScript)
- Top paths table (aggregated across all hours)
- Browser, platform, and referrer category breakdowns
- Dark mode support via `prefers-color-scheme`

## Operational Notes

- The analyzer is stateless: every run reprocesses all `access_log*` files.
- Line dedup handles overlap between current and rotated log files.
- For crash-safe updates, write to a temp file and `rename` atomically.
- Memory: ~40 bytes per unique line hash + ~40 bytes per unique IP per month
  for exact counts. Acceptable for sites with <100K lines.

## Sanity Checks

- `unique_visitors_exact` should be <= `requests` per month.
- `unique_visitors_estimate` should be close to `unique_visitors_exact`
  (within ~1% for sets >1000).
- Monthly totals should match the sum of hourly totals.
- Dedup counts should be small unless log files genuinely overlap.
