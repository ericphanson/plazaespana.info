# Respectful Upstream Data Fetching Plan

**Date:** 2025-10-20
**Priority:** HIGH
**Status:** 📋 Planning

---

## Problem Statement

We're making 4 sequential HTTP requests every hour:
- 3 requests to datos.madrid.es (JSON, XML, CSV - same dataset, different formats)
- 1 request to esmadrid.com (XML)

**Current approach:**
- ✅ User-Agent header set with project info
- ✅ 30-second timeout
- ✅ Sequential fetching (not parallel)
- ❌ No delays between requests
- ❌ No HTTP caching (If-Modified-Since, ETag)
- ❌ No exponential backoff on failures
- ❌ No rate limit detection (429 status)
- ❌ No request throttling

**Concern:** We may have been rate-limited or blocked by esmadrid.com.

---

## Current Implementation Analysis

### What We're Doing Right ✅

1. **User-Agent Header** (line 294 in `internal/fetch/client.go`):
   ```go
   userAgent: "madrid-events-site-generator/1.0 (https://github.com/ericphanson/madrid-events)"
   ```
   - Identifies our project
   - Provides contact info
   - Professional and respectful

2. **Timeout** (30 seconds):
   - Prevents hanging connections
   - Reasonable timeout value

3. **Sequential Fetching**:
   - Not hammering servers with parallel requests
   - Good practice for multiple requests to same host

4. **Graceful Failure Handling**:
   - Falls back to snapshot if all sources fail
   - Doesn't crash on individual source failures

5. **Hourly Schedule**:
   - Not overly aggressive
   - Reasonable for event data that changes slowly

### What We're Missing ❌

1. **No HTTP Caching**:
   - Not checking `If-Modified-Since` or `ETag` headers
   - Re-downloading full datasets even if unchanged
   - Wasting bandwidth for both sides

2. **No Delays Between Requests**:
   - 3 requests to datos.madrid.es happen back-to-back
   - No breathing room for upstream servers

3. **No Rate Limit Detection**:
   - Don't check for HTTP 429 (Too Many Requests)
   - Don't detect 403/503 as potential rate limiting
   - No backoff strategy when detected

4. **No Exponential Backoff**:
   - Single retry or immediate failure
   - No increasing delays on repeated failures

5. **No Request Throttling**:
   - Could add artificial delay between requests
   - Especially for multiple requests to same host

---

## Recommended Improvements

### Priority 1: IMMEDIATE (Required)

#### 1.1 Add Delays Between Requests to Same Host

**Why:** Prevent looking like a bot/scraper

**Implementation:**
```go
// internal/pipeline/pipeline.go

func (p *Pipeline) FetchAll() PipelineResult {
    var result PipelineResult

    // Fetch JSON
    result.JSONEvents, result.JSONErrors = p.fetchJSONIsolated()

    // Polite delay before next request to same host
    time.Sleep(2 * time.Second)

    // Fetch XML
    result.XMLEvents, result.XMLErrors = p.fetchXMLIsolated()

    // Polite delay before next request to same host
    time.Sleep(2 * time.Second)

    // Fetch CSV
    result.CSVEvents, result.CSVErrors = p.fetchCSVIsolated()

    return result
}
```

**Impact:** Adds 4 seconds to build time (acceptable for hourly cron)

#### 1.2 Detect Rate Limiting (429, 403, 503)

**Why:** Know when we're being throttled

**Implementation:**
```go
// internal/fetch/client.go

func (c *Client) fetch(url string) ([]byte, error) {
    // ... existing code ...

    // Check for rate limiting signals
    if resp.StatusCode == http.StatusTooManyRequests {
        retryAfter := resp.Header.Get("Retry-After")
        return nil, fmt.Errorf("rate limited (429): retry after %s", retryAfter)
    }

    if resp.StatusCode == http.StatusForbidden {
        return nil, fmt.Errorf("forbidden (403): possible rate limit or block")
    }

    if resp.StatusCode == http.StatusServiceUnavailable {
        return nil, fmt.Errorf("service unavailable (503): upstream may be overloaded")
    }

    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
    }

    // ... rest of code ...
}
```

#### 1.3 Add HTTP Caching (If-Modified-Since)

**Why:** Massively reduce bandwidth if data hasn't changed

**Implementation:**
```go
// Save Last-Modified header from previous fetch
// Check If-Modified-Since on next fetch
// Handle 304 Not Modified response

type Client struct {
    httpClient     *http.Client
    userAgent      string
    lastModified   map[string]string  // URL -> Last-Modified header
}

func (c *Client) fetch(url string) ([]byte, error) {
    req, err := http.NewRequest("GET", url, nil)
    if err != nil {
        return nil, fmt.Errorf("creating request: %w", err)
    }
    req.Header.Set("User-Agent", c.userAgent)

    // Add If-Modified-Since if we have it
    if lastMod, ok := c.lastModified[url]; ok {
        req.Header.Set("If-Modified-Since", lastMod)
    }

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return nil, fmt.Errorf("HTTP request failed: %w", err)
    }
    defer resp.Body.Close()

    // Handle 304 Not Modified
    if resp.StatusCode == http.StatusNotModified {
        return nil, fmt.Errorf("not modified (304): use cached data")
    }

    if resp.StatusCode == http.StatusOK {
        // Save Last-Modified for next time
        if lastMod := resp.Header.Get("Last-Modified"); lastMod != "" {
            c.lastModified[url] = lastMod
        }
    }

    // ... rest of code ...
}
```

**Note:** Requires persistent storage (file) for Last-Modified headers

### Priority 2: RECOMMENDED (Nice to Have)

#### 2.1 Exponential Backoff on Failures

**Why:** Gracefully handle temporary failures

**Implementation:**
```go
func (c *Client) fetchWithRetry(url string, maxRetries int) ([]byte, error) {
    var lastErr error

    for attempt := 0; attempt <= maxRetries; attempt++ {
        if attempt > 0 {
            // Exponential backoff: 2^attempt seconds
            delay := time.Duration(1<<uint(attempt)) * time.Second
            time.Sleep(delay)
        }

        body, err := c.fetch(url)
        if err == nil {
            return body, nil
        }

        lastErr = err

        // Don't retry on certain errors (404, 403, etc.)
        if shouldNotRetry(err) {
            break
        }
    }

    return nil, lastErr
}
```

#### 2.2 Respect Retry-After Header

**Why:** Server tells us exactly when to retry

**Implementation:**
```go
if resp.StatusCode == http.StatusTooManyRequests {
    retryAfter := resp.Header.Get("Retry-After")
    if retryAfter != "" {
        // Parse Retry-After (either seconds or HTTP date)
        if delay, err := parseRetryAfter(retryAfter); err == nil {
            time.Sleep(delay)
            // Retry the request
        }
    }
}
```

#### 2.3 Add Jitter to Requests

**Why:** Avoid synchronized thundering herd if multiple instances run

**Implementation:**
```go
// Add random jitter to delays (±500ms)
jitter := time.Duration(rand.Intn(1000)-500) * time.Millisecond
time.Sleep(2*time.Second + jitter)
```

### Priority 3: OPTIONAL (Future)

- **Request tracking**: Log all requests with timing/status to audit trail
- **Backoff multiplier config**: Make delays configurable
- **Circuit breaker**: Stop requesting after repeated failures
- **robots.txt checking**: Parse and respect robots.txt (if they have one)

---

## Implementation Tasks

### Task 1: Add Request Delays (15 min)
- Update `internal/pipeline/pipeline.go` FetchAll()
- Add 2-second delays between requests to same host
- Add comment explaining why

### Task 2: Detect Rate Limiting (20 min)
- Update `internal/fetch/client.go` fetch()
- Check for 429, 403, 503 status codes
- Return descriptive errors
- Add to audit trail warnings

### Task 3: HTTP Caching Support (45 min)
- Add lastModified map to Client struct
- Send If-Modified-Since headers
- Handle 304 Not Modified responses
- Persist Last-Modified to file (data/http-cache.json)
- Fall back to snapshot on 304

### Task 4: Testing (20 min)
- Test with delays
- Test rate limit detection (mock 429 response)
- Test HTTP caching (mock 304 response)
- Verify build still works

### Task 5: Documentation (10 min)
- Document respectful fetching practices in CLAUDE.md
- Add comments explaining delays
- Update README if needed

---

## Expected Impact

### Before
- 4 requests with no delays (~2s build time for fetching)
- Full re-download every hour (1-2 MB × 4 = 4-8 MB/hour)
- No rate limit awareness
- Potential to get blocked

### After (Priority 1 only)
- 4 requests with 2s delays (~8s build time for fetching)
- Full re-download every hour (same bandwidth)
- Rate limit detection with clear errors
- HTTP caching reduces bandwidth by ~90% (if data unchanged)

### After (All priorities)
- 4 requests with smart delays + retries (~8-15s worst case)
- Bandwidth reduced by 90% via caching
- Automatic retry on temporary failures
- Respectful of upstream rate limits
- Clear audit trail of all requests

---

## Trade-offs

**Pros:**
- ✅ Respectful to upstream servers
- ✅ Reduced bandwidth (HTTP caching)
- ✅ Better failure handling
- ✅ Less likely to get blocked
- ✅ Clear detection of rate limiting

**Cons:**
- ⏱️ Slower builds (2s → 8s for Priority 1)
- 🔧 More complexity (caching logic)
- 💾 Need persistent storage for cache headers

**Decision:** Trade-off is worth it. 6 extra seconds per hour is negligible, and being respectful is critical.

---

## Success Criteria

✅ Delays between requests to same host (2+ seconds)
✅ Rate limit detection (429/403/503)
✅ HTTP caching with If-Modified-Since
✅ Clear errors when rate limited
✅ Audit trail includes HTTP status codes
✅ Build time < 15 seconds (acceptable for hourly cron)
✅ No increase in failed builds

---

## Alternative: Use Only One Format

**Consideration:** We fetch 3 formats (JSON, XML, CSV) from datos.madrid.es for the same data.

**Why we do this:**
- Redundancy: if one format is malformed, others work
- Deduplication: merging helps catch missing fields
- Different sources sometimes have different completeness

**Could we use only JSON?**
- Yes, technically
- Would reduce requests from 3 → 1 to datos.madrid.es
- But we'd lose redundancy

**Recommendation:** Keep all 3 for now, but add delays. The redundancy has saved us from data quality issues.

---

## Next Steps

1. **Implement Priority 1 tasks** (IMMEDIATE)
2. **Test thoroughly**
3. **Monitor for rate limiting in audit trail**
4. **Consider Priority 2 if we see issues**
5. **Document best practices**

