# Respectful Upstream Fetching - Implementation Plan

**Date:** 2025-10-20
**Plan:** docs/plans/2025-10-20-012-respectful-upstream-fetching.md
**Priority:** HIGH - Risk of getting blocked
**Status:** 📋 Ready to implement

---

## Critical Insight: Development vs Production

**Problem:** During development/testing, we might run builds 10-20+ times per hour, which looks like abuse to upstream servers.

**Solution:** Different behavior for dev vs prod:
- **Production (hourly cron):** Respectful delays, HTTP caching, retries
- **Development (testing):** Aggressive local caching, minimal upstream requests, clear warnings

---

## Additional Requirements

### Logging During Delays/Retries

**Requirement:** User must know why build is taking time.

**Implementation:**
- Log messages when throttle delays occur: `"Waiting 2s before next request to datos.madrid.es (respectful delay)"`
- Log messages when using cached data: `"Using cached data for URL (age: 5m, saving bandwidth)"`
- Log messages on rate limit: `"Rate limited by upstream (429), will retry in 5m"`
- All logs visible in console output

### Build Report Integration

**Requirement:** Report fetch timing and behavior in build report.

**Add to BuildReport:**
```go
type FetchAttempt struct {
    // ... existing fields ...

    // New fields
    CacheHit      bool          `json:"cache_hit"`
    CacheAge      time.Duration `json:"cache_age_seconds,omitempty"`
    ThrottleDelay time.Duration `json:"throttle_delay_ms,omitempty"`
    RateLimited   bool          `json:"rate_limited"`
}
```

**Display in HTML report:**
- Show cache hit rate per pipeline
- Show total throttle delays
- Show any rate limit incidents
- Color-code: green for cache hits, yellow for delays, red for rate limits

### Justfile Integration

**Requirement:** Default `just` commands should be respectful.

**Update justfile:**
```bash
# Development mode (default for local testing)
dev:
    ./build/buildsite -config config.toml -mode development
    python3 -m http.server 8080 --directory public

# Build site (development mode for testing)
build:
    go build -o build/buildsite ./cmd/buildsite
    ./build/buildsite -config config.toml -mode development

# Production mode (explicit)
build-prod:
    go build -o build/buildsite ./cmd/buildsite
    ./build/buildsite -config config.toml -mode production
```

**Rationale:** Developers run `just dev` and `just build` frequently during development. These should default to development mode to avoid hitting upstream APIs unnecessarily.

---

## Architecture: Request Throttling System

### New Components

1. **RequestThrottle** - Controls timing and rate limiting
2. **HTTPCache** - Persistent cache with smart expiration
3. **ClientMode** - Dev vs Prod behavior
4. **RequestAudit** - Track all HTTP requests

### Component Interactions

```
┌─────────────────────────────────────────────────────────┐
│ Client (fetch request)                                  │
├─────────────────────────────────────────────────────────┤
│ 1. Check ClientMode (dev vs prod)                      │
│ 2. Check HTTPCache (local cache)                       │
│ 3. RequestThrottle (apply delays)                      │
│ 4. Make HTTP request                                   │
│ 5. Detect rate limiting (429/403/503)                  │
│ 6. Update HTTPCache                                    │
│ 7. RequestAudit (log request)                          │
└─────────────────────────────────────────────────────────┘
```

---

## Data Structures

### 1. ClientMode (New)

```go
// internal/fetch/mode.go

package fetch

// ClientMode controls fetch behavior for different environments.
type ClientMode string

const (
    // ProductionMode: Normal operation (hourly cron)
    // - Respectful delays between requests
    // - HTTP caching with If-Modified-Since
    // - Cache TTL: 30 minutes (cache expires if data < 30 min old)
    ProductionMode ClientMode = "production"

    // DevelopmentMode: Testing/debugging (frequent builds)
    // - Aggressive local caching (cache TTL: 1 hour)
    // - WARNING printed to console if making real request
    // - Max 1 request per URL per 5 minutes
    DevelopmentMode ClientMode = "development"
)

// ModeConfig holds mode-specific configuration.
type ModeConfig struct {
    Mode           ClientMode
    CacheTTL       time.Duration  // How long to trust cached data
    MinDelay       time.Duration  // Minimum delay between requests to same host
    MaxRequestRate int            // Max requests per URL per time window
    TimeWindow     time.Duration  // Time window for rate limiting
}

func DefaultProductionConfig() ModeConfig {
    return ModeConfig{
        Mode:           ProductionMode,
        CacheTTL:       30 * time.Minute,
        MinDelay:       2 * time.Second,
        MaxRequestRate: 1,  // 1 request per time window
        TimeWindow:     1 * time.Hour,
    }
}

func DefaultDevelopmentConfig() ModeConfig {
    return ModeConfig{
        Mode:           DevelopmentMode,
        CacheTTL:       1 * time.Hour,
        MinDelay:       5 * time.Second,
        MaxRequestRate: 1,  // 1 request per 5 minutes
        TimeWindow:     5 * time.Minute,
    }
}
```

### 2. HTTPCache (New)

```go
// internal/fetch/cache.go

package fetch

import (
    "crypto/sha256"
    "encoding/json"
    "fmt"
    "os"
    "path/filepath"
    "time"
)

// CacheEntry stores cached HTTP response data.
type CacheEntry struct {
    URL          string    `json:"url"`
    Body         []byte    `json:"body"`
    LastModified string    `json:"last_modified"`
    ETag         string    `json:"etag"`
    FetchedAt    time.Time `json:"fetched_at"`
    StatusCode   int       `json:"status_code"`
}

// HTTPCache manages persistent HTTP response caching.
type HTTPCache struct {
    cacheDir string
    ttl      time.Duration
}

// NewHTTPCache creates a cache with the given directory and TTL.
func NewHTTPCache(cacheDir string, ttl time.Duration) (*HTTPCache, error) {
    if err := os.MkdirAll(cacheDir, 0755); err != nil {
        return nil, fmt.Errorf("creating cache dir: %w", err)
    }
    return &HTTPCache{
        cacheDir: cacheDir,
        ttl:      ttl,
    }, nil
}

// Get retrieves cached entry if valid (not expired).
func (c *HTTPCache) Get(url string) (*CacheEntry, error) {
    path := c.cachePath(url)

    data, err := os.ReadFile(path)
    if err != nil {
        if os.IsNotExist(err) {
            return nil, nil // Cache miss
        }
        return nil, err
    }

    var entry CacheEntry
    if err := json.Unmarshal(data, &entry); err != nil {
        return nil, fmt.Errorf("parsing cache entry: %w", err)
    }

    // Check if expired
    if time.Since(entry.FetchedAt) > c.ttl {
        return nil, nil // Cache expired
    }

    return &entry, nil
}

// Set stores response in cache.
func (c *HTTPCache) Set(entry CacheEntry) error {
    entry.FetchedAt = time.Now()

    data, err := json.MarshalIndent(entry, "", "  ")
    if err != nil {
        return fmt.Errorf("marshaling cache entry: %w", err)
    }

    path := c.cachePath(entry.URL)

    // Atomic write
    tempPath := path + ".tmp"
    if err := os.WriteFile(tempPath, data, 0644); err != nil {
        return fmt.Errorf("writing cache: %w", err)
    }

    if err := os.Rename(tempPath, path); err != nil {
        return fmt.Errorf("renaming cache: %w", err)
    }

    return nil
}

// cachePath generates a safe filename from URL.
func (c *HTTPCache) cachePath(url string) string {
    hash := sha256.Sum256([]byte(url))
    filename := fmt.Sprintf("%x.json", hash[:8]) // First 8 bytes of hash
    return filepath.Join(c.cacheDir, filename)
}
```

### 3. RequestThrottle (New)

```go
// internal/fetch/throttle.go

package fetch

import (
    "fmt"
    "net/url"
    "sync"
    "time"
)

// RequestThrottle manages request rate limiting per host.
type RequestThrottle struct {
    mu            sync.Mutex
    lastRequest   map[string]time.Time  // host -> last request time
    requestCount  map[string]int        // host -> request count in window
    windowStart   map[string]time.Time  // host -> window start time
    config        ModeConfig
}

// NewRequestThrottle creates a throttle with the given config.
func NewRequestThrottle(config ModeConfig) *RequestThrottle {
    return &RequestThrottle{
        lastRequest:  make(map[string]time.Time),
        requestCount: make(map[string]int),
        windowStart:  make(map[string]time.Time),
        config:       config,
    }
}

// Wait blocks until it's safe to make a request to the given URL.
// Returns error if rate limit would be exceeded.
func (t *RequestThrottle) Wait(urlStr string) error {
    t.mu.Lock()
    defer t.mu.Unlock()

    // Extract host from URL
    u, err := url.Parse(urlStr)
    if err != nil {
        return fmt.Errorf("parsing URL: %w", err)
    }
    host := u.Host

    now := time.Now()

    // Check time-based rate limit
    if windowStart, exists := t.windowStart[host]; exists {
        elapsed := now.Sub(windowStart)
        if elapsed < t.config.TimeWindow {
            // Still in window, check count
            count := t.requestCount[host]
            if count >= t.config.MaxRequestRate {
                return fmt.Errorf("rate limit: max %d requests per %v to %s",
                    t.config.MaxRequestRate, t.config.TimeWindow, host)
            }
        } else {
            // Window expired, reset
            t.windowStart[host] = now
            t.requestCount[host] = 0
        }
    } else {
        // First request to this host
        t.windowStart[host] = now
        t.requestCount[host] = 0
    }

    // Apply minimum delay since last request
    if lastReq, exists := t.lastRequest[host]; exists {
        elapsed := now.Sub(lastReq)
        if elapsed < t.config.MinDelay {
            waitTime := t.config.MinDelay - elapsed
            t.mu.Unlock()  // Unlock while sleeping
            time.Sleep(waitTime)
            t.mu.Lock()
            now = time.Now()  // Update time after sleep
        }
    }

    // Record this request
    t.lastRequest[host] = now
    t.requestCount[host]++

    return nil
}
```

### 4. RequestAudit (New)

```go
// internal/fetch/audit.go

package fetch

import (
    "time"
)

// RequestRecord tracks a single HTTP request.
type RequestRecord struct {
    URL            string        `json:"url"`
    Method         string        `json:"method"`
    StatusCode     int           `json:"status_code"`
    Duration       time.Duration `json:"duration_ms"`
    CacheHit       bool          `json:"cache_hit"`
    CacheAge       time.Duration `json:"cache_age_seconds,omitempty"`
    Error          string        `json:"error,omitempty"`
    Timestamp      time.Time     `json:"timestamp"`
    RateLimited    bool          `json:"rate_limited"`
}

// RequestAuditor tracks all HTTP requests.
type RequestAuditor struct {
    records []RequestRecord
}

// NewRequestAuditor creates a new auditor.
func NewRequestAuditor() *RequestAuditor {
    return &RequestAuditor{
        records: make([]RequestRecord, 0),
    }
}

// Record adds a request record.
func (a *RequestAuditor) Record(record RequestRecord) {
    a.records = append(a.records, record)
}

// GetRecords returns all recorded requests.
func (a *RequestAuditor) GetRecords() []RequestRecord {
    return a.records
}
```

### 5. Updated Client (Modified)

```go
// internal/fetch/client.go

type Client struct {
    httpClient *http.Client
    userAgent  string
    cache      *HTTPCache
    throttle   *RequestThrottle
    auditor    *RequestAuditor
    mode       ClientMode
}

// NewClient creates a fetch client with the given mode and config.
func NewClient(timeout time.Duration, mode ClientMode, cacheDir string) (*Client, error) {
    var config ModeConfig
    switch mode {
    case ProductionMode:
        config = DefaultProductionConfig()
    case DevelopmentMode:
        config = DefaultDevelopmentConfig()
    default:
        config = DefaultProductionConfig()
    }

    cache, err := NewHTTPCache(cacheDir, config.CacheTTL)
    if err != nil {
        return nil, fmt.Errorf("creating cache: %w", err)
    }

    return &Client{
        httpClient: &http.Client{Timeout: timeout},
        userAgent:  "plazaespana-info-site-generator/1.0 (https://github.com/ericphanson/plazaespana.info)",
        cache:      cache,
        throttle:   NewRequestThrottle(config),
        auditor:    NewRequestAuditor(),
        mode:       mode,
    }, nil
}
```

---

## Implementation Tasks

### Phase 1: Core Infrastructure (1 hour)

#### Task 1.1: Create ClientMode and Config (15 min)

**File:** `internal/fetch/mode.go` (NEW)

**Code:**
- ClientMode type (Production, Development)
- ModeConfig struct
- DefaultProductionConfig()
- DefaultDevelopmentConfig()

**Tests:** `internal/fetch/mode_test.go`
- Test config defaults
- Test mode string parsing

**Commit:** `feat: add client mode for dev vs prod behavior`

---

#### Task 1.2: Create HTTPCache (20 min)

**File:** `internal/fetch/cache.go` (NEW)

**Code:**
- HTTPCache struct
- CacheEntry struct
- NewHTTPCache()
- Get() - retrieve cached entry
- Set() - store entry
- cachePath() - safe filename from URL

**Tests:** `internal/fetch/cache_test.go`
- Test cache miss
- Test cache hit
- Test cache expiration (TTL)
- Test atomic writes

**Commit:** `feat: add persistent HTTP caching with TTL`

---

#### Task 1.3: Create RequestThrottle (20 min)

**File:** `internal/fetch/throttle.go` (NEW)

**Code:**
- RequestThrottle struct
- NewRequestThrottle()
- Wait() - block until safe to request
- Per-host tracking
- Rate limit enforcement

**Tests:** `internal/fetch/throttle_test.go`
- Test minimum delay enforcement
- Test rate limit (max requests per window)
- Test multiple hosts independently
- Test window reset

**Commit:** `feat: add request throttling with per-host rate limits`

---

#### Task 1.4: Create RequestAuditor (5 min)

**File:** `internal/fetch/audit.go` (NEW)

**Code:**
- RequestRecord struct
- RequestAuditor struct
- Record()
- GetRecords()

**Tests:** None needed (simple data structure)

**Commit:** `feat: add HTTP request auditing`

---

### Phase 2: Client Integration (45 min)

#### Task 2.1: Update Client Constructor (10 min)

**File:** `internal/fetch/client.go` (MODIFY)

**Changes:**
- Add mode parameter to NewClient()
- Initialize cache, throttle, auditor
- Store mode for behavior switches

**Breaking change:** Yes - constructor signature changes

**Migration:**
```go
// Old
client := fetch.NewClient(30 * time.Second)

// New
client, err := fetch.NewClient(30 * time.Second, fetch.ProductionMode, "data/http-cache")
if err != nil {
    // handle error
}
```

**Commit:** `refactor: update Client with mode and dependencies`

---

#### Task 2.2: Update fetch() Method - Part 1: Cache Check (15 min)

**File:** `internal/fetch/client.go` (MODIFY)

**Changes to `fetch()` method:**

```go
func (c *Client) fetch(url string) ([]byte, error) {
    startTime := time.Now()
    var record RequestRecord
    record.URL = url
    record.Method = "GET"
    record.Timestamp = startTime
    defer func() {
        record.Duration = time.Since(startTime)
        c.auditor.Record(record)
    }()

    // Step 1: Check cache first
    cached, err := c.cache.Get(url)
    if err != nil {
        // Cache read error, log but continue
        log.Printf("Warning: cache read error for %s: %v", url, err)
    }

    if cached != nil {
        // Cache hit!
        record.CacheHit = true
        record.CacheAge = time.Since(cached.FetchedAt)
        record.StatusCode = cached.StatusCode

        if c.mode == DevelopmentMode {
            log.Printf("[DEV MODE] Using cached data for %s (age: %v)", url, record.CacheAge)
        }

        return cached.Body, nil
    }

    // Cache miss - will make real request
    record.CacheHit = false

    if c.mode == DevelopmentMode {
        log.Printf("[DEV MODE] ⚠️  MAKING REAL HTTP REQUEST to %s", url)
        log.Printf("[DEV MODE] ⚠️  Consider using cached data or mocks for testing")
    }

    // Step 2: Apply throttling
    if err := c.throttle.Wait(url); err != nil {
        record.Error = err.Error()
        record.RateLimited = true
        return nil, fmt.Errorf("rate limit exceeded: %w", err)
    }

    // ... continue with HTTP request (next task)
}
```

**Commit:** `feat: integrate HTTP caching into fetch method`

---

#### Task 2.3: Update fetch() Method - Part 2: HTTP Request with Rate Limit Detection (15 min)

**File:** `internal/fetch/client.go` (MODIFY)

**Changes to `fetch()` method (continued):**

```go
    // Step 3: Make HTTP request
    req, err := http.NewRequest("GET", url, nil)
    if err != nil {
        record.Error = err.Error()
        return nil, fmt.Errorf("creating request: %w", err)
    }
    req.Header.Set("User-Agent", c.userAgent)

    // Add If-Modified-Since if we have cached data
    if cached != nil && cached.LastModified != "" {
        req.Header.Set("If-Modified-Since", cached.LastModified)
    }
    if cached != nil && cached.ETag != "" {
        req.Header.Set("If-None-Match", cached.ETag)
    }

    resp, err := c.httpClient.Do(req)
    if err != nil {
        record.Error = err.Error()
        return nil, fmt.Errorf("HTTP request failed: %w", err)
    }
    defer resp.Body.Close()

    record.StatusCode = resp.StatusCode

    // Step 4: Handle rate limiting
    if resp.StatusCode == http.StatusTooManyRequests {
        record.RateLimited = true
        retryAfter := resp.Header.Get("Retry-After")
        record.Error = fmt.Sprintf("rate limited (429), retry after: %s", retryAfter)
        return nil, fmt.Errorf("rate limited (429): retry after %s", retryAfter)
    }

    if resp.StatusCode == http.StatusForbidden {
        record.RateLimited = true
        record.Error = "possible rate limit or block (403)"
        return nil, fmt.Errorf("forbidden (403): possible rate limit or block")
    }

    if resp.StatusCode == http.StatusServiceUnavailable {
        record.Error = "service unavailable (503)"
        return nil, fmt.Errorf("service unavailable (503): upstream may be overloaded")
    }

    // Step 5: Handle 304 Not Modified
    if resp.StatusCode == http.StatusNotModified {
        if cached != nil {
            // Use cached data but update cache age
            cached.FetchedAt = time.Now()
            c.cache.Set(*cached)
            return cached.Body, nil
        }
        // Shouldn't happen, but handle gracefully
        return nil, fmt.Errorf("got 304 but no cached data")
    }

    if resp.StatusCode != http.StatusOK {
        record.Error = fmt.Sprintf("HTTP %d", resp.StatusCode)
        return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
    }

    // Step 6: Read response body
    body, err := io.ReadAll(resp.Body)
    if err != nil {
        record.Error = err.Error()
        return nil, fmt.Errorf("reading response: %w", err)
    }

    // Step 7: Update cache
    entry := CacheEntry{
        URL:          url,
        Body:         body,
        LastModified: resp.Header.Get("Last-Modified"),
        ETag:         resp.Header.Get("ETag"),
        StatusCode:   resp.StatusCode,
    }
    if err := c.cache.Set(entry); err != nil {
        log.Printf("Warning: failed to cache response: %v", err)
    }

    return body, nil
}
```

**Commit:** `feat: add rate limit detection and HTTP caching headers`

---

#### Task 2.4: Export Request Audit (5 min)

**File:** `internal/fetch/client.go` (MODIFY)

**Add method:**
```go
// GetRequestAudit returns all recorded HTTP requests.
func (c *Client) GetRequestAudit() []RequestRecord {
    return c.auditor.GetRecords()
}
```

**Commit:** `feat: expose HTTP request audit records`

---

### Phase 3: Pipeline Integration (30 min)

#### Task 3.1: Update Pipeline to Add Delays (15 min)

**File:** `internal/pipeline/pipeline.go` (MODIFY)

**Changes to `FetchAll()` method:**

```go
func (p *Pipeline) FetchAll() PipelineResult {
    var result PipelineResult

    // Fetch JSON (isolated - errors captured, don't crash)
    result.JSONEvents, result.JSONErrors = p.fetchJSONIsolated()

    // Polite delay before next request to same host
    // datos.madrid.es appreciates the breathing room
    time.Sleep(2 * time.Second)

    // Fetch XML (isolated - JSON failure doesn't prevent this)
    result.XMLEvents, result.XMLErrors = p.fetchXMLIsolated()

    // Polite delay before next request to same host
    time.Sleep(2 * time.Second)

    // Fetch CSV (isolated - previous failures don't prevent this)
    result.CSVEvents, result.CSVErrors = p.fetchCSVIsolated()

    return result
}
```

**Note:** Throttle already handles delays in Client, but these are explicit and visible.

**Commit:** `feat: add polite delays between requests to same host`

---

#### Task 3.2: Update main.go to Use Client Mode (15 min)

**File:** `cmd/buildsite/main.go` (MODIFY)

**Changes:**

1. Add mode flag:
```go
mode := flag.String("mode", "production", "Client mode: production or development")
```

2. Parse mode and create client:
```go
var clientMode fetch.ClientMode
switch *mode {
case "development", "dev":
    clientMode = fetch.DevelopmentMode
    log.Println("⚠️  Running in DEVELOPMENT mode (aggressive caching, minimal upstream requests)")
case "production", "prod":
    clientMode = fetch.ProductionMode
default:
    log.Fatalf("Invalid mode: %s (use 'production' or 'development')", *mode)
}

cacheDir := filepath.Join(cfg.Snapshot.DataDir, "http-cache")
client, err := fetch.NewClient(30*time.Second, clientMode, cacheDir)
if err != nil {
    log.Fatalf("Failed to create client: %v", err)
}
```

3. Export request audit at end:
```go
// Export HTTP request audit
requestAudit := client.GetRequestAudit()
log.Printf("\n=== HTTP Request Audit ===")
for _, req := range requestAudit {
    status := fmt.Sprintf("HTTP %d", req.StatusCode)
    if req.CacheHit {
        status = fmt.Sprintf("CACHE HIT (age: %v)", req.CacheAge)
    } else if req.RateLimited {
        status = "RATE LIMITED"
    } else if req.Error != "" {
        status = "ERROR"
    }
    log.Printf("%-60s %s (%.2fs)", req.URL, status, req.Duration.Seconds())
}
```

**Commit:** `feat: add mode flag and HTTP request audit to build`

---

### Phase 4: Configuration & Documentation (30 min)

#### Task 4.1: Update config.toml (5 min)

**File:** `config.toml` (MODIFY)

**Add:**
```toml
[fetch]
# Client mode: "production" or "development"
# Production: Respectful delays, HTTP caching, for hourly cron
# Development: Aggressive caching, warnings on real requests
mode = "production"

# HTTP cache directory (relative to data_dir)
cache_dir = "http-cache"
```

**Commit:** `config: add fetch mode configuration`

---

#### Task 4.2: Update .gitignore (2 min)

**File:** `.gitignore` (MODIFY)

**Add:**
```
# HTTP cache (generated data)
data/http-cache/
```

**Commit:** `chore: ignore HTTP cache directory`

---

#### Task 4.3: Update CLAUDE.md (10 min)

**File:** `CLAUDE.md` (MODIFY)

**Add section after "Robustness Strategy":**

```markdown
### Respectful Upstream Fetching

**Critical:** We fetch from public APIs and must be respectful.

**Production Mode (hourly cron):**
- 2-second delays between requests to same host
- HTTP caching with If-Modified-Since (90% bandwidth reduction)
- Rate limit detection (429/403/503)
- Max 1 request per URL per hour

**Development Mode (testing):**
- Aggressive local caching (1-hour TTL)
- Warnings printed on real HTTP requests
- Max 1 request per URL per 5 minutes
- Use cached data or mocks for frequent testing

**Usage:**
```bash
# Production (default)
./buildsite -config config.toml

# Development (testing)
./buildsite -config config.toml -mode development
```

**Cache location:** `data/http-cache/` (gitignored)

**Request audit:** All HTTP requests logged in build output with timing, cache hits, and rate limit status.
```

**Commit:** `docs: document respectful fetching practices`

---

#### Task 4.4: Update README.md (3 min)

**File:** `README.md` (MODIFY)

**Add to quickstart:**
```markdown
## Development Mode

When testing locally (to avoid hitting upstream APIs repeatedly):

```bash
just dev -mode development
```

This enables aggressive HTTP caching and warns before making real requests.
```

**Commit:** `docs: add development mode to README`

---

#### Task 4.5: Create Testing Guide (10 min)

**File:** `docs/testing-without-upstream.md` (NEW)

```markdown
# Testing Without Hitting Upstream APIs

## Problem

During development, you might run builds 10-20+ times per hour. This looks like abuse to upstream servers and could get us blocked.

## Solutions

### 1. Use Development Mode (Recommended)

```bash
./buildsite -config config.toml -mode development
```

**What it does:**
- Aggressive HTTP caching (1-hour TTL)
- Max 1 request per URL per 5 minutes
- Prints warnings before making real requests
- Uses cached data whenever possible

**First run:** Makes real requests, caches responses
**Subsequent runs:** Uses cached data (no upstream requests)

### 2. Use file:// URLs (Best for Tests)

```bash
./buildsite \
  -json-url file:///workspace/testdata/fixtures/madrid-events.json \
  -xml-url file:///workspace/testdata/fixtures/madrid-events.xml \
  -csv-url file:///workspace/testdata/fixtures/madrid-events.csv \
  -esmadrid-url file:///workspace/testdata/fixtures/esmadrid.xml \
  -out-dir ./public \
  -data-dir ./data
```

**What it does:**
- No HTTP requests at all
- Instant builds
- Deterministic results

### 3. Pre-cache Data

Run once in production mode to populate cache:
```bash
./buildsite -config config.toml -mode production
```

Then switch to development mode:
```bash
./buildsite -config config.toml -mode development
```

Cache persists in `data/http-cache/` until expired.

### 4. Clear Cache When Needed

```bash
rm -rf data/http-cache
```

Forces fresh fetch on next build.

## HTTP Request Audit

Every build shows request audit:

```
=== HTTP Request Audit ===
https://datos.madrid.es/.../eventos.json    CACHE HIT (age: 5m)    (0.00s)
https://datos.madrid.es/.../eventos.xml     CACHE HIT (age: 5m)    (0.00s)
https://datos.madrid.es/.../eventos.csv     CACHE HIT (age: 5m)    (0.00s)
https://www.esmadrid.com/.../agenda.xml     HTTP 200               (1.23s)
```

**CACHE HIT:** Used local cache (no upstream request)
**HTTP 200:** Made real request
**RATE LIMITED:** Hit rate limit (too many requests)

## Best Practices

1. **Default to development mode** when testing
2. **Use file:// URLs** for unit tests
3. **Run production mode sparingly** (only when needed)
4. **Check request audit** to verify cache usage
5. **Clear cache** if testing cache behavior
```

**Commit:** `docs: add guide for testing without hitting upstream`

---

### Phase 5: Testing & Validation (1 hour)

#### Task 5.1: Write Unit Tests (30 min)

**Files:**
- `internal/fetch/cache_test.go` - Test cache behavior
- `internal/fetch/throttle_test.go` - Test rate limiting
- `internal/fetch/mode_test.go` - Test mode configs
- `internal/fetch/client_test.go` - Update existing tests for new constructor

**Test scenarios:**
- Cache hit/miss
- Cache expiration
- Rate limit enforcement
- Minimum delay between requests
- Mode-specific behavior
- 429/403/503 detection

**Commit:** `test: add comprehensive tests for fetch improvements`

---

#### Task 5.2: Integration Test - Production Mode (15 min)

**Manual test:**

```bash
# Clear cache
rm -rf data/http-cache

# First run (will make real requests)
./build/buildsite -config config.toml -mode production

# Check request audit - should show HTTP 200 for all

# Second run within 30 minutes (should use cache)
./build/buildsite -config config.toml -mode production

# Check request audit - should show CACHE HIT for all
```

**Verify:**
- ✅ Delays between requests (check timing in logs)
- ✅ Cache populated in `data/http-cache/`
- ✅ Second run uses cache
- ✅ Request audit shows cache hits
- ✅ Build completes successfully

---

#### Task 5.3: Integration Test - Development Mode (10 min)

**Manual test:**

```bash
# Run in dev mode
./build/buildsite -config config.toml -mode development

# Should see warnings:
# [DEV MODE] ⚠️  MAKING REAL HTTP REQUEST to ...

# Run again immediately
./build/buildsite -config config.toml -mode development

# Should use cache, no warnings

# Try to run 3+ times quickly
./build/buildsite -config config.toml -mode development
./build/buildsite -config config.toml -mode development
./build/buildsite -config config.toml -mode development

# Should hit rate limit on subsequent runs
```

**Verify:**
- ✅ First run warns about real requests
- ✅ Subsequent runs use cache
- ✅ Rate limit enforced (1 request per 5 min)
- ✅ Clear error message when rate limited

---

#### Task 5.4: Test Rate Limit Detection (5 min)

**Mock test:** Create test that simulates 429 response

```go
func TestRateLimitDetection(t *testing.T) {
    // Mock HTTP server that returns 429
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Retry-After", "300")
        w.WriteHeader(http.StatusTooManyRequests)
    }))
    defer server.Close()

    // Fetch should detect and report rate limit
    client, _ := fetch.NewClient(5*time.Second, fetch.ProductionMode, t.TempDir())
    _, err := client.FetchJSON(server.URL, time.UTC)

    if err == nil || !strings.Contains(err.Error(), "rate limited") {
        t.Errorf("Expected rate limit error, got: %v", err)
    }
}
```

**Commit:** `test: add integration tests for fetch improvements`

---

### Phase 6: Deployment Prep (15 min)

#### Task 6.1: Update Deployment Docs (10 min)

**File:** `ops/deploy-notes.md` (MODIFY)

**Add warning:**
```markdown
## Important: Respectful Fetching

The binary now includes HTTP caching and rate limiting. By default, it runs in **production mode** which is appropriate for hourly cron jobs.

**Do NOT run in development mode on the server** - that's only for local testing.

**Cron command remains the same:**
```bash
/home/bin/buildsite -config /home/config.toml
```

(Mode defaults to production)

**Cache location:** `/home/data/http-cache/` (automatically created)
```

**Commit:** `docs: update deployment notes for fetch improvements`

---

#### Task 6.2: FreeBSD Build Test (5 min)

```bash
just freebsd
file build/buildsite

# Verify static binary
ldd build/buildsite  # Should say "not a dynamic executable"
```

**Commit:** `build: verify FreeBSD binary with new fetch code`

---

## Implementation Schedule

**Total time:** ~3.5 hours

| Phase | Time | Tasks |
|-------|------|-------|
| Phase 1: Core Infrastructure | 1h | 4 tasks (mode, cache, throttle, audit) |
| Phase 2: Client Integration | 45m | 4 tasks (constructor, fetch methods) |
| Phase 3: Pipeline Integration | 30m | 2 tasks (delays, main.go) |
| Phase 4: Configuration & Docs | 30m | 5 tasks (config, docs, guides) |
| Phase 5: Testing & Validation | 1h | 4 tasks (unit, integration, manual) |
| Phase 6: Deployment Prep | 15m | 2 tasks (deploy docs, build) |

---

## Success Criteria

### Functional Requirements

✅ **Production mode works:**
- 2-second delays between requests
- HTTP caching with If-Modified-Since
- Cache hits logged in audit
- Rate limit detection (429/403/503)
- No breaking of existing builds

✅ **Development mode works:**
- Aggressive caching (1-hour TTL)
- Warnings on real HTTP requests
- Rate limit enforced (1 req per 5 min)
- Can run tests without hitting upstream

✅ **Request audit:**
- All HTTP requests logged
- Shows cache hits, timing, errors
- Rate limit status visible

✅ **Cache management:**
- Persistent across builds
- Automatic expiration (TTL)
- Atomic writes (temp + rename)
- Gitignored directory

### Non-Functional Requirements

✅ **Performance:**
- Production build time: 8-12 seconds (acceptable for hourly cron)
- Development build time: < 2 seconds (with cache hits)
- Cache reduces bandwidth by 90%

✅ **Reliability:**
- All existing tests pass
- New tests for cache, throttle, mode
- Graceful handling of cache errors
- Falls back to real requests if cache fails

✅ **Maintainability:**
- Clear separation of concerns (cache, throttle, audit)
- Well-documented behavior
- Easy to understand request flow
- Testing guide for developers

---

## Rollout Plan

### Step 1: Implement in Feature Branch (Day 1)
- Complete all tasks
- Run all tests
- Verify both modes work

### Step 2: Test Locally (Day 1)
- Run multiple builds in dev mode
- Verify cache behavior
- Check rate limiting works
- Review request audit

### Step 3: Deploy to Production (Day 2)
- Merge to main
- Build FreeBSD binary
- Deploy via SFTP
- Monitor first few cron runs

### Step 4: Monitor (Week 1)
- Check for rate limit errors in logs
- Verify cache is working
- Monitor bandwidth usage
- Adjust TTL if needed

---

## Rollback Plan

If issues arise:

1. **Quick rollback:**
   - Revert to previous binary on server
   - Delete `data/http-cache` directory

2. **Partial rollback:**
   - Set cache TTL to 0 (disables caching)
   - Keep rate limit detection

3. **Emergency:**
   - Revert entire feature branch
   - Redeploy previous version

---

## Future Enhancements (Not in Scope)

- Exponential backoff with retries
- Respect Retry-After header
- Compression support (gzip, brotli)
- Conditional requests with ETag
- Request jitter (prevent thundering herd)
- Circuit breaker pattern
- Prometheus metrics for monitoring

---

## Questions for Review

1. **Is 2-second delay sufficient?** Could increase to 3-5 seconds if needed
2. **Is 30-minute cache TTL appropriate?** Could increase to 1 hour for production
3. **Should we add retry logic?** Not in this iteration, but could add later
4. **Do we need compression?** Responses are ~1MB, probably fine without
5. **Should development mode be even more aggressive?** Could increase to 2-hour TTL

