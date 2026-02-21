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
- Top paths per hour with "other" bucket
- Optional split output: per-month JSON + lifetime.json + HTML report

## Building

### Local Build (macOS/Linux)

```bash
# Build the executable
jpm build

# Executable will be in: build/log-analyzer
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
- Builds Janet with `JANET_SINGLE_THREADED` enabled (no pthread dependency)
- Generates a proper Janet amalgamation via `make` before cross-compiling
- Produces a ~724KB FreeBSD ELF executable

## Testing

```bash
# Run all tests
jpm test
```

Tests are in `test/` and cover the HyperLogLog implementation and the main analyzer functions (parsing, timezone conversion, classification, path bucketing).

## Usage

```bash
# JSON to stdout (default)
./build/log-analyzer /path/to/logs

# Split output: per-month JSON + lifetime.json + report.html
./build/log-analyzer --out-dir /path/to/output /path/to/logs
```

### On NFSN

```bash
/home/private/bin/log-analyzer /home/logs
```

## Output

JSON to stdout by default. See [DESIGN.md](DESIGN.md) for the full output schema.

With `--out-dir`, writes:
- `YYYY-MM.json` per month (hourly data + monthly summary)
- `lifetime.json` (merged HLLs, total counts, list of months)
- `report.html` (self-contained static HTML, no JavaScript, dark mode)

## Data Retention

The analyzer is stateless — it re-reads all `access_log*` files from scratch every run. This means accuracy depends on what you keep around:

- **Raw logs** (`access_log*`): Needed for exact counts. NFSN rotates weekly and retains 4-8 weeks, so exact hourly/monthly stats cover that window. Once a log file is deleted, its data is gone from future runs.
- **Monthly JSON** (`YYYY-MM.json`): Preserves exact counts and hourly breakdowns for that month. Keep these in the repo indefinitely — they're small and won't change once the month's logs are rotated away.
- **HLL sketches** (`unique_visitors_hll` in each monthly file): These are the key to long-term unique visitor counts. The lifetime total in `lifetime.json` is computed by merging all monthly HLLs. Even after raw logs are deleted, the HLL gives a ~1% accurate unique count for that month, and merging HLLs correctly handles visitors who appear in multiple months.

In short: raw logs are ephemeral, monthly JSON files are the permanent record, and HLL sketches are what make lifetime unique counts possible without storing any IPs.

## Project Structure

```
log-analyzer/
├── project.janet          # Build configuration
├── DESIGN.md              # Detailed design and output schema
├── src/
│   ├── main.janet         # Main source code
│   ├── hll.janet          # HyperLogLog implementation
│   └── json.janet         # JSON encoder
├── test/
│   ├── hll-test.janet     # HLL tests
│   └── main-test.janet    # Analyzer function tests
├── build/
│   ├── log-analyzer       # Compiled executable
│   └── log-analyzer.c     # Generated C source
└── README.md
```

## Requirements

- **Build time**: Janet with jpm (for `jpm build`)
- **Run time**: None (standalone executable)
