# Respectful Upstream Fetching Implementation

**Date:** 2025-10-20
**Plan:** docs/plans/2025-10-20-012-respectful-upstream-fetching-implementation.md
**Status:** In Progress

## Objective

Implement comprehensive respectful fetching system to prevent API abuse during both production (hourly cron) and development (frequent testing). Key features: HTTP caching, rate limiting, dual modes, request auditing, and clear logging.

## Implementation Log

---

### Setup

**Status:** ✅ Complete
**Time:** 2025-10-20

**Actions:**
- Updated plan with logging requirements
- Updated plan with build report integration
- Updated plan with justfile integration
- Created implementation log

**Commits:**
- `bb6bbf1` - docs: update plan with logging and justfile requirements

---

## Phase 1: Core Infrastructure

### Task 1.1: Create ClientMode and Config

**Status:** ✅ Complete
**Time:** 2025-10-20

**Files Created:**
- `internal/fetch/mode.go` - ClientMode types and configs
- `internal/fetch/mode_test.go` - Tests for mode functionality

**Implementation:**
- ClientMode type (ProductionMode, DevelopmentMode)
- ModeConfig struct with TTL, delays, rate limits
- DefaultProductionConfig() - 30min TTL, 2s delays, 1 req/hour
- DefaultDevelopmentConfig() - 1hour TTL, 5s delays, 1 req/5min
- ParseMode() - String to ClientMode conversion

**Tests:** 3 tests, all passing
- TestDefaultProductionConfig
- TestDefaultDevelopmentConfig
- TestParseMode

**Result:** Mode configuration system ready

---

### Task 1.2: Create HTTPCache

**Status:** ✅ Complete
**Time:** 2025-10-20

**Files Created:**
- `internal/fetch/cache.go` - Persistent HTTP response caching
- `internal/fetch/cache_test.go` - Cache tests

**Implementation:**
- CacheEntry struct with URL, Body, LastModified, ETag, FetchedAt, StatusCode
- HTTPCache struct with cacheDir and ttl
- NewHTTPCache() - Creates cache directory (os.MkdirAll)
- Get() - Retrieves cached entry if not expired (TTL check)
- Set() - Stores entry with atomic write (temp file + rename)
- cachePath() - SHA256-based safe filename generation (first 8 bytes)

**Features:**
- TTL-based expiration (configurable per mode)
- Atomic writes prevent partial cache corruption
- SHA256 hash ensures safe filenames from arbitrary URLs
- Cache miss returns nil (not error)
- Expired entries treated as cache miss

**Tests:** 5 tests, all passing
- TestHTTPCache_Miss - Verify cache miss behavior
- TestHTTPCache_HitAndExpiration - Verify cache hit and TTL expiration (100ms TTL)
- TestHTTPCache_AtomicWrite - Verify temp file cleanup
- TestHTTPCache_MultipleURLs - Verify independent cache entries
- TestHTTPCache_ETag - Verify ETag storage

**Result:** HTTP caching system ready for integration

---

### Task 1.3: Create RequestThrottle

**Status:** ✅ Complete
**Time:** 2025-10-20

**Files Created:**
- `internal/fetch/throttle.go` - Per-host rate limiting
- `internal/fetch/throttle_test.go` - Throttle tests

**Implementation:**
- RequestThrottle struct with minDelay, lastReq map, mutex
- NewRequestThrottle() - Creates throttle with configurable delay
- Wait() - Blocks until enough time has passed since last request to same host
  - Tracks last request time per host
  - Returns actual delay waited (for logging)
  - Thread-safe with mutex

**Features:**
- Per-host tracking (independent delays for different hosts)
- Calculates exact wait time needed
- First request to host has no delay
- Thread-safe for concurrent use
- Returns error for invalid URLs

**Tests:** 5 tests, all passing
- TestRequestThrottle_FirstRequest - No delay on first request
- TestRequestThrottle_SubsequentRequest - Enforces 100ms delay
- TestRequestThrottle_DifferentHosts - Independent tracking per host
- TestRequestThrottle_DelayExpired - No delay after minDelay expires
- TestRequestThrottle_InvalidURL - Error handling

**Result:** Request throttling ready for integration

---

### Task 1.4: Create RequestAudit

**Status:** ✅ Complete
**Time:** 2025-10-20

**Files Created:**
- `internal/fetch/audit.go` - Request audit trail
- `internal/fetch/audit_test.go` - Audit tests

**Implementation:**
- RequestRecord struct with URL, Timestamp, CacheHit, StatusCode, DelayMs, RateLimited, Error
- RequestAuditor struct with records slice and mutex
- NewRequestAuditor() - Creates empty auditor
- Record() - Adds request to audit trail (thread-safe)
- Export() - Writes audit trail to JSON file
- Records() - Returns copy of all records (thread-safe)

**Features:**
- Thread-safe concurrent access with mutex
- JSON export for build reports
- Captures cache hits, delays, rate limits, errors
- DelayMs stored as int64 (milliseconds) for JSON compatibility

**Tests:** 4 tests, all passing
- TestRequestAuditor_Record - Add multiple records
- TestRequestAuditor_Export - Write and read JSON file
- TestRequestAuditor_ConcurrentAccess - Concurrent writes from 10 goroutines
- TestRequestAuditor_ErrorRecord - Error field storage

**Bug Fixed:**
- Records() method was doubling records (used `append` instead of `copy`)
- Fixed by using built-in `copy()` function properly

**Result:** Request audit trail ready for integration

---

## Phase 1: Complete ✅

**Summary:**
- 4 tasks completed
- 4 new modules created (mode, cache, throttle, audit)
- 17 tests, all passing (3 + 5 + 5 + 4)
- Core infrastructure ready for client integration

**Modules:**
1. ClientMode - Production/Development configurations
2. HTTPCache - Persistent HTTP caching with TTL
3. RequestThrottle - Per-host rate limiting
4. RequestAudit - Request tracking and export

---

## Phase 2: Client Integration

### Task 2.1: Update Client Constructor

**Status:** ✅ Complete
**Time:** 2025-10-20

**Files Modified:**
- `internal/fetch/client.go` - Client struct and constructor
- `internal/fetch/client_test.go` - Updated test calls
- `internal/fetch/csv_test.go` - Updated test calls
- `internal/fetch/json_test.go` - Updated test calls
- `internal/fetch/xml_test.go` - Updated test calls

**Changes:**
- Added new fields to Client struct: cache, throttle, auditor, config
- Updated NewClient signature: `func NewClient(timeout time.Duration, config ModeConfig, cacheDir string) (*Client, error)`
- NewClient now returns error (cache creation can fail)
- Added Auditor() method to access request auditor
- Updated all 11 test files with new NewClient signature

**Implementation Details:**
- Client now initializes HTTPCache with configurable TTL
- Client now initializes RequestThrottle with configurable min delay
- Client now initializes RequestAuditor for tracking requests
- All test files use DefaultDevelopmentConfig() for testing
- All tests use t.TempDir() for isolated cache directories

**Tests:** All fetch package tests passing (31 tests)

**Result:** Client constructor updated and ready for cache/throttle integration

---

### Task 2.2 & 2.3: Update fetch() Method (Combined)

**Status:** ✅ Complete
**Time:** 2025-10-20

**Files Modified:**
- `internal/fetch/client.go` - fetch() method

**Implementation:** Comprehensive respectful fetching in fetch() method:

1. **Cache Check** (before HTTP request):
   - Check cache.Get(url) first
   - If cached and not expired, return immediately (cache hit)
   - Record audit event for cache hit

2. **Throttling**:
   - Call throttle.Wait(url) before HTTP request
   - Enforces per-host minimum delay
   - Logs delay to stderr so user knows why build is slow
   - Format: `[mode] Waiting Xms before fetching URL`

3. **If-Modified-Since Header**:
   - Add header if we have cached data (even if expired)
   - Allows server to return 304 Not Modified

4. **304 Not Modified Handling**:
   - Use cached body if server returns 304
   - Record as cache hit in audit

5. **Rate Limit Detection**:
   - Detect 429 (Too Many Requests)
   - Detect 403 (Forbidden)
   - Detect 503 (Service Unavailable)
   - Mark as RateLimited in audit trail
   - Return clear error message

6. **Cache Storage**:
   - Store successful responses (200 OK)
   - Capture Last-Modified and ETag headers
   - Log warning if cache write fails (don't fail request)

7. **Request Auditing**:
   - Record all fetch attempts
   - Track: URL, timestamp, cache hit, status code, delay, errors
   - Mark rate-limited requests

**Features:**
- Cache hit path: No HTTP request, instant return
- 304 path: HTTP request but no body transfer (saves bandwidth)
- Throttle delays logged to stderr
- Rate limit errors clearly identified
- All requests audited for build reports

**Tests:** All 31 fetch package tests passing

**Result:** fetch() method fully implements respectful fetching

---

### Task 2.4: Export Request Audit

**Status:** ✅ Complete
**Time:** 2025-10-20

**Files Modified:**
- `cmd/buildsite/main.go` - Add mode flag, update client creation, export audit
- `internal/pipeline/pipeline_test.go` - Update NewClient calls

**Changes:**

1. **Added `-fetch-mode` flag**:
   - Default: "development"
   - Values: "production" or "development"
   - Affects caching TTL and throttling delays

2. **Updated client creation** (line 144-161):
   - Parse fetch mode from flag
   - Get appropriate ModeConfig
   - Create cache directory: `{dataDir}/http-cache`
   - Pass config and cache dir to NewClient
   - Log fetch mode settings on startup

3. **Added audit export** (line 692-698):
   - Export to `{dataDir}/request-audit.json`
   - Log success/warning message
   - Runs after all pipeline work completes

4. **Fixed test files**:
   - Updated pipeline_test.go with new NewClient signature

**Output:**
- Console log: `Fetch mode: development (cache TTL: 1h, min delay: 5s)`
- Audit file: `data/request-audit.json` with all HTTP request details

**Tests:** All tests passing (including pipeline tests)

**Result:** Audit trail export integrated, Phase 2 complete

---

## Phase 2: Complete ✅

**Summary:**
- 3 tasks completed (Task 2.2 & 2.3 combined)
- Client fully integrated with cache, throttle, and auditor
- fetch() method implements comprehensive respectful fetching
- Audit trail exported to data directory
- All tests passing (38 total)

**Key Features:**
- HTTP caching with If-Modified-Since headers
- Per-host throttling with user-visible delays
- Rate limit detection (429/403/503)
- Request auditing for all fetches
- Mode-based configuration (production/development)

---

## Phase 3: Pipeline Integration

### Task 3.1 & 3.2: Pipeline Delays and Logging (Combined)

**Status:** ✅ Complete
**Time:** 2025-10-20

**Files Modified:**
- `internal/fetch/client.go` - Add Config() method
- `internal/pipeline/pipeline.go` - Add delays and logging

**Changes:**

1. **Added Client.Config() method**:
   - Exposes client's ModeConfig for pipeline to use
   - Allows pipeline to access MinDelay and other settings

2. **Updated Pipeline struct**:
   - Added minDelay field (from client config)
   - Added fetchMode field (for logging)
   - NewPipeline extracts config from client

3. **Updated FetchAll() with delays**:
   - Explicit time.Sleep(minDelay) between format fetches
   - Sleep after JSON, sleep after XML
   - Prevents overwhelming upstream with rapid successive requests

4. **Added comprehensive pipeline logging**:
   - Log before each format fetch: "[Pipeline] Fetching JSON..."
   - Log results: "[Pipeline] JSON: X events, Y errors"
   - Log delays: "[Pipeline] Waiting 5s before fetching next format (respectful delay)..."
   - User always knows what's happening and why builds take time

**Behavior:**
- Development mode: 5s delays between JSON→XML→CSV
- Production mode: 2s delays between JSON→XML→CSV
- Both modes: Double protection (pipeline sleep + fetch throttle)

**Example output:**
```
[Pipeline] Fetching JSON from datos.madrid.es...
[Pipeline] JSON: 1055 events, 0 errors
[Pipeline] Waiting 5s before fetching next format (respectful delay)...
[Pipeline] Fetching XML from datos.madrid.es...
[Pipeline] XML: 1001 events, 0 errors
[Pipeline] Waiting 5s before fetching next format (respectful delay)...
[Pipeline] Fetching CSV from datos.madrid.es...
[Pipeline] CSV: 1055 events, 0 errors
```

**Tests:** All pipeline tests passing (test duration: 61s due to delays)

**Result:** Pipeline implements respectful sequential fetching with clear logging

---

## Phase 3: Complete ✅

**Summary:**
- 2 tasks completed (combined into one commit)
- Pipeline now has explicit delays between format fetches
- Comprehensive logging shows progress and delays
- All tests passing

**Key Features:**
- Explicit time.Sleep() between formats
- Clear [Pipeline] logging prefix
- User always knows why build is delayed
- Double protection: pipeline sleep + fetch throttle

---

## Phase 4: Configuration & Documentation

### Task 4.1-4.3: Update Config and Documentation

**Status:** ✅ Complete
**Time:** 2025-10-20

**Files Modified:**
- `config.toml` - Add [fetch] section
- `.gitignore` - Update data/ comment
- `CLAUDE.md` - Add "Respectful Upstream Fetching" section

**Changes:**

1. **config.toml**:
   - Added [fetch] section with mode, cache_dir, audit_path
   - Documented both development and production mode settings
   - Clear comments explaining cache TTL, delays, and rate limits

2. **.gitignore**:
   - Updated data/ comment to clarify it includes http-cache and request-audit.json
   - No new ignores needed (data/ already covers everything)

3. **CLAUDE.md**:
   - Added comprehensive "Respectful Upstream Fetching" section (67 lines)
   - Explains problem (development testing looks like attack)
   - Documents development vs production modes
   - Lists all implementation files (mode, cache, throttle, audit)
   - Details HTTP caching, throttling, rate limit detection, auditing
   - Shows configuration and flag usage
   - Result: Safe to run `just dev` 20+ times without risk

**Result:** Complete documentation of respectful fetching system

---

### Task 4.4: Update README.md

**Status:** ✅ Complete
**Time:** 2025-10-20

**Files Modified:**
- `README.md` - Add "Respectful Upstream Fetching" section

**Changes:**
- Added new section (61 lines) after "How It Works"
- Explains problem, solution, features, and usage
- Shows development vs production mode differences
- Provides usage examples with `just dev` and `-fetch-mode` flag
- Includes config.toml snippet
- Result statement: "Safe to run `just dev` 20+ times during testing"

**Content:**
- Problem: Dev testing looks like attack without throttling
- Solution: Dual-mode system (development/production)
- Features: HTTP caching, throttling, rate detection, auditing
- Usage examples for both modes
- Configuration snippet

**Result:** User-facing documentation complete

---

## Phase 4: Complete ✅

**Summary:**
- 4 tasks completed (combined into 2 commits)
- config.toml: Added [fetch] section with full documentation
- .gitignore: Clarified data/ covers http-cache
- CLAUDE.md: Added 67-line "Respectful Upstream Fetching" section
- README.md: Added 61-line user-facing documentation

**Documentation Coverage:**
- Technical details (CLAUDE.md) for developers
- User-facing guide (README.md) for all users
- Configuration reference (config.toml) with comments
- All aspects of respectful fetching documented

---
