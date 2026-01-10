# Log Analyzer

Apache access log analyzer that counts unique visitors per month. Compiled Janet executable.

## Features

- Parses Apache Combined Log Format using PEG grammar
- Tracks unique IP addresses (IPv4 and IPv6) per month
- Provides all-time unique visitor count
- Fast processing with progress indicators

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
- Produces a 712KB FreeBSD ELF executable

## Testing

```bash
# Run all tests
jpm test
```

Tests are in `test/` and cover the HyperLogLog implementation.

## Usage

```bash
# Use default log directory
./build/log-analyzer

# Specify custom log directory
./build/log-analyzer /path/to/logs
```

### On NFSN

```bash
/home/private/bin/log-analyzer /home/logs
```

## Output

```
===================================================
Unique Visitors Per Month
===================================================

2025-10:    869 unique visitors
2025-11:   1086 unique visitors
2025-12:   1364 unique visitors
2026-01:    365 unique visitors

---------------------------------------------------
Total unique visitors (all time): 3236
===================================================
```

## Binary Details

- **Type**: Mach-O 64-bit executable (macOS ARM64)
- **Size**: ~728KB
- **Dependencies**: Self-contained (no Janet installation required)
- **Tree-shaken**: Only includes used Janet stdlib functions

## Project Structure

```
log-analyzer/
├── project.janet          # Build configuration
├── src/
│   ├── main.janet        # Main source code
│   └── hll.janet         # HyperLogLog implementation
├── test/
│   └── hll-test.janet    # HLL tests (run with jpm test)
├── build/
│   ├── log-analyzer      # Compiled executable
│   ├── log-analyzer.c    # Generated C source
│   └── *.o               # Object files
└── README.md
```

## Default Log Directory

`/Users/eph/plazaespana.info/awstats-data/logs`

## Requirements

- **Build time**: Janet with jpm (for `jpm build`)
- **Run time**: None (standalone executable)
