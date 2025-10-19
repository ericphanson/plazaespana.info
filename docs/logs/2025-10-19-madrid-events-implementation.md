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

*Log will be updated after each task completion with status, test results, and any issues encountered.*
