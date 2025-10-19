# Madrid Events Site Generator - Implementation Log

**Plan:** `docs/plans/2025-10-19-madrid-events-site-generator.md`
**Started:** 2025-10-19
**Execution Mode:** Subagent-driven development

---

## Environment Verification

**Tooling Check (Pre-implementation)**
- ✅ Go version: 1.25.3 linux/arm64
- ✅ gofmt available: `/usr/local/go/bin/gofmt`
- ✅ Build script verified: `scripts/build-freebsd.sh` (works, needs go.mod)
- ✅ All required tools present

**Status:** Ready to begin implementation

---

## Task Progress

### Task 1: Project Initialization
**Status:** ✅ Completed
**Completed:** 2025-10-19
**Commit:** 6f36d97

**Steps Completed:**
1. ✅ Initialized Go module with `go mod init github.com/yourusername/madrid-events`
2. ✅ Updated `.gitignore` to include `buildsite` binary artifact (other entries already present)
3. ✅ Verified Go version: go1.25.3 (exceeds required 1.21+)
4. ✅ Committed changes with proper attribution

**Files Created/Modified:**
- Created: `go.mod` (module: github.com/yourusername/madrid-events, go 1.25.3)
- Modified: `.gitignore` (added buildsite artifact)

**Test Results:** N/A (no tests for initialization task)

**Issues Encountered:** None - .gitignore already contained most required entries from previous setup

---

### Task 2: Create Directory Structure
**Status:** ✅ Completed
**Completed:** 2025-10-19
**Commit:** eb10313

**Steps Completed:**
1. ✅ Created directory tree with `mkdir -p` for all required directories
2. ✅ Created `.gitkeep` files in each directory (8 total)
3. ✅ Verified structure with `tree -L 2 -a`
4. ✅ Committed changes with proper attribution

**Directories Created:**
- `cmd/buildsite/` - Main CLI entry point
- `internal/fetch/` - HTTP client for Madrid open data API
- `internal/parse/` - Format-specific decoders
- `internal/filter/` - Location and time filtering
- `internal/render/` - Static site generation
- `internal/snapshot/` - Resilience/fallback system
- `templates/` - HTML templates
- `assets/` - Frontend assets (CSS)
- `ops/` - Deployment artifacts

**Files Created:**
- `cmd/buildsite/.gitkeep`
- `internal/fetch/.gitkeep`
- `internal/parse/.gitkeep`
- `internal/filter/.gitkeep`
- `internal/render/.gitkeep`
- `internal/snapshot/.gitkeep`
- `templates/.gitkeep`
- `assets/.gitkeep`
- `ops/.gitkeep`

**Verification Results:**
```
tree -L 2 -a output shows all 9 directories with .gitkeep files
Total: 8 files created (parse directory created but not used in final plan)
```

**Test Results:** N/A (no tests for directory structure task)

**Issues Encountered:** None - all directories created successfully

---

### Task 3: Define Event Types (TDD)
**Status:** ✅ Completed
**Completed:** 2025-10-19
**Commit:** af867f3

**Steps Completed:**
1. ✅ Wrote failing test in `internal/fetch/types_test.go`
2. ✅ Ran test to verify failure (undefined: JSONResponse, RawEvent)
3. ✅ Wrote minimal implementation in `internal/fetch/types.go`
4. ✅ Fixed JSON tag issue (removed ,string from float64 fields)
5. ✅ Ran test to verify success - all tests pass

**Files Created:**
- `internal/fetch/types_test.go` - Test for RawEvent and JSONResponse types
- `internal/fetch/types.go` - Event type definitions matching Madrid API structure

**Test Results:**
```
Initial run: FAIL (expected - undefined types)
After implementation: FAIL (JSON unmarshal error - ,string tag on float64)
After fix: PASS
- TestEvent_UnmarshalJSON: PASS
- TestRawEvent_Fields: PASS
```

**Implementation Details:**
- `RawEvent` struct with fields matching Madrid API (ID-EVENTO, TITULO, etc.)
- Support for both JSON and XML tags for multi-format fallback
- `JSONResponse` wraps JSON-LD structure with @context and @graph
- Coordinates stored as float64 (not string) for direct JSON unmarshaling

**Issues Encountered:**
- Initial implementation used `json:"COORDENADA-LATITUD,string"` tag which is invalid for unmarshaling numeric values
- Fixed by removing `,string` modifier - Madrid API provides coordinates as numbers

---

### Task 4: HTTP Client with User-Agent (TDD)
**Status:** ✅ Completed
**Completed:** 2025-10-19
**Commit:** 537e3f8

**Steps Completed:**
1. ✅ Wrote failing test in `internal/fetch/client_test.go`
2. ✅ Ran test to verify failure (undefined: NewClient, FetchJSON)
3. ✅ Wrote minimal implementation in `internal/fetch/client.go`
4. ✅ Ran test to verify success - all tests pass

**Files Created:**
- `internal/fetch/client_test.go` - Tests for HTTP client with User-Agent verification
- `internal/fetch/client.go` - HTTP client implementation with User-Agent header

**Test Results:**
```
Initial run: FAIL (expected - undefined: NewClient)
After implementation: PASS
- TestNewClient: PASS (0.00s)
- TestClient_FetchWithUserAgent: PASS (0.01s)
- TestEvent_UnmarshalJSON: PASS (0.00s) [from Task 3]
- TestRawEvent_Fields: PASS (0.00s) [from Task 3]
Total: 4/4 tests passing
```

**Implementation Details:**
- `Client` struct with configurable timeout and User-Agent
- `NewClient()` constructor accepting timeout parameter
- `FetchJSON()` method that:
  - Creates HTTP request with User-Agent header
  - Handles HTTP errors (non-200 status codes)
  - Reads and decodes JSON response
  - Returns JSONResponse or error with context
- User-Agent: "madrid-events-site-generator/1.0 (https://github.com/yourusername/madrid-events)"

**Issues Encountered:** None - implementation followed TDD approach exactly as planned

---

### Task 5: XML Fetch Fallback (TDD)
**Status:** ✅ Completed
**Completed:** 2025-10-19
**Commit:** 6efae97

**Steps Completed:**
1. ✅ Wrote failing test in `internal/fetch/client_test.go` (TestClient_FetchXML)
2. ✅ Ran test to verify failure (undefined: FetchXML)
3. ✅ Added `encoding/xml` import to `internal/fetch/client.go`
4. ✅ Wrote minimal implementation (XMLResponse type and FetchXML method)
5. ✅ Ran test to verify success - all tests pass

**Files Modified:**
- Modified: `internal/fetch/client_test.go` - Added TestClient_FetchXML
- Modified: `internal/fetch/client.go` - Added XMLResponse type and FetchXML method

**Test Results:**
```
Initial run: FAIL (expected - undefined: FetchXML)
After implementation: PASS
- TestNewClient: PASS (0.00s)
- TestClient_FetchWithUserAgent: PASS (0.00s)
- TestClient_FetchXML: PASS (0.00s) [NEW]
- TestEvent_UnmarshalJSON: PASS (0.00s) [from Task 3]
- TestRawEvent_Fields: PASS (0.00s) [from Task 3]
Total: 5/5 tests passing
```

**Implementation Details:**
- `XMLResponse` struct wraps Madrid API XML structure with `<response>` root and `<event>` children
- `FetchXML()` method that:
  - Creates HTTP request with User-Agent header
  - Handles HTTP errors (non-200 status codes)
  - Reads and decodes XML response
  - Returns []RawEvent or error with context
- Uses same error handling pattern as FetchJSON for consistency
- RawEvent struct already had XML tags from Task 3, enabling seamless XML unmarshaling

**Issues Encountered:** None - implementation followed TDD approach exactly as planned

---

### Task 6: CSV Fetch Fallback (TDD)
**Status:** ✅ Completed
**Completed:** 2025-10-19
**Commit:** 192fec5

**Steps Completed:**
1. ✅ Wrote failing tests in `internal/fetch/client_test.go` (TestClient_FetchCSV_Semicolon and TestClient_FetchCSV_Comma)
2. ✅ Ran test to verify failure (undefined: FetchCSV)
3. ✅ Added `bytes` and `encoding/csv` imports to `internal/fetch/client.go`
4. ✅ Wrote minimal implementation (FetchCSV method, parseCSV and getField helper functions)
5. ✅ Fixed delimiter detection issue (added validation for ID-EVENTO column to detect wrong delimiter)
6. ✅ Ran test to verify success - all tests pass

**Files Modified:**
- Modified: `internal/fetch/client_test.go` - Added TestClient_FetchCSV_Semicolon and TestClient_FetchCSV_Comma
- Modified: `internal/fetch/client.go` - Added FetchCSV method, parseCSV and getField helper functions

**Test Results:**
```
Initial run: FAIL (expected - undefined: FetchCSV)
After initial implementation: FAIL (TestClient_FetchCSV_Comma failing - empty event data)
After delimiter validation fix: PASS
- TestNewClient: PASS (0.00s)
- TestClient_FetchWithUserAgent: PASS (0.00s)
- TestClient_FetchXML: PASS (0.00s) [from Task 5]
- TestClient_FetchCSV_Semicolon: PASS (0.00s) [NEW]
- TestClient_FetchCSV_Comma: PASS (0.00s) [NEW]
- TestEvent_UnmarshalJSON: PASS (0.00s) [from Task 3]
- TestRawEvent_Fields: PASS (0.00s) [from Task 3]
Total: 7/7 tests passing
```

**Implementation Details:**
- `FetchCSV()` method that:
  - Creates HTTP request with User-Agent header
  - Handles HTTP errors (non-200 status codes)
  - Reads response body into memory
  - Tries semicolon delimiter first (Madrid's preferred format)
  - Falls back to comma delimiter if semicolon fails or produces no events
  - Returns []RawEvent or error with context
- `parseCSV()` helper function that:
  - Parses CSV with specified delimiter
  - Builds header map from first row
  - Validates presence of ID-EVENTO column (detects wrong delimiter usage)
  - Parses coordinates using fmt.Sscanf
  - Returns []RawEvent or error
- `getField()` helper function for safe column access by name
- Supports both semicolon (;) and comma (,) delimiters with automatic fallback

**Issues Encountered:**
- Initial implementation had a subtle bug: when parsing comma-delimited CSV with semicolon delimiter, the CSV parser still "succeeds" but treats entire lines as single fields
- This caused the semicolon parse to return 2 records (header + data) but with only 1 field each containing the full comma-separated line
- The headerMap would not contain "ID-EVENTO" as a separate key
- Fixed by adding validation: check if "ID-EVENTO" column exists in headerMap before processing data rows
- This ensures the fallback mechanism works correctly by detecting incorrect delimiter usage

---

### Task 7: Haversine Distance Filter (TDD)
**Status:** ✅ Completed
**Completed:** 2025-10-19
**Commit:** 251f866

**Steps Completed:**
1. ✅ Wrote failing test in `internal/filter/geo_test.go` (TestHaversineDistance and TestWithinRadius)
2. ✅ Ran test to verify failure (undefined: HaversineDistance, WithinRadius)
3. ✅ Wrote minimal implementation in `internal/filter/geo.go`
4. ✅ Ran test to verify success - all tests pass
5. ✅ Updated log file with results

**Files Created:**
- Created: `internal/filter/geo_test.go` - Tests for Haversine distance calculation and radius checking
- Created: `internal/filter/geo.go` - Haversine distance implementation for geo filtering

**Test Results:**
```
Initial run: FAIL (expected - undefined: HaversineDistance, undefined: WithinRadius)
After implementation: PASS
- TestHaversineDistance/Same_point: PASS (0.00s) [distance: 0.0 km]
- TestHaversineDistance/Plaza_de_España_to_nearby_point_(~350m): PASS (0.00s) [distance: ~0.35 km]
- TestHaversineDistance/Plaza_de_España_to_far_point_(~5km): PASS (0.00s) [distance: ~5.0 km]
- TestWithinRadius/At_plaza: PASS (0.00s) [within 0.35 km radius]
- TestWithinRadius/Just_inside: PASS (0.00s) [within 0.35 km radius]
- TestWithinRadius/Far_away: PASS (0.00s) [outside 0.35 km radius]
Total: 6/6 tests passing in 0.002s
```

**Implementation Details:**
- `HaversineDistance()` function that:
  - Calculates great-circle distance between two GPS coordinates
  - Uses Haversine formula: a = sin²(Δlat/2) + cos(lat1) × cos(lat2) × sin²(Δlon/2)
  - Returns distance in kilometers (Earth radius: 6371.0 km)
  - Converts degrees to radians for trigonometric calculations
- `WithinRadius()` function that:
  - Checks if distance between two points ≤ specified radius
  - Returns boolean result
  - Wraps HaversineDistance for convenience
- Constants: `earthRadiusKm = 6371.0` (standard Earth radius in kilometers)
- All calculations use float64 precision for accuracy

**Test Coverage:**
- Same point (0 km distance) - validates algorithm handles identical coordinates
- Nearby point (~350m) - validates accuracy at Plaza de España filter radius (0.35 km)
- Far point (~5 km) - validates accuracy at larger distances
- Radius checks for points inside and outside filter boundary
- Uses tolerance thresholds to account for floating-point precision

**Issues Encountered:** None - implementation followed TDD approach exactly as planned, all tests passed on first try

---

### Task 8: Time Parsing with Europe/Madrid Timezone (TDD)
**Status:** ✅ Completed
**Completed:** 2025-10-19
**Commit:** 80aa29c

**Steps Completed:**
1. ✅ Wrote failing test in `internal/filter/time_test.go` (TestParseEventDateTime and TestIsInFuture)
2. ✅ Ran test to verify failure (undefined: ParseEventDateTime, IsInFuture)
3. ✅ Wrote minimal implementation in `internal/filter/time.go`
4. ✅ Ran test to verify success - all tests pass
5. ✅ Updated log file with results

**Files Created:**
- Created: `internal/filter/time_test.go` - Tests for date/time parsing with Madrid timezone
- Created: `internal/filter/time.go` - Date/time parsing implementation for Europe/Madrid

**Test Results:**
```
Initial run: FAIL (expected - undefined: ParseEventDateTime, undefined: IsInFuture)
After implementation: PASS
- TestParseEventDateTime/Valid_date_with_time: PASS (0.00s) [15/11/2025 19:30]
- TestParseEventDateTime/Valid_date_without_time_(all-day): PASS (0.00s) [20/11/2025 all-day]
- TestParseEventDateTime/Invalid_date_format: PASS (0.00s) [error expected and received]
- TestIsInFuture: PASS (0.00s) [future/past time comparison]
Total: 4/4 new tests passing + 6 from Task 7 = 10/10 tests passing in 0.002s
```

**Implementation Details:**
- `ParseEventDateTime()` function that:
  - Parses Madrid API date format (DD/MM/YYYY)
  - Supports optional time in HH:MM format
  - Uses ParseInLocation with Europe/Madrid timezone
  - Layout: "02/01/2006" for dates, "02/01/2006 15:04" for date+time
  - Concatenates fecha and hora when time is provided
  - Returns time.Time in specified timezone or error with context
- `IsInFuture()` function that:
  - Compares event time against reference time (typically now)
  - Returns boolean result using time.After()
  - Enables filtering of past events
- Timezone-aware: All times parsed to Europe/Madrid to handle DST correctly
- Handles all-day events: Empty hora field treated as midnight (start of day)

**Test Coverage:**
- Valid date with time (15/11/2025 19:30) - validates full datetime parsing
- Valid date without time (20/11/2025) - validates all-day event handling
- Invalid date format (2025-11-15) - validates error handling for wrong format
- Future/past comparison - validates IsInFuture logic with 24-hour offsets
- Timezone verification - ensures parsed times are in Europe/Madrid location

**Issues Encountered:** None - implementation followed TDD approach exactly as planned, all tests passed on first try

---

### Task 9: Event Deduplication (TDD)
**Status:** ✅ Completed
**Completed:** 2025-10-19
**Commit:** e8dc2ea

**Steps Completed:**
1. ✅ Wrote failing test in `internal/filter/dedupe_test.go` (TestDeduplicateByID and TestDeduplicateByID_Empty)
2. ✅ Ran test to verify failure (undefined: DeduplicateByID)
3. ✅ Wrote minimal implementation in `internal/filter/dedupe.go`
4. ✅ Ran test to verify success - all tests pass
5. ✅ Updated log file with results

**Files Created:**
- Created: `internal/filter/dedupe_test.go` - Tests for event deduplication by ID-EVENTO
- Created: `internal/filter/dedupe.go` - Event deduplication implementation

**Test Results:**
```
Initial run: FAIL (expected - undefined: DeduplicateByID)
After implementation: PASS
- TestDeduplicateByID: PASS (0.00s) [5 events -> 3 unique events]
- TestDeduplicateByID_Empty: PASS (0.00s) [empty list -> empty list]
- TestHaversineDistance: PASS (0.00s) [from Task 7]
- TestWithinRadius: PASS (0.00s) [from Task 7]
- TestParseEventDateTime: PASS (0.00s) [from Task 8]
- TestIsInFuture: PASS (0.00s) [from Task 8]
Total: 6/6 tests passing in 0.003s
```

**Implementation Details:**
- `DeduplicateByID()` function that:
  - Accepts []fetch.RawEvent slice
  - Removes duplicates based on ID-EVENTO field
  - Keeps first occurrence of each unique ID
  - Skips events with empty ID-EVENTO (defensive)
  - Uses map[string]bool to track seen IDs
  - Returns new slice with unique events only
  - Preserves original order of first occurrences
- Algorithm: O(n) time complexity with single pass through events
- Memory: O(k) where k is number of unique event IDs

**Test Coverage:**
- Duplicate removal: 5 events with 2 duplicate IDs reduced to 3 unique events
- First occurrence preserved: Verified "First" title kept over "Duplicate First"
- All unique IDs present in result: EVT-001, EVT-002, EVT-003
- Empty list handling: Empty input returns empty output (no panics)
- Defensive: Skips events without ID-EVENTO field

**Issues Encountered:** None - implementation followed TDD approach exactly as planned, all tests passed on first try

---

### Task 10: Snapshot Manager for Resilience (TDD)
**Status:** ✅ Completed
**Completed:** 2025-10-19
**Commit:** d46a549

**Steps Completed:**
1. ✅ Wrote failing test in `internal/snapshot/manager_test.go` (TestManager_SaveAndLoad and TestManager_LoadSnapshot_NotExists)
2. ✅ Ran test to verify failure (undefined: NewManager)
3. ✅ Wrote minimal implementation in `internal/snapshot/manager.go`
4. ✅ Ran test to verify success - all tests pass
5. ✅ Updated log file with results

**Files Created:**
- Created: `internal/snapshot/manager_test.go` - Tests for snapshot save/load with atomic writes
- Created: `internal/snapshot/manager.go` - Snapshot manager implementation for fallback resilience

**Test Results:**
```
Initial run: FAIL (expected - undefined: NewManager)
After implementation: PASS
- TestManager_SaveAndLoad: PASS (0.00s) [save 2 events, load 2 events]
- TestManager_LoadSnapshot_NotExists: PASS (0.00s) [error expected and received]
Total: 2/2 tests passing in 0.005s
```

**Implementation Details:**
- `Manager` struct with configurable data directory
- `NewManager()` constructor accepting data directory path
- `SaveSnapshot()` method that:
  - Creates data directory if needed (os.MkdirAll with 0755 permissions)
  - Encodes events to JSON with pretty printing (2-space indentation)
  - Writes to temporary file (last_success.json.tmp)
  - Atomically renames temp file to final location (last_success.json)
  - Returns error with context on any failure
- `LoadSnapshot()` method that:
  - Reads last_success.json from data directory
  - Decodes JSON to []fetch.RawEvent
  - Returns events or error with context
  - Returns error if snapshot file doesn't exist (expected behavior for fallback)
- Atomic writes prevent serving partial updates during file writes
- Uses filepath.Join for cross-platform path construction

**Test Coverage:**
- Save and load cycle: 2 events saved, 2 events loaded with correct content
- Atomic write: Verifies snapshot file exists after SaveSnapshot
- Data integrity: Verifies loaded event IDs match saved event IDs
- Non-existent file: LoadSnapshot returns error when file doesn't exist (expected)
- Temporary directory: Uses t.TempDir() for isolated test execution

**Issues Encountered:** None - implementation followed TDD approach exactly as planned, all tests passed on first try

---

### Task 11: HTML Template Rendering (TDD)
**Status:** ✅ Completed
**Completed:** 2025-10-19
**Commit:** 953a733

**Steps Completed:**
1. ✅ Created HTML template file (templates/index.tmpl.html)
2. ✅ Created types.go with TemplateData and TemplateEvent structs
3. ✅ Wrote failing test in `internal/render/html_test.go` (TestHTMLRenderer_Render)
4. ✅ Ran test to verify failure (undefined: NewHTMLRenderer)
5. ✅ Wrote minimal implementation in `internal/render/html.go`
6. ✅ Ran test to verify success - all tests pass
7. ✅ Committed changes with proper attribution
8. ✅ Updated log file with results

**Files Created:**
- Created: `templates/index.tmpl.html` - HTML template with Spanish localization
- Created: `internal/render/types.go` - TemplateData and TemplateEvent type definitions
- Created: `internal/render/html_test.go` - Tests for HTML renderer with temp file verification
- Created: `internal/render/html.go` - HTML renderer implementation with atomic writes

**Test Results:**
```
Initial run: FAIL (expected - undefined: NewHTMLRenderer)
After implementation: PASS
- TestHTMLRenderer_Render: PASS (0.00s) [template rendering + atomic write]
Total: 1/1 tests passing in 0.003s
```

**Implementation Details:**
- `HTMLRenderer` struct with configurable template path
- `NewHTMLRenderer()` constructor accepting template path
- `Render()` method that:
  - Parses HTML template using html/template package
  - Writes to temporary file (outputPath.tmp)
  - Executes template with provided TemplateData
  - Atomically renames temp file to final location
  - Cleans up temp file on template execution error
  - Returns error with context on any failure
- `TemplateData` struct contains:
  - Lang: Language code for localization (es/en)
  - CSSHash: Content hash for cache-busting CSS filename
  - LastUpdated: Formatted timestamp of generation
  - Events: Slice of TemplateEvent structs for rendering
- `TemplateEvent` struct contains:
  - IDEvento: Event ID for anchor links
  - Titulo: Event title
  - StartHuman: Human-readable start time
  - NombreInstalacion: Venue name
  - ContentURL: Link to full event details
- HTML template features:
  - Conditional Spanish/English titles based on Lang
  - Content-hashed CSS reference (/assets/site.{{.CSSHash}}.css)
  - Semantic HTML5 structure (header, main, article, footer)
  - Empty state message when no events available
  - Madrid open data attribution in footer
  - Conditional rendering of venue and details link

**Test Coverage:**
- Template parsing and execution
- Data binding (Lang, LastUpdated, Events)
- Multiple events rendering (2 test events)
- Output file creation and content verification
- String matching for rendered event titles

**Atomic Write Pattern Verification:**
- Confirmed temp file + rename pattern in implementation
- Prevents serving partial HTML during updates
- Critical for cron-based static site generation
- Cleanup of temp file on template execution errors

**Issues Encountered:**
- Initial types.go had unused "time" import (fixed before test run)
- No other issues - implementation followed TDD approach exactly as planned

---

### Task 12: JSON Output Rendering (TDD)
**Status:** ✅ Completed
**Completed:** 2025-10-20
**Commit:** edad461

**Steps Completed:**
1. ✅ Added JSONEvent type to `internal/render/types.go`
2. ✅ Wrote failing test in `internal/render/json_test.go` (TestJSONRenderer_Render)
3. ✅ Ran test to verify failure (undefined: NewJSONRenderer)
4. ✅ Wrote minimal implementation in `internal/render/json.go`
5. ✅ Ran test to verify success - all tests pass
6. ✅ Committed changes with Co-Authored-By attribution
7. ✅ Updated log file with results

**Files Created/Modified:**
- Modified: `internal/render/types.go` - Added JSONEvent struct with JSON tags
- Created: `internal/render/json_test.go` - Tests for JSON renderer with atomic write verification
- Created: `internal/render/json.go` - JSON renderer implementation with atomic writes

**Test Results:**
```
Initial run: FAIL (expected - undefined: NewJSONRenderer)
After implementation: PASS
- TestHTMLRenderer_Render: PASS (0.00s) [from Task 11]
- TestJSONRenderer_Render: PASS (0.00s) [NEW - JSON rendering + atomic write]
Total: 2/2 tests passing in 0.004s
```

**Implementation Details:**
- `JSONRenderer` struct (empty, stateless)
- `NewJSONRenderer()` constructor for consistency with HTMLRenderer
- `Render()` method that:
  - Encodes []JSONEvent to JSON with pretty printing (2-space indentation)
  - Writes to temporary file (outputPath.tmp)
  - Atomically renames temp file to final location
  - Returns error with context on any failure
- `JSONEvent` struct contains:
  - ID: Event identifier (from ID-EVENTO)
  - Title: Event title
  - StartTime: ISO 8601 formatted start time (RFC3339)
  - EndTime: ISO 8601 formatted end time (optional, omitempty)
  - VenueName: Venue name (optional, omitempty)
  - DetailsURL: Link to full event details (optional, omitempty)
- All fields use JSON tags for proper serialization
- Optional fields use `omitempty` to minimize output size

**Test Coverage:**
- JSON encoding and marshaling
- Single event rendering with all fields populated
- Output file creation and content verification
- JSON validity verification (unmarshal roundtrip)
- Event data integrity (ID field verification)

**Atomic Write Pattern Verification:**
- Confirmed temp file + rename pattern in implementation
- Prevents serving partial JSON during updates
- Critical for cron-based static site generation
- Matches pattern used in HTMLRenderer (Task 11) and SnapshotManager (Task 10)

**Issues Encountered:** None - implementation followed TDD approach exactly as planned, all tests passed on first try

---

### Task 13: Main CLI Orchestration
**Status:** ✅ Completed
**Completed:** 2025-10-20
**Commit:** 9544c5a

**Steps Completed:**
1. ✅ Read Task 13 requirements from implementation plan
2. ✅ Reviewed existing internal package structure (fetch, filter, render, snapshot)
3. ✅ Created `cmd/buildsite/main.go` with complete pipeline orchestration
4. ✅ Built binary successfully (11M Linux binary)
5. ✅ Tested CLI structure with fake URL to verify graceful error handling
6. ✅ Updated log file with results

**Files Created:**
- Created: `cmd/buildsite/main.go` - Complete CLI orchestration with fetch/filter/render pipeline

**Build Results:**
```
Binary: build/buildsite
Size: 11M (Linux/ARM64 - development build)
Build command: go build -o build/buildsite ./cmd/buildsite
Status: SUCCESS (no compilation errors)
```

**CLI Structure Test:**
```bash
./build/buildsite -json-url https://example.com/test.json -out-dir ./public -data-dir ./data

Output:
2025/10/20 00:32:49 Fetching JSON from: https://example.com/test.json
2025/10/20 00:32:49 JSON fetch failed: HTTP request failed: Get "https://example.com/test.json": dial tcp 23.215.0.138:443: connect: no route to host
2025/10/20 00:32:49 All fetch attempts failed, loading snapshot...
2025/10/20 00:32:49 Failed to load snapshot: reading snapshot: open data/last_success.json: no such file or directory

Status: Expected behavior - CLI handles failures gracefully, attempts full fallback chain
```

**Implementation Details:**

**Flag Parsing:**
- `-json-url` (required): Madrid events JSON URL
- `-xml-url` (optional): XML fallback URL
- `-csv-url` (optional): CSV fallback URL
- `-out-dir` (default: ./public): Output directory for static files
- `-data-dir` (default: ./data): Data directory for snapshots
- `-lat` (default: 40.42338): Reference latitude (Plaza de España)
- `-lon` (default: -3.71217): Reference longitude (Plaza de España)
- `-radius-km` (default: 0.35): Filter radius in kilometers
- `-timezone` (default: Europe/Madrid): Timezone for event times

**Pipeline Orchestration:**

1. **Initialization:**
   - Load Europe/Madrid timezone
   - Create HTTP client with 30s timeout
   - Create snapshot manager for data directory

2. **Fetch with Fallback Chain:**
   - Try JSON from primary URL
   - If fails, try XML from fallback URL
   - If fails, try CSV from fallback URL
   - If all fail, load last successful snapshot
   - Log each attempt and result

3. **Deduplication:**
   - Call filter.DeduplicateByID() on raw events
   - Remove duplicate ID-EVENTO entries
   - Log count after deduplication

4. **Filtering:**
   - Geographic: Skip events with missing coordinates (0,0)
   - Geographic: filter.WithinRadius() for Plaza de España proximity
   - Temporal: Parse event dates with filter.ParseEventDateTime()
   - Temporal: Use end date if available, otherwise start date
   - Temporal: filter.IsInFuture() to exclude past events
   - Log count after filtering

5. **Data Transformation:**
   - Convert to render.TemplateEvent for HTML rendering
   - Convert to render.JSONEvent for JSON API output
   - Format timestamps (human-readable for HTML, RFC3339 for JSON)

6. **Rendering:**
   - Create output directory with os.MkdirAll
   - Render HTML to {outDir}/index.html using HTMLRenderer
   - Render JSON to {outDir}/events.json using JSONRenderer
   - Both use atomic writes (temp file + rename)
   - Log generated file paths

7. **Snapshot Management:**
   - On successful fetch: Save to snapshot for future fallback
   - On fetch failure: Load from snapshot (with stale data warning)
   - Snapshot failures logged but don't stop execution

**Error Handling:**
- Fatal errors: Missing required flags, invalid timezone, snapshot load failure (when all fetches fail)
- Warning errors: Snapshot save failure (logged but doesn't stop execution)
- Graceful errors: Individual fetch failures (try next fallback), invalid event dates (skip event)
- All errors include context with fmt.Errorf wrapping

**Logging Strategy:**
- Start: Log fetch attempts with URLs
- Success: Log event counts at each stage (fetched, deduplicated, filtered)
- Failure: Log specific error messages for each fallback attempt
- Output: Log generated file paths
- Completion: Log "Build complete!" message

**Integration with Components:**
- fetch.NewClient() - HTTP client from Task 4-6
- fetch.FetchJSON/FetchXML/FetchCSV() - Multi-format fetching
- filter.DeduplicateByID() - Deduplication from Task 9
- filter.WithinRadius() - Haversine distance from Task 7
- filter.ParseEventDateTime() - Timezone parsing from Task 8
- filter.IsInFuture() - Temporal filtering from Task 8
- snapshot.NewManager() - Snapshot manager from Task 10
- render.NewHTMLRenderer() - HTML rendering from Task 11
- render.NewJSONRenderer() - JSON rendering from Task 12

**Module Path:**
- Uses: github.com/yourusername/madrid-events (from go.mod)
- Imports all internal packages correctly
- No compilation errors or missing dependencies

**Next Steps:**
- Run go test ./... to verify all tests still pass
- Commit changes with Co-Authored-By attribution
- Consider FreeBSD cross-compilation test (Task 18)

**Issues Encountered:** None - CLI compiled successfully, all imports resolved, error handling works as expected

---

### Task 14: Frontend Assets - CSS
**Status:** ✅ Completed
**Completed:** 2025-10-20
**Commit:** (pending)

**Steps Completed:**
1. ✅ Created hand-rolled CSS file (assets/site.css)
2. ✅ Created asset hashing script (scripts/hash-assets.sh)
3. ✅ Made script executable (chmod +x)
4. ✅ Tested asset hashing script successfully
5. ✅ Will commit with Co-Authored-By attribution

**Files Created:**
- Created: `assets/site.css` - Hand-rolled CSS with dark mode support
- Created: `scripts/hash-assets.sh` - Content hashing script for cache busting

**Script Test Results:**
```
Command: ./scripts/hash-assets.sh
Output: Generated: public/assets/site.aabdcaf3.css

Generated Files:
- public/assets/site.aabdcaf3.css (1.2K)
- public/assets/css.hash (contains: aabdcaf3)

Hash Value: aabdcaf3 (first 8 chars of SHA256)
Status: SUCCESS
```

**Implementation Details:**

**CSS Features:**
- CSS custom properties (variables) for theming
- Automatic dark mode support via `@media (prefers-color-scheme: dark)`
- Light mode colors: white background, dark text, blue links
- Dark mode colors: dark background (#0f1115), light text, blue links (#8ab4f8)
- Semantic variables: --bg, --fg, --muted, --card, --link, --accent, --radius, --shadow, --max
- Responsive design: max-width of 900px for content
- System font stack: system-ui, -apple-system, Segoe UI, Roboto, Ubuntu
- Card-based layout: rounded corners (14px), subtle shadows
- Grid-based main layout with 1rem gap
- Accessibility: focus states with outline, semantic color variables

**CSS Selectors:**
- Universal box-sizing: border-box
- Smooth scrolling on html element
- Body: zero margin, variable-based theming, 16px base font size, 1.55 line height
- Container elements: max-width constraint, centered, 1rem padding
- Article cards: background, border-radius, padding, box-shadow
- Typography: muted timestamps (.stamp), event time/location (.when, .where)
- Links: colored, focus outline with 2px solid accent color
- Footer: muted color, smaller font, top margin

**Hash Script Features:**
- Bash script with strict error handling (set -euo pipefail)
- Checks if assets/site.css exists before processing
- Generates SHA256 hash using sha256sum
- Truncates to first 8 characters for filename
- Copies CSS to public/assets/site.$HASH.css
- Writes hash to public/assets/css.hash for template integration
- Creates public/assets directory if needed
- Provides user feedback on success/warning

**Integration Points:**
- Hash value (aabdcaf3) will be read by main.go to update CSSHash in TemplateData
- HTML template references /assets/site.{{.CSSHash}}.css for cache busting
- .htaccess (Task 15) will set 30-day cache for CSS files (safe with content hashing)
- Script should be run before deployment or as part of build process

**Cache Busting Strategy:**
- Content-based hashing ensures unique filename per CSS version
- Browser can cache aggressively (30 days) without stale content risk
- Template dynamically references correct hashed filename
- No need for query string cache busting (?v=123)

**File Sizes:**
- site.css: 1.2K (1,229 bytes) - hand-rolled, no bloat
- site.aabdcaf3.css: 1.2K (identical copy with hashed name)
- css.hash: 9 bytes (8-char hash + newline)

**Next Steps:**
- Update main.go to read css.hash file and use value in TemplateData.CSSHash
- Currently main.go uses placeholder "placeholder" for CSSHash
- Consider running hash-assets.sh as part of build-freebsd.sh or separate deploy script

**Issues Encountered:** None - script worked perfectly on first run, hash generated successfully

---

*Log will be updated after each task completion with status, test results, and any issues encountered.*
