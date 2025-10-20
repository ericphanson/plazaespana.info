# Implementation Log: Parallel Pipelines with Robust Data Handling

**Date:** 2025-10-20
**Plan:** `docs/plans/2025-10-20-parallel-pipelines.md`
**Approach:** Subagent-driven development with code review

---

## Phase 0: Test Infrastructure

### Task 0: Create Test Fixtures (PARTIAL)

**Status:** STARTED
**Started:** 2025-10-20 02:18:00

**Goal:** Download Madrid data once for testing, avoid server spam

**Progress:**

✅ Created infrastructure:
- `scripts/fetch-fixtures.sh` - Download script
- `testdata/fixtures/.gitkeep` - Directory structure
- `.gitignore` entries for fixture files

⚠️ Issue: Curl was failing silently without `-f` flag
- Fixed script to use `curl -f -s -L` with proper output flag

✅ **FIXED and COMPLETED:**
- Downloaded all 3 fixtures successfully
- JSON: 40,903 lines
- XML: 34,009 lines
- CSV: 1,003 lines

**Next:** Implement parsers (Tasks 2, 3, 4)

---

## Phase 1: Foundation

### Task 1: Create Canonical Event Type

**Status:** COMPLETED
**Completed:** 2025-10-20 02:20:00

**Implementation:**

✅ Created `internal/event/event.go`:
- CanonicalEvent struct (all required fields)
- SourcedEvent wrapper with source tracking
- ParseResult with events and errors
- ParseError with detailed context

✅ Created `internal/event/event_test.go`:
- TestCanonicalEvent_Creation
- TestSourcedEvent_Tracking (JSON, XML, CSV)
- TestParseResult_Creation
- TestParseError_Fields

**Test Results:** 4/4 tests passing

**Next:** Task 10 (data quality validation)

---

### Task 10: Add Data Quality Validation

**Status:** COMPLETED
**Completed:** 2025-10-20 02:21:00

**Implementation:**

✅ Created `internal/validate/validate.go`:
- ValidateEvent() - checks required fields (ID, Title, StartTime)
- ValidateEvent() - checks coordinate bounds (-90 to 90, -180 to 180)
- SanitizeEvent() - trims whitespace, default EndTime, dedup sources

✅ Created `internal/validate/validate_test.go`:
- TestValidateEvent_RequiredFields (5 test cases)
- TestValidateEvent_CoordinateBounds (11 test cases)
- TestSanitizeEvent_Whitespace
- TestSanitizeEvent_DefaultEndTime (3 test cases)
- TestSanitizeEvent_DeduplicateSources

**Test Results:** 5/5 test functions, 20 total test cases passing

**Next:** Task 2 (JSON parser with robust parsing)

---

### Task 2: Update JSON Parser with Robust Parsing

**Status:** COMPLETED
**Completed:** 2025-10-20 02:23:00

**Implementation:**

✅ Updated `internal/fetch/types.go`:
- Created JSONEvent struct with correct JSON-LD field names (id, title, dtstart, etc.)
- Added parseJSONTime() for Madrid's datetime format
- Implemented JSONEvent.ToCanonical() with validation

✅ Updated `internal/fetch/client.go`:
- FetchJSON() now returns event.ParseResult
- Individual event error recovery (one bad event doesn't crash batch)
- Added fetch() helper with file:// URL support for fixtures
- Removed debug logging

✅ Created `internal/fetch/json_test.go`:
- TestJSONEvent_ToCanonical (8 subtests)
- TestFetchJSON_FieldMapping (uses real fixture)
- TestFetchJSON_PartialFailure

**Test Results:** All JSON tests passing, 1001/1001 events parsed from fixture (100% success)

**Next:** Task 3 (XML parser)

---

### Task 3: Update XML Parser with Robust Parsing

**Status:** COMPLETED
**Completed:** 2025-10-20 02:25:00

**Implementation:**

✅ Updated `internal/fetch/types.go`:
- Created XMLEvent struct with custom UnmarshalXML for Madrid's nested `<atributo nombre="...">` structure
- Implemented recursive attribute extraction for LOCALIZACION data
- Added parseXMLTime() for datetime parsing
- Implemented XMLEvent.ToCanonical() with validation

✅ Updated `internal/fetch/client.go`:
- FetchXML() now returns event.ParseResult
- Individual event error recovery
- Uses fetch() helper (supports file:// URLs)

✅ Created `internal/fetch/xml_test.go`:
- TestXMLEvent_ToCanonical (8 subtests)
- TestFetchXML_FieldMapping (uses real fixture)
- TestFetchXML_PartialFailure

✅ Updated `internal/fetch/client_test.go`:
- Fixed TestClient_FetchXML for new signature

**Test Results:** All XML tests passing, 1001/1001 events parsed from fixture (100% success)

**Next:** Task 4 (CSV parser)

---

### Task 4: Update CSV Parser with Robust Parsing

**Status:** COMPLETED
**Completed:** 2025-10-20 02:27:00

**Implementation:**

✅ Updated `internal/fetch/client.go`:
- FetchCSV() now returns event.ParseResult
- Added parseCSVRow() helper
- Individual row error recovery
- Preserved Windows-1252 encoding conversion
- Preserved delimiter auto-detection

✅ Updated `internal/fetch/types.go`:
- Created CSVEvent struct
- Implemented CSVEvent.ToCanonical() with validation

✅ Created `internal/fetch/csv_test.go`:
- TestFetchCSV_FieldMapping (uses real fixture)
- TestFetchCSV_EncodingConversion (verifies UTF-8)
- TestFetchCSV_PartialFailure

✅ Updated `internal/fetch/client_test.go`:
- Fixed TestClient_FetchCSV_Semicolon
- Fixed TestClient_FetchCSV_Comma

**Test Results:** All CSV tests passing, 1001/1001 events parsed from fixture (100% success)

**Phase 1 Complete!** All three parsers (JSON, XML, CSV) now return ParseResult with robust error recovery.

**Next:** Commit Phase 1, code review, then Task 5 (pipeline orchestrator)

---

## Code Review Fixes

### Fix 1: XML HORA Time Parsing Bug

**Status:** COMPLETED
**Completed:** 2025-10-20 03:15:00

**Issue:** Code review identified that `parseXMLTime()` was not applying the HORA field when date string included time portion (e.g., "2025-10-27 00:00:00.0" with HORA "19:00").

**Root Cause:** Function returned immediately after successfully parsing date, never checking if HORA should override the time portion.

**Fix Applied** in `internal/fetch/types.go` (lines 252-266):
```go
// If we have a separate time string, check if we should override the time portion
// This handles XML where FECHA has "00:00:00" but HORA has the actual time
if timeStr != "" {
    // Check if the parsed time is midnight (likely placeholder)
    if t.Hour() == 0 && t.Minute() == 0 && t.Second() == 0 {
        // Parse time in HH:MM format
        timeOnlyFormat := "15:04"
        timeVal, timeErr := time.Parse(timeOnlyFormat, timeStr)
        if timeErr == nil {
            // Combine date with actual time from HORA field
            t = time.Date(t.Year(), t.Month(), t.Day(),
                timeVal.Hour(), timeVal.Minute(), 0, 0, loc)
        }
    }
}
```

**Test Update:** Updated `internal/fetch/xml_test.go` line 68 to expect 19:00 instead of 00:00.

**Verification:** All XML tests passing (1001/1001 events from fixture)

---

### Fix 2: Make main.go Compile (Temporary Shim)

**Status:** COMPLETED
**Completed:** 2025-10-20 03:18:00

**Issue:** `just test` failing because main.go still used old fetch signatures (missing `*time.Location` parameter).

**Solution:** Added temporary shim layer to convert new `event.ParseResult` back to old `[]fetch.RawEvent` format. This allows tests to pass while deferring full integration until Task 7.

**Changes in `cmd/buildsite/main.go`:**
- Pass `loc` parameter to FetchJSON/FetchXML/FetchCSV
- Convert `ParseResult.Events` (CanonicalEvent) back to RawEvent format
- Handle new error structure (check `len(result.Events) > 0` instead of `err == nil`)

**Test Results:** All 22 tests passing across 5 packages

**Note:** This is a temporary bridge solution. Task 7 will remove RawEvent entirely and work directly with CanonicalEvent.

---

## Phase 2: Pipeline

### Task 5: Create Pipeline Orchestrator

**Status:** COMPLETED
**Completed:** 2025-10-20 03:30:00

**Implementation:**

✅ Created `internal/pipeline/pipeline.go`:
- Pipeline struct with NewPipeline() constructor
- PipelineResult struct tracking events and errors from all 3 sources
- FetchAll() method for sequential fetching with isolation
- fetchJSONIsolated/fetchXMLIsolated/fetchCSVIsolated with panic recovery
- Merge() method for deduplication with source tracking

✅ Created `internal/pipeline/pipeline_test.go` (6 tests):
- TestPipeline_FetchAll_Sequential (verifies all 3 sources work)
- TestPipeline_FetchAll_ErrorIsolation (JSON failure doesn't prevent CSV/XML)
- TestPipeline_Merge_Deduplication (3003 events → 1055 unique)
- TestPipeline_Merge_SourceTracking (947 events in all 3 sources, 54 in 2, 54 in 1)
- TestPipeline_Merge_HandlesFailures (merge works with partial failures)
- TestPipeline_Merge_EmptyResult (handles all-failures case)

**Key Features:**
- Each source isolated with panic recovery (one failure doesn't crash others)
- Sequential fetching (avoids server spam, meets user requirement)
- Deduplication by ID with multi-source tracking
- Empty slice returned (not nil) when no events

**Test Results:** 6/6 tests passing, all using real fixtures

**Insights from real data:**
- 3003 total events (1001 each from JSON/XML/CSV)
- 1055 unique events after deduplication
- 1948 duplicates removed (64.8% deduplication rate)
- 947 events found in all 3 sources (89.7%)
- 54 events found in 2 sources (5.1%)
- 54 events found in 1 source only (5.1%)

This confirms the three sources have significant overlap but aren't identical - robust multi-source fetching is valuable!

**Next:** Task 6 (update build reporting) - skipped for now, moving to Task 7 (main.go integration)

---
