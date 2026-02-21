# Log Analyzer

Apache access log analyzer that outputs JSON analytics with privacy-preserving unique visitor tracking. Compiled Janet executable.

## Features

- Parses Apache Combined Log Format using PEG grammar
- Converts timestamps to UTC using log timezone offset
- Deduplicates overlapping rotated log files
- Tracks unique IPs per hour (exact) and per month (exact + HLL)
- HLL sketches stored as base64 for merging across log rotations
- Classifies referrers (direct/search/social/internal/external)
- Classifies browsers and platforms for non-bot traffic
- Bot detection via User-Agent substring matching
- Top paths per hour with "other" bucket (query strings stripped)
- Two-phase CLI:
  - `analyze` mode generates JSON artifacts
  - `report` mode consumes JSON artifacts and renders `report.html`

## Building

### Local Build (macOS/Linux)

```bash
# Build the executable
jpm build

# Executable will be in: build/log-analyzer
```

Using `just`:

```bash
# Show available commands
just

# Build and test
just build
just test
```

### FreeBSD Cross-Compilation (for NFSN Deployment)

Cross-compile for FreeBSD using Zig (works from macOS/Linux):

```bash
# Build FreeBSD binary with Zig cross-compilation
./build-freebsd-zig.sh

# Deploy to NFSN
scp build/log-analyzer-freebsd $NFSN_USER@$NFSN_HOST:/home/private/bin/log-analyzer
```

**How it works:**
- Uses Zig's cross-compilation to target `x86_64-freebsd`
- Pins Janet to `v1.41.2` for reproducible builds
- Builds Janet with `JANET_SINGLE_THREADED` enabled (no pthread dependency)
- Generates a Janet amalgamation (`build/c/janet.c`) via `make` before cross-compiling
- Reuses cached Janet sources/amalgamation in `/tmp` for faster rebuilds
- Builds both binaries in one run:
  - `build/log-analyzer-freebsd` (`x86_64-freebsd`)
  - `build/log-analyzer` (native macOS arch)

**Cache control flags (for troubleshooting):**
- `JANET_FORCE_REFRESH=1 ./build-freebsd-zig.sh` (refresh cached Janet clone in `/tmp`)
- `JANET_FORCE_AMALG_REBUILD=1 ./build-freebsd-zig.sh` (rebuild cached amalgamation in `/tmp`)

## Testing

```bash
# Run all tests (Janet + Python HLL interop checks)
just test
```

Tests cover the HyperLogLog implementation, analyzer functions (parsing, timezone conversion, classification, path bucketing), JSON encoder/decoder roundtrips, report rendering safety, and Python/Janet HLL compatibility for publish-time merging.

## Usage

```bash
# JSON to stdout (default)
./build/log-analyzer /path/to/logs

# Phase 1: generate per-month JSON + lifetime.json
./build/log-analyzer --mode analyze --out-dir /path/to/output /path/to/logs

# Phase 2: render report from JSON artifacts
./build/log-analyzer --mode report --json-dir /path/to/output --report-path /path/to/output/report.html

# If no log directory is provided, defaults to /home/logs
./build/log-analyzer
```

### On NFSN

```bash
# Production entrypoint (analyze + publish + backup + report)
/home/private/bin/log-analyzer-daily.sh

# Direct analyzer invocation (debug only)
/home/private/bin/log-analyzer --mode analyze --out-dir /tmp/log-analyzer-out /home/logs
```

## Output

JSON to stdout by default. See [DESIGN.md](DESIGN.md) for the full output schema.

With `--mode analyze --out-dir`, writes:
- `YYYY-MM.json` per month (hourly data + monthly summary)
- `lifetime.json` (merged HLLs, total counts, list of months)

With `--mode report --json-dir ... --report-path ...`, writes:
- `report.html` (self-contained static HTML, no JavaScript, dark mode) from persisted JSON artifacts

## Data Retention

The analyzer is stateless — it re-reads all `access_log*` files from scratch every run. This means accuracy depends on what you keep around:

- **Raw logs** (`access_log*`): Needed for exact counts. NFSN rotates weekly and retains 4-8 weeks, so exact hourly/monthly stats cover that window. Once a log file is deleted, its data is gone from future runs.
- **Monthly JSON** (`YYYY-MM.json`): Preserves exact counts and hourly breakdowns for that month. Keep these persisted indefinitely (for example in `/home/private/log-analyzer-data` plus Bunny backup) — they're small and should remain immutable once a month is closed.
- **HLL sketches** (`unique_visitors_hll` in each monthly file): These are the key to long-term unique visitor counts. The lifetime total in `lifetime.json` is computed by merging all monthly HLLs. Even after raw logs are deleted, the HLL gives a ~1% accurate unique count for that month, and merging HLLs correctly handles visitors who appear in multiple months.
- **Compressed logs** (`access_log*.gz`): The NFS cron wrapper expands `.gz` logs into a temp dir before `analyze` mode runs, so rotated compressed files are included.

In short: raw logs are ephemeral, monthly JSON files are the permanent record, and HLL sketches are what make lifetime unique counts possible without storing any IPs.

## Project Structure

```
log-analyzer/
├── justfile              # Developer commands (build/test/run/freebsd)
├── project.janet          # Build configuration
├── DESIGN.md              # Detailed design and output schema
├── src/
│   ├── main.janet         # Main source code
│   ├── hll.janet          # HyperLogLog implementation
│   ├── json.janet         # JSON encoder
│   ├── json_decode.janet  # JSON decoder (for report mode)
│   └── report.janet       # Report renderer from persisted JSON
├── test/
│   ├── hll-test.janet     # HLL tests
│   ├── json-roundtrip-test.janet # JSON encoder/decoder roundtrip tests
│   ├── main-test.janet    # Analyzer function tests
│   └── report-test.janet  # Report rendering safety tests
├── build/
│   ├── log-analyzer       # Compiled executable
│   └── log-analyzer.c     # Generated C source
└── README.md
```

## Requirements

- **Build time**: Janet with jpm (for `jpm build`)
- **Test time**: `python3` (for publish/HLL interop tests run via `just test`)
- **Run time**: None (standalone executable)
