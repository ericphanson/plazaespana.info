# Design Document: Web Analytics Aggregation System

## Problem Statement

Replace AWStats with a modern, maintainable analytics system that:
- Processes Apache Combined Log Format access logs
- Generates accurate, privacy-preserving aggregate statistics
- Supports temporal slicing (by hour/day/month)
- Can identify and filter bot/scraper traffic
- Stores semi-aggregated data (not full raw logs) in version control
- Integrates with existing FreeBSD hosting and Go toolchain

**Current pain points:**
- AWStats is unmaintained (last update 2020)
- Severely misconfigured (processing 1 record/update despite 2795+ log entries)
- Complex Perl dependencies and brittle configuration
- Opaque binary database format
- 500 Google Search Console clicks vs 33 tracked visits = 93% data loss

## Design Constraints

1. **Static site architecture**: No runtime server, all analytics must be batch-processed
2. **FreeBSD hosting**: Must cross-compile for FreeBSD/amd64
3. **Privacy-first**: No individual IPs stored in git repository
4. **Git storage**: Aggregate data must be git-friendly (textual, diffable, small)
5. **Cron execution**: Runs hourly on NFSN server
6. **Low overhead**: Minimal resource usage, fast processing

**User preferences (from questions):**
- ✅ Hourly granularity
- ✅ JSON storage format
- ✅ Standalone binary
- ✅ Slice by: time, path, referrer category, browser/OS, bot detection signals

## The Log Aggregation Problem Space

### What Are We Actually Trying to Solve?

Web server logs contain individual request records:
```
4.230.44.177 - - [04/Jan/2026:00:35:02 +0000] "GET / HTTP/1.1" 200 91996 "-" "Mozilla/5.0..."
```

We need to transform millions of these into queryable aggregates:
- "How many unique visitors per day?"
- "What browsers are people using?"
- "Which pages are most popular?"
- "How much traffic is from bots vs humans?"

**The core challenge:** Balance between **granularity** (detail level) and **storage size** (data volume).

### Common Pitfalls

#### 1. **The "Store Everything" Trap**
**Problem:** Keep all raw logs forever "just in case"
- 2795 log lines ≈ 662 KB
- 1 year at this rate ≈ 34 MB of logs
- Seems small, but: logs are not diffable, pollute git history, no query capability

**How others handle it:**
- **Matomo/Plausible**: Store individual page views in database, periodically archive old data
- **GoAccess**: Processes logs on-demand, doesn't persist aggregates
- **Grafana Loki**: Indexes logs but stores compressed chunks, auto-expires old data
- **Industry standard**: Keep 30-90 days of raw logs, archive rest to cold storage (S3 Glacier)

#### 2. **The "Too Coarse" Problem**
**Problem:** Aggregate too heavily, lose ability to answer questions
- Example: Only store "visits per month" → can't see hourly traffic patterns
- Example: Only store "total browser counts" → can't see how browser share evolved over time

**How others handle it:**
- **GoAccess**: Stores hourly breakdowns, daily summaries, top N lists (configurable thresholds)
- **AWStats**: Stores hourly/daily/monthly, but also per-URL, per-referrer, per-browser
- **Plausible**: Time-series bucketed by hour, dimensions (page/referrer/country) stored separately

**Best practice:** Use hierarchical aggregation
- Finest grain: Hourly buckets with dimensions
- Coarser grain: Daily/monthly rollups (pre-computed for fast queries)
- Top-N tracking: Keep full list for recent data, top 100 for historical

#### 3. **The Unique Visitor Counting Problem**
**Problem:** Accurately count unique visitors without storing IPs

**Why it's hard:**
- Raw approach: Hash IPs per time bucket → requires storing hashes
- Naive aggregation: Sum up "unique visitors per hour" ≠ "unique visitors per day" (same visitor across hours counted multiple times)

**How others solve it:**
- **HyperLogLog (HLL)**: Probabilistic data structure, ~1% error, constant memory
  - Used by: Redis, Postgres, Druid, ClickHouse
  - Storage: ~12 KB per counter (regardless of cardinality)
  - Can merge HLL sketches across time buckets

- **Bloom filters**: Track "have we seen this IP this month?"
  - More accurate than HLL but can't merge across time periods

- **Time-bucketed hashes**: Store IP hashes per day, discard after aggregation
  - Used by: GoAccess, AWStats
  - Privacy concern: Hashes can be reversed for common IPs

- **Accept limitations**: Only count "visits" (sessions), not unique IPs
  - Used by: Plausible (session = 30min window)
  - Simpler, privacy-friendly, "good enough" for most use cases

**Recommendation for this project:**
- Short-term (current month): Use HyperLogLog for unique IP counting
- Long-term (archived months): Store only visit counts, not uniques
- Rationale: HLL gives accurate uniques without storing IPs, old data doesn't need exact uniques

#### 4. **The Bot Detection Challenge**
**Problem:** 30-70% of web traffic is bots/scrapers, skews all metrics

**Detection signals:**
- User-Agent string: "bot", "crawler", "spider", "GPTBot", "ClaudeBot"
- Request patterns: High request rate, sequential page access, no referrer
- HTTP signatures: Specific header combinations, missing Accept-Language
- Behavioral: No JavaScript execution, ignores robots.txt, hits non-linked pages

**How others handle it:**
- **GoAccess**: Simple User-Agent regex matching against known bot list
- **Matomo**: Bot detection + device fingerprinting
- **Plausible**: Filter known bots, count sessions (requires JavaScript - not applicable here)
- **Cloudflare Analytics**: ML-based bot scoring (proprietary)

**Bot list sources:**
- [matomo-org/device-detector](https://github.com/matomo-org/device-detector) - 1000+ bot patterns
- [atmire/COUNTER-Robots](https://github.com/atmire/COUNTER-Robots) - Academic crawler list
- [monperrus/crawler-user-agents](https://github.com/monperrus/crawler-user-agents) - Actively maintained regex list

**Recommendation:**
- Maintain bot regex list in config/JSON
- Classify traffic as: human, known-bot, suspected-bot, unknown
- Store separate counts for each category
- Can retroactively reclassify by reprocessing aggregates (benefit of storing per-UA-family counts)

#### 5. **The Incremental Update Problem**
**Problem:** Logs arrive continuously, need to update aggregates without reprocessing everything

**Naive approach:**
1. Read entire log file
2. Parse all entries
3. Rebuild aggregates from scratch
→ Slow for large logs, wastes CPU

**How others handle it:**

**A. Checkpoint/cursor approach** (AWStats, GoAccess with --persist):
- Store "last processed line offset" or "last processed timestamp"
- On next run, seek to checkpoint and process only new lines
- **Pitfall:** Log rotation breaks offset pointers
- **Solution:** Use timestamp-based cursors + handle rotation

**B. Time-bucketed files** (Grafana Loki, Prometheus):
- Write logs to time-partitioned files: `access_log.2026-01-10`
- Only process files within update window
- **Pitfall:** Late-arriving data (e.g., buffered logs) can miss their bucket
- **Solution:** Reprocess last N hours to catch stragglers

**C. Write-ahead aggregates** (custom batch systems):
- Maintain "current hour" aggregate in memory
- Flush completed hours to permanent storage
- **Pitfall:** Crashes lose in-progress data
- **Solution:** Periodic checkpointing + accept small data loss window

**D. Stateless reprocessing with deduplication** (simple, robust):
- Always reprocess last 24-48 hours of logs
- Use deterministic aggregation (same input → same output)
- **Pitfall:** Wastes CPU on redundant processing
- **Solution:** Accept the overhead (2-3k lines/day is trivial for Go to parse)

**Recommendation for this project:** Approach D (stateless reprocessing)
- Rationale: Logs are small (~3k lines/day), Go parser is fast (~1-2ms for 10k lines)
- Benefit: Simple implementation, no state management, naturally handles log rotation
- Trade-off: Slight CPU overhead vs complexity reduction

#### 6. **The Multi-Dimensional Slicing Problem**
**Problem:** Want to query "iOS users from Google who visited /events.json on Jan 5 between 2-3pm"

**Storage explosion:**
- 24 hours × 10 paths × 5 referrer categories × 5 OS families × 5 browsers = 30,000 combinations/day
- Most are zero (sparse data)
- Pre-computing all combinations wastes space

**How others solve it:**

**A. Star schema** (traditional data warehousing):
- Fact table: [timestamp, visit_count, page_id, referrer_id, browser_id, os_id]
- Dimension tables: browsers[id, name], pages[id, url], etc.
- Used by: Snowflake, BigQuery, Redshift
- **Downside:** Requires SQL database

**B. Columnar storage** (OLAP):
- Store each dimension in separate column files
- Tools: Parquet, Apache Arrow, ClickHouse
- **Downside:** Complex tooling, not git-friendly

**C. Nested JSON with sparse encoding** (document DBs):
```json
{
  "2026-01-05T14:00:00Z": {
    "total_requests": 45,
    "by_path": {
      "/": {"requests": 30, "by_browser": {"chrome": 20, "safari": 10}},
      "/events.json": {"requests": 15, "by_browser": {"chrome": 10}}
    }
  }
}
```
- Used by: Elasticsearch, MongoDB aggregations
- **Downside:** Hard to query without loading entire document

**D. Separate aggregates per slice** (simple, git-friendly):
```
analytics/
  2026/01/
    hourly_traffic.json         # Total requests per hour
    paths.json                  # Top 100 paths with hourly counts
    referrers.json              # Referrer categories per hour
    browsers.json               # Browser families per hour
    bots.json                   # Bot vs human classification
```
- Used by: Custom systems, static site generators
- Query pattern: Load relevant file(s), filter/aggregate in memory
- **Downside:** Can't do arbitrary cross-dimensional queries without loading multiple files

**Recommendation for this project:** Approach D (separate files per dimension)
- Rationale: Data volume is low, queries are simple (not OLAP-style), git-friendly
- Each JSON file is small (<50 KB), human-readable, diffable
- Can add new dimensions by adding new files (forward-compatible)

### How Modern Tools Approach This

#### **GoAccess** (C, ~50k LOC)
- **Architecture:** In-memory hash tables, optional on-disk persistence
- **Storage:** Binary B+ trees (gdbm/tokyocabinet/berkeleydb)
- **Aggregation:** Real-time updates, configurable time buckets
- **Bot handling:** Regex list, configurable crawler detection
- **Output:** Static HTML + JSON + CSV
- **Unique visitors:** Hash IPs into hash table per time period
- **Strengths:** Fast, low memory (100k visitors ≈ 30 MB RAM), beautiful terminal UI
- **Weaknesses:** Not designed for version control, binary DB format, C codebase

#### **Matomo** (PHP, ~500k LOC)
- **Architecture:** MySQL database, async log import workers
- **Storage:** Relational tables (visits, actions, dimensions)
- **Aggregation:** Hierarchical (daily → weekly → monthly → yearly)
- **Bot handling:** Device Detector library (1000+ patterns)
- **Output:** Interactive web UI, APIs, scheduled reports
- **Unique visitors:** Cookies + fingerprinting (requires JS)
- **Strengths:** Feature-rich, enterprise-grade, privacy controls
- **Weaknesses:** Heavy (PHP + MySQL + Redis), requires runtime environment

#### **Plausible** (Elixir, ~50k LOC)
- **Architecture:** Elixir app + ClickHouse columnar database
- **Storage:** ClickHouse tables with time partitioning
- **Aggregation:** Real-time ingest + materialized views
- **Bot handling:** Filter known crawlers, session-based (requires JS)
- **Output:** Real-time dashboard, API
- **Unique visitors:** ClickHouse HyperLogLog
- **Strengths:** Modern UI, real-time, privacy-focused, GDPR-compliant
- **Weaknesses:** Requires ClickHouse (5+ GB RAM), not designed for log parsing

#### **Prometheus + Loki** (Go, time-series stack)
- **Architecture:** Prometheus (metrics) + Loki (logs) + Grafana (visualization)
- **Storage:**
  - Prometheus: Time-series DB with compression (1.4 bytes/sample)
  - Loki: Log chunks compressed with gzip, indexed by labels
- **Aggregation:** PromQL queries, recording rules for pre-aggregation
- **Bot handling:** N/A (general-purpose logging)
- **Output:** Grafana dashboards, alerting
- **Unique visitors:** Requires custom exporter, HyperLogLog in Prometheus
- **Strengths:** Industry standard, powerful querying, handles scale
- **Weaknesses:** Complex (3+ components), overkill for small sites, requires runtime

## Alternative: Building on GoAccess JSON Output

### Overview

GoAccess supports JSON export via `--output-format=json` or `goaccess --json-pretty-print`. Instead of building a log parser from scratch, we could:

1. Run GoAccess to generate JSON aggregates
2. Post-process the JSON to fit our schema
3. Commit transformed JSON to git repository

### GoAccess JSON Output Structure

GoAccess generates comprehensive JSON with these top-level sections:

```json
{
  "general": {
    "log_file": "/home/logs/access_log",
    "total_requests": 57,
    "unique_visitors": 26,
    "bandwidth": 2489088,
    "generation_time": "2026-01-10 01:00:06"
  },
  "visitors": [
    {"date": "10/Jan/2026:00:00", "hits": 5, "visitors": 3, "bytes": 150000},
    {"date": "10/Jan/2026:01:00", "hits": 8, "visitors": 5, "bytes": 220000}
  ],
  "requests": [
    {"url": "/", "hits": 33, "visitors": 26, "bytes": 1471479, "protocol": "HTTP/1.1", "method": "GET"},
    {"url": "/stats/", "hits": 13, "visitors": 1, "bytes": 858478}
  ],
  "browsers": [
    {"browser": "Chrome", "hits": 30, "visitors": 20},
    {"browser": "Safari", "hits": 15, "visitors": 10}
  ],
  "os": [
    {"os": "macOS", "hits": 25, "visitors": 15},
    {"os": "Android", "hits": 15, "visitors": 8}
  ],
  "referrers": [
    {"url": "https://www.google.com/search?q=...", "hits": 15, "visitors": 5}
  ],
  "status_codes": [
    {"code": "200", "hits": 40},
    {"code": "404", "hits": 5}
  ]
}
```

**Key capabilities:**
- Hourly/daily breakdowns (via `visitors` array)
- Top-N tracking for paths, referrers, browsers, OS
- Bot detection (configurable via `browsers.list` and `--exclude-crawler`)
- Bandwidth/bytes tracking
- Status code distribution

### Mapping to Our Requirements

| Requirement | GoAccess Support | Gap Analysis |
|-------------|------------------|--------------|
| **Hourly granularity** | ✅ Yes (via `--date-spec=hr` and `visitors` array) | None |
| **JSON output** | ✅ Yes (`--output-format=json`) | None |
| **Bot detection** | ✅ Yes (built-in browser list + custom regex) | Need to extract bot counts separately |
| **Privacy (no IPs)** | ✅ Yes (JSON contains only aggregates) | None |
| **Git-friendly** | ⚠️ Partial (JSON is text, but structure may change) | Need stable schema wrapper |
| **Unique visitors** | ✅ Yes (hash-based counting per time period) | Not HyperLogLog (can't merge across periods) |
| **Temporal slicing** | ✅ Yes (hourly, daily, monthly via `--date-spec`) | None |
| **Referrer categories** | ⚠️ Partial (stores raw referrers, not categorized) | Need post-processing to group search/direct/external |
| **Cross-compilation** | ❌ No (C binary, must compile on FreeBSD or cross-compile with GCC) | Requires FreeBSD toolchain or pre-built binary |

### Proposed Architecture

```
┌─────────────────┐
│  Access Logs    │
│  /home/logs/    │
└────────┬────────┘
         │
         │ (Hourly cron)
         ▼
┌─────────────────┐
│    GoAccess     │  Pre-built FreeBSD binary or cross-compiled
│  (log parser)   │  Flags: --json-pretty-print --date-spec=hr --exclude-crawler
└────────┬────────┘
         │
         │ (Raw JSON output)
         ▼
┌─────────────────┐
│ goaccess-json   │  Temporary file: /tmp/goaccess-YYYY-MM.json
│  (transient)    │
└────────┬────────┘
         │
         │ (Post-process)
         ▼
┌─────────────────┐
│  transform-stats│  Go binary, reads GoAccess JSON, transforms to our schema
│  (Go binary)    │  - Categorize referrers (search/direct/external)
└────────┬────────┘  - Separate bot counts
         │            - Add HyperLogLog for mergeable uniques (optional)
         │            - Normalize timestamps to RFC3339
         ▼
┌─────────────────┐
│  Analytics DB   │  JSON files: awstats-data/analytics/YYYY-MM.json
│  (Git storage)  │  (Our schema from "Recommended Design")
└─────────────────┘
```

### Implementation Effort

**Compared to from-scratch approach:**

| Component | From Scratch | GoAccess-based | Savings |
|-----------|--------------|----------------|---------|
| Log parsing | ~200 LOC | 0 LOC (GoAccess) | -200 LOC |
| Aggregation | ~300 LOC | 0 LOC (GoAccess) | -300 LOC |
| Bot detection | ~200 LOC | ~50 LOC (configure GoAccess list) | -150 LOC |
| JSON transform | 0 LOC | ~200 LOC (map GoAccess → our schema) | +200 LOC |
| **Total code** | ~700 LOC | ~250 LOC | **-450 LOC** |
| **Binary size** | ~2-3 MB | GoAccess ~400 KB + transform ~1 MB = ~1.4 MB | -1.6 MB |

**Additional overhead:**
- Must install/cross-compile GoAccess for FreeBSD (~5 MB binary)
- Two-binary pipeline (GoAccess → transform) vs. single binary
- Dependency on GoAccess maintenance and stability

### Pros and Cons

**Pros:**
1. **Battle-tested parser:** GoAccess has parsed billions of logs, handles edge cases (malformed lines, encoding issues, log rotation)
2. **Less code to maintain:** ~60% less custom code (450 LOC savings)
3. **Rich features out-of-box:** Bandwidth tracking, geo-IP (optional), detailed browser/OS detection
4. **Performance:** GoAccess is highly optimized C code (processes 100k lines/sec on modest hardware)
5. **Configurable bot detection:** Actively maintained crawler list, easy to update without recompiling

**Cons:**
1. **Two-binary complexity:** Must deploy and orchestrate GoAccess + transform binary
2. **Cross-compilation:** GoAccess requires C toolchain for FreeBSD, or must download pre-built binary
3. **Schema dependency:** If GoAccess changes JSON format, our transform breaks (low risk, but possible)
4. **Unique visitor merging:** GoAccess doesn't use HyperLogLog, can't merge "unique visitors per hour" to get "unique visitors per day" accurately
5. **Less control:** Can't easily extend log parser for custom fields or logic
6. **Binary size:** GoAccess is ~400 KB (vs. ~0 KB for pure Go parser)

### Recommendation

**Use GoAccess if:**
- You want to minimize custom code and maintenance burden
- You're comfortable with two-binary pipeline
- Unique visitor merging across time periods is not critical (accept "visits" instead of "true uniques")
- You can cross-compile or obtain FreeBSD binaries for GoAccess

**Build from scratch if:**
- You want single-binary deployment simplicity
- HyperLogLog-based unique visitor counting is important
- You want full control over aggregation logic and future extensibility
- You prefer pure Go toolchain (no C dependencies)

**Hybrid approach (best of both):**
1. **Phase 1:** Use GoAccess to get working analytics quickly (~1 day effort)
2. **Phase 2:** If limitations emerge (unique visitor merging, custom slicing), incrementally replace GoAccess with custom Go parser
3. **Benefit:** Ship fast, migrate later only if needed

**For this project, recommend:** **From-scratch approach** (as outlined in next section)
- Rationale: Log volume is low (~3k lines/day), Go parser is trivial (~200 LOC), single-binary deployment is simpler for NFSN hosting
- Trade-off: Write more code upfront, but gain full control and no external dependencies

---

## Alternative: From-Scratch Janet Implementation

### Overview

Janet is a functional Lisp-like language with built-in PEG (Parsing Expression Grammar) support that compiles to standalone C executables. A working prototype exists in `awstats-data/log-analyzer/` that demonstrates log parsing and unique visitor counting.

**Prototype stats:**
- **Code size:** 115 LOC (src/main.janet)
- **Binary size:** ~728 KB (macOS ARM64, self-contained)
- **Dependencies:** Zero (tree-shaken stdlib only)
- **Performance:** Fast (~10k lines/sec with progress dots)

### Why Janet for Log Processing?

Janet has several unique advantages for text processing tasks:

1. **Built-in PEG parser:** First-class pattern matching more powerful than regex
2. **Immutable-by-default:** Functional data structures reduce bugs
3. **Small binaries:** Tree-shaking produces 728 KB executables (vs 2-3 MB Go)
4. **Lisp syntax:** Expressive for data transformation
5. **Self-contained:** No runtime dependencies, includes Janet VM

### Current Prototype Capabilities

The existing `log-analyzer` implementation demonstrates:

**Apache log parsing with PEG:**
```janet
(def apache-log-peg
  ~{:main (* :ip :s "-" :s "-" :s "[" :date "]" :s (to "\n"))
    :ip (<- (some (+ :d (set ".:a-f"))))  # IPv4 or IPv6
    :date (* :day "/" :month "/" :year ":" :time :s :tz)
    :day (<- (between 1 2 :d))
    :month (<- (+ "Jan" "Feb" "Mar" "Apr" "May" "Jun"...))
    :year (<- (repeat 4 :d))})
```

**Unique visitor tracking:**
- Uses hash tables to track unique IPs per month
- Supports both IPv4 and IPv6 addresses
- Processes multiple log files with progress indicators
- Outputs formatted monthly summaries

**What's missing for full analytics:**
- User-Agent parsing (browser/OS detection, bot classification)
- Referrer categorization (search/direct/external)
- Path/URL aggregation with top-N tracking
- Hourly granularity (currently only monthly)
- JSON output (currently only terminal pretty-print)
- HyperLogLog for mergeable unique counts

### Mapping to Requirements

| Requirement | Janet Support | Implementation Effort |
|-------------|---------------|----------------------|
| **Apache log parsing** | ✅ PEG built-in (14 LOC) | Complete |
| **Unique IP tracking** | ✅ Hash tables (prototype working) | Extend to hourly buckets |
| **Hourly granularity** | ⚠️ Need to add bucketing | ~30 LOC (extend parse-log-line) |
| **JSON output** | ✅ stdlib `json/encode` | ~20 LOC (replace print-results) |
| **Bot detection** | ⚠️ Need User-Agent parsing + patterns | ~100 LOC (PEG + regex list) |
| **Browser/OS detection** | ⚠️ Need User-Agent library | ~150 LOC or vendor lib |
| **Referrer categorization** | ⚠️ Need URL parsing | ~50 LOC (PEG patterns) |
| **HyperLogLog** | ❌ No stdlib, need implementation | ~200 LOC or C FFI |
| **FreeBSD binary** | ❌ Must build on FreeBSD | See cross-compilation section |

### FreeBSD Cross-Compilation Challenge

**Current status:** Janet doesn't easily cross-compile to FreeBSD from macOS/Linux.

**Why it's hard:**
- Janet compiles to C via `jpm build`, then invokes system C compiler
- Cross-compilation requires FreeBSD toolchain (clang with FreeBSD sysroot)
- No easy "GOOS=freebsd GOARCH=amd64" equivalent like Go

**Workaround options:**

**Option 1: Build directly on NFSN (prototype approach)**
```bash
# SSH to NFSN FreeBSD server
ssh $NFSN_USER@$NFSN_HOST

# Install Janet via FreeBSD pkg
pkg install -y lang/janet  # Janet 1.35.2 available

# Upload source, build on server
cd /home/tmp/log-analyzer
jpm build
cp build/log-analyzer /home/private/bin/
```

**Pros:** Simple, guaranteed compatibility
**Cons:** Requires NFSN access for every build, can't test FreeBSD binary locally

**Option 2: FreeBSD VM/jail**
- Run FreeBSD in VM (bhyve, QEMU) or Docker (limited FreeBSD support)
- Build inside VM, copy binary out
- **Effort:** ~1-2 hours to set up VM

**Option 3: Cross-compilation toolchain**
- Install FreeBSD cross-compiler on macOS/Linux
- Configure `jpm` to use cross-compiler via `CC` env var
- **Effort:** ~3-5 hours, fragile, OS-specific

**Recommendation:** Use Option 1 (build on NFSN) for now, investigate Option 2 if frequent iteration needed.

### Implementation Effort to Full Analytics

**Starting from prototype (115 LOC), add:**

| Component | Lines of Code | Notes |
|-----------|---------------|-------|
| **Extend to hourly buckets** | ~30 LOC | Parse hour from timestamp, bucket by YYYY-MM-DD-HH |
| **User-Agent parsing (PEG)** | ~80 LOC | Extract browser/OS/version with PEG patterns |
| **Bot detection** | ~100 LOC | Regex list + classification logic |
| **Referrer categorization** | ~50 LOC | Parse referrer URL, classify as search/direct/external |
| **Path aggregation (top-N)** | ~60 LOC | Track path counts, sort, trim to top 100 |
| **JSON output** | ~30 LOC | Serialize data structure with `json/encode` |
| **Monthly file management** | ~40 LOC | Load existing month, merge, save atomically |
| **HyperLogLog (optional)** | ~200 LOC | Pure Janet impl, or skip if using hash-based uniques |
| **Total additions** | **~390-590 LOC** | |
| **Grand total** | **~505-705 LOC** | Comparable to Go estimate (500-800 LOC) |

**Development time estimate:**
- Core features (no HLL): ~2-3 days
- With HyperLogLog: ~3-4 days
- Testing + FreeBSD deployment: +1 day

### Comparison: Janet vs Go vs GoAccess

| Factor | Janet | Go | GoAccess + Transform |
|--------|-------|----|--------------------|
| **Total LOC** | ~505-705 | ~500-800 | ~250 |
| **Binary size** | ~1 MB | ~2-3 MB | ~1.4 MB (both binaries) |
| **PEG/parsing built-in?** | ✅ Yes (stdlib) | ❌ No (regex or vendor lib) | ✅ Yes (C, battle-tested) |
| **Cross-compile to FreeBSD** | ❌ Hard (build on target) | ✅ Trivial (`GOOS=freebsd`) | ❌ Hard (C toolchain) |
| **Development speed** | Fast (PEG, REPL) | Medium (verbose, compile cycle) | Fast (less code) |
| **Runtime performance** | Fast (C + JIT) | Very fast (native) | Very fast (C) |
| **HyperLogLog support** | ❌ DIY or skip | ✅ Vendor libs available | ❌ Not in GoAccess |
| **Browser/OS detection** | ⚠️ DIY PEG patterns | ✅ Vendor libs (user-agent pkg) | ✅ Built-in |
| **Bot detection** | ⚠️ DIY regex | ✅ Vendor libs | ✅ Built-in + configurable |
| **Maintainability** | Good (concise, functional) | Good (standard tooling) | Medium (two-binary pipeline) |
| **Learning curve** | High (Lisp, PEG) | Low (familiar syntax) | Low (config-driven) |

### Pros and Cons

**Pros:**
1. **Extremely concise:** PEG makes log parsing ~10x terser than regex/manual parsing
2. **Small binaries:** 728 KB vs 2-3 MB (Go) - matters for NFSN bandwidth/storage
3. **Functional style:** Immutable data structures reduce state bugs
4. **REPL-driven development:** Fast iteration with interactive testing
5. **Prototype exists:** Already 50% of the way there

**Cons:**
1. **FreeBSD build friction:** Can't cross-compile easily, must build on target or in VM
2. **Less mature ecosystem:** Fewer vendor libs for User-Agent parsing, bot detection, HLL
3. **Learning curve:** Lisp syntax and PEG unfamiliar to most developers
4. **No HyperLogLog:** Would need to implement from scratch (~200 LOC) or use hash-based uniques
5. **Smaller community:** Fewer Stack Overflow answers, tutorials, examples

### When to Choose Janet

**Use Janet if:**
- You value code conciseness and expressive PEG parsing
- You're comfortable with Lisp syntax and functional programming
- You can build on NFSN server or set up FreeBSD VM
- You're okay implementing HyperLogLog yourself or skipping it
- Binary size optimization matters (728 KB vs 2-3 MB)

**Choose Go if:**
- FreeBSD cross-compilation is critical (frequent builds, CI/CD)
- You want mature vendor libraries for User-Agent parsing, bot detection
- You need HyperLogLog with well-tested implementations
- Team familiarity with Go outweighs Janet's terseness
- You prefer traditional imperative style

**Choose GoAccess if:**
- Minimizing custom code is top priority
- You're okay with two-binary pipeline complexity
- Unique visitor merging across time periods is not critical

### Recommendation for This Project

**Use Janet if the author is comfortable with:**
1. Building on NFSN server for production binaries (low friction for hourly cron)
2. Implementing HyperLogLog in Janet (~200 LOC) or accepting hash-based uniques
3. Writing PEG patterns for User-Agent parsing (~80 LOC, fun but requires learning)
4. Lisp syntax and functional programming style

**The prototype demonstrates Janet is viable.** The remaining work (hourly bucketing, JSON output, bot detection) is straightforward and plays to Janet's strengths (PEG, data transformation).

**Key decision point:** Is FreeBSD build friction (Option 1: build on server) acceptable? If yes, Janet is competitive with Go. If no, Go's `GOOS=freebsd` cross-compilation is a strong advantage.

---

## Recommended Design for This Project

### Architecture

```
┌─────────────────┐
│  Access Logs    │  Apache Combined format, rotated daily/weekly
│  /home/logs/    │
└────────┬────────┘
         │
         │ (Hourly cron)
         ▼
┌─────────────────┐
│  analyzelog     │  Go binary, parses logs, updates aggregates
│  (standalone)   │
└────────┬────────┘
         │
         │ (Writes JSON)
         ▼
┌─────────────────┐
│  Analytics DB   │  JSON files in /home/data/analytics/
│  (JSON files)   │  - YYYY-MM.json (monthly aggregates)
└────────┬────────┘
         │
         │ (Git sync via GitHub Actions)
         ▼
┌─────────────────┐
│  Git Repository │  Committed to awstats-data/ (version-controlled)
│  (public/git)   │
└────────┬────────┘
         │
         │ (Optional: HTML report generator)
         ▼
┌─────────────────┐
│  Static Report  │  /home/public/stats/ (like current AWStats HTML)
│  (HTML/CSS)     │
└─────────────────┘
```

### Data Model

**File structure:**
```
awstats-data/analytics/
  2025-10.json    # October 2025 aggregates
  2025-11.json    # November 2025 aggregates
  2025-12.json    # December 2025 aggregates
  2026-01.json    # January 2026 aggregates (current month, frequently updated)
```

**Schema (2025-10.json):**
```json
{
  "month": "2025-10",
  "generated_at": "2025-11-02T01:00:06Z",
  "total_requests": 57,
  "total_visits": 27,
  "unique_visitors_hll": "base64-encoded-hyperloglog-sketch",
  "unique_visitors_estimate": 26,
  "log_files_processed": [
    "/home/logs/access_log.20251022",
    "/home/logs/access_log.20251026"
  ],

  "hourly": [
    {
      "hour": "2025-10-22T21:00:00Z",
      "requests": 46,
      "visits": 25,
      "unique_ips_hll": "...",
      "bytes_sent": 2398092,
      "status_codes": {"200": 40, "404": 5, "301": 1}
    },
    {
      "hour": "2025-10-26T00:00:00Z",
      "requests": 11,
      "visits": 2,
      "unique_ips_hll": "...",
      "bytes_sent": 91996,
      "status_codes": {"200": 11}
    }
  ],

  "paths": [
    {
      "path": "/",
      "requests": 33,
      "visits": 26,
      "bytes_sent": 1471479,
      "avg_response_time_ms": 45
    },
    {
      "path": "/stats/",
      "requests": 13,
      "visits": 1,
      "bytes_sent": 858478
    }
  ],

  "referrers": {
    "direct": {"requests": 40, "visits": 20},
    "search": {"requests": 15, "visits": 5, "sources": {"google": 15}},
    "external": {"requests": 2, "visits": 2}
  },

  "user_agents": {
    "browsers": {
      "Chrome": {"count": 30, "versions": {"132.0": 20, "131.0": 10}},
      "Safari": {"count": 15, "versions": {"18.0": 15}},
      "Firefox": {"count": 10, "versions": {"143.0": 10}}
    },
    "os": {
      "macOS": {"count": 25, "versions": {"15": 25}},
      "Android": {"count": 15, "versions": {"12": 10, "13": 5}},
      "Linux": {"count": 10}
    },
    "bots": {
      "known_bots": {"count": 5, "names": {"Googlebot": 3, "GPTBot": 2}},
      "suspected_bots": {"count": 2, "signals": ["no-referrer-sequential-access"]}
    }
  }
}
```

**Size estimate:** ~5-10 KB per month (compressed: ~1-2 KB)

### Processing Algorithm

**High-level flow:**
```go
func ProcessLogs(month string) (*Analytics, error) {
    // 1. Find all log files for this month (including rotated)
    logFiles := findLogFilesForMonth(month)

    // 2. Initialize aggregates
    agg := NewAnalytics(month)

    // 3. Parse each log file
    for _, logFile := range logFiles {
        for line := range readLogLines(logFile) {
            entry := parseApacheCombinedLog(line)

            // 4. Classify entry
            classification := classifyEntry(entry) // human, bot, suspected-bot

            // 5. Update aggregates
            agg.AddEntry(entry, classification)
        }
    }

    // 6. Finalize (compute derived metrics)
    agg.Finalize()

    return agg, nil
}
```

**Key decisions:**

1. **Stateless reprocessing:** Always reprocess entire month
   - Pro: Handles late-arriving logs, simple implementation
   - Con: ~1-2ms overhead per run (acceptable for hourly cron)

2. **HyperLogLog for unique visitors:**
   - Store serialized HLL sketch per hour
   - Can merge sketches: `unique_visitors_day = merge(hour_00_hll, hour_01_hll, ...)`
   - Library: [github.com/axiomhq/hyperloglog](https://github.com/axiomhq/hyperloglog) (pure Go, 16 KB per sketch)

3. **Bot detection:**
   - Maintain `bots.json` config with regex patterns (update without recompiling)
   - Store both bot counts AND suspected-bot counts
   - Can retroactively improve detection by reprocessing with updated patterns

4. **Top-N tracking:**
   - Keep top 100 paths, top 50 referrers, top 20 browsers
   - Store "other" bucket for long tail
   - Prevents unbounded growth from scraper attacks (random URLs)

### Integration with Existing System

**How it fits:**

1. **Cron job** (`/home/bin/cron-hourly.sh`):
```bash
#!/bin/sh
# Existing: Run buildsite hourly
/home/bin/buildsite -json-url ... -out-dir /home/public

# New: Run analyzelog hourly (processes last 48 hours)
/home/bin/analyzelog -log-dir /home/logs -out-dir /home/data/analytics -lookback 48h
```

2. **GitHub Actions** (`.github/workflows/sync-analytics.yml`):
   - Similar to existing AWStats sync
   - Downloads `awstats-data/analytics/*.json`
   - No privacy filtering needed (already aggregated, no IPs)
   - Creates PR with stats summary

3. **Optional: HTML report generator:**
   - Could extend `generator/internal/report/` package
   - Or standalone template renderer (like current `buildsite`)
   - Outputs `/home/public/stats/index.html` (replaces AWStats HTML)

**Effort estimate:**
- Parser + aggregator: ~500-800 LOC
- Bot detection: ~200 LOC
- HTML report generator: ~400 LOC (can reuse existing report templates)
- Total: ~1200-1400 LOC (comparable to existing `internal/fetch` + `internal/filter`)

## References & Further Reading

**Modern log aggregation patterns:**
- [Log Aggregation Guide 2025](https://www.motadata.com/blog/log-aggregation-guide/) - Overview of architectures
- [Top 9 Log Aggregation Tools (SigNoz)](https://signoz.io/comparisons/log-aggregation-tools/) - Tool comparisons

**Self-hosted analytics:**
- [GoAccess](https://goaccess.io/) - Real-time web log analyzer
- [Plausible Analytics (self-hosted)](https://plausible.io/self-hosted-web-analytics) - Privacy-focused alternative
- [Matomo](https://matomo.org/) - Full-featured web analytics platform

**Bot detection:**
- [matomo-org/device-detector](https://github.com/matomo-org/device-detector) - Comprehensive bot/device detection
- [monperrus/crawler-user-agents](https://github.com/monperrus/crawler-user-agents) - Actively maintained bot regex list

**HyperLogLog implementations:**
- [axiomhq/hyperloglog](https://github.com/axiomhq/hyperloglog) - Fast Go implementation
- [HyperLogLog in Practice](https://research.google/pubs/hyperloglog-in-practice-algorithmic-engineering-of-a-state-of-the-art-cardinality-estimation-algorithm/) - Google's paper on production HLL

**Apache log parsing in Go:**
- [stdlib `net/http/httputil`](https://pkg.go.dev/net/http/httputil) - HTTP parsing utilities
- [satyrius/gonx](https://github.com/satyrius/gonx) - nginx/Apache log parser
- Roll your own with `regexp` (100 LOC, full control)
