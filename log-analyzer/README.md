# Log Analyzer

Privacy-preserving Apache access log analyzer for plazaespana.info. Compiled Janet executable, deployed to NFSN and run as a daily cron job.

## Deployment

The FreeBSD binary (`log-analyzer-freebsd`) is built in CI and deployed to NFSN as part of every push to `main` — no manual deploy step needed. The deploy also uploads `ops/log-analyzer-daily.sh` (cron wrapper) and `ops/log-analyzer-publish.py` (publish pipeline) to `/home/private/bin/` on the server.

On NFSN, configure a daily scheduled task running:
```
/home/private/bin/log-analyzer-daily.sh
```

## Results

The report is published at **`plazaespana.info/analytics-report.html`** after each daily run.

Private JSON data (month shards + `lifetime.json`) is stored in `/home/private/log-analyzer-data/` and backed up to Bunny CDN storage.

## Local Development

```bash
# Build and test
just build
just test

# Build FreeBSD binary (done automatically by CI)
just log-analyzer-freebsd   # from repo root: just log-analyzer-freebsd
```

For troubleshooting the cross-compile, see [BUILDING.md](BUILDING.md). For the full output schema and design, see [DESIGN.md](DESIGN.md).

## Implementation Notes

Written in [Janet](https://janet-lang.org/), a small Lisp-like language that compiles to a standalone C amalgamation. This makes cross-compilation tractable: the build script (`build-freebsd-zig.sh`) patches the Janet source to enable `JANET_SINGLE_THREADED` (no pthread dependency), generates the amalgamation via `make`, then uses Zig as a cross-compiler targeting `x86_64-freebsd`. The result is a single static binary with no runtime dependencies.

The analyzer is intentionally stateless — it re-reads all `access_log*` logs from scratch each run and deduplicates by hashing raw lines. Exact per-month counts are preserved in `YYYY-MM.json` shards; once a month closes those files are treated as immutable. Long-term unique visitor counts use HyperLogLog sketches stored in the month files, which can be merged across months without storing any IPs.
