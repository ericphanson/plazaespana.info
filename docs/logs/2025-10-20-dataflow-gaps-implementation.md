# Dataflow Gaps Implementation Log

**Date:** 2025-10-20
**Plan:** docs/plans/2025-10-20-013-dataflow-gaps-implementation.md
**Status:** In Progress

## Implementation Progress

### Phase 1: Audit Completeness

#### Task 1.1: Extend Audit File with Parse Errors (30 min)
**Status:** ✅ Complete
**Started:** 2025-10-20
**Completed:** 2025-10-20

**Goal:** Add ParseErrors section to audit file to capture events that fail to parse.

**Files modified:**
- `internal/audit/types.go` - NEW: Created with AuditParseError and ParseErrorsAudit types
- `internal/audit/export.go` - Updated AuditFile struct, SaveAuditJSON signature, added processParseErrors()
- `internal/audit/export_test.go` - Updated tests to include parse errors, added TestSaveAuditJSON_WithParseErrors
- `cmd/buildsite/main.go` - Updated to collect cultural parse errors and city parse errors, pass to audit

**Implementation:**
1. ✅ Created types.go with AuditParseError and ParseErrorsAudit structs
2. ✅ Updated AuditFile to include ParseErrors field
3. ✅ Updated SaveAuditJSON signature to accept culturalParseErrors and cityParseErrors
4. ✅ Added processParseErrors() function to convert event.ParseError to AuditParseError
5. ✅ Updated main.go to track city parse errors with details (was just a counter)
6. ✅ Updated main.go to collect all cultural parse errors from pipeResult
7. ✅ Updated main.go to pass parse errors to SaveAuditJSON
8. ✅ Updated logging to show parse error count
9. ✅ Added test for parse errors in audit file
10. ✅ All tests passing (4 tests in audit package)

**Test Results:**
```
=== RUN   TestSaveAuditJSON
--- PASS: TestSaveAuditJSON (0.00s)
=== RUN   TestProcessCulturalEvents
--- PASS: TestProcessCulturalEvents (0.00s)
=== RUN   TestSaveAuditJSON_WithParseErrors
--- PASS: TestSaveAuditJSON_WithParseErrors (0.00s)
=== RUN   TestProcessCityEvents
--- PASS: TestProcessCityEvents (0.00s)
PASS
ok      github.com/ericphanson/madrid-events/internal/audit     0.008s
```

**Notes:**
- Parse errors now fully audited with source, index, raw data, error message, and recovery type
- Audit file now includes parse_errors section with cultural and city subsections
- Backward compatible - existing audit files will continue to work
- Successfully compiled and all tests passing

---

#### Task 1.2: Fix City Event Count Reporting (15 min)
**Status:** In Progress
**Started:** 2025-10-20

**Goal:** Fix city fetch attempt to show parsed events count, not services fetched count.

**Problem:** Currently cityFetchAttempt.EventCount shows `len(esmadridServices)` (services fetched) instead of `len(allCityEvents)` (events successfully parsed).

**Files to modify:**
- `cmd/buildsite/main.go` - Update cityFetchAttempt.EventCount

**Implementation:**
1. ✅ Updated cityFetchAttempt.EventCount to show parsed events (len(cityEvents))
2. ✅ Added error message showing "Parsed X/Y services successfully" when parse errors occur
3. ✅ Updated report after parsing completes
4. ✅ Added fmt import to main.go
5. ✅ Compiled successfully

**Changes:**
- After parsing city events, update cityFetchAttempt.EventCount to show parsed count
- When parse errors exist, set Error field to show both counts (parsed/fetched)
- Update buildReport.CityPipeline.Fetching.Attempts with corrected data

**Before:**
- EventCount showed services fetched (e.g., 50 services)
- No indication of parse failures

**After:**
- EventCount shows successfully parsed events (e.g., 45 events)
- Error field shows "Parsed 45/50 services successfully" when parse errors occur
- More accurate reporting for build report consumers

**Status:** ✅ Complete
**Completed:** 2025-10-20

**Notes:**
- Build report now accurately reflects parsed events, not just fetched services
- Parse error visibility improved with ratio display
- No changes to report structure (backward compatible)

---

### Phase 2: Reporting Accuracy

#### Task 2.1: Fix Cultural Filtering Stats (45 min)
**Status:** In Progress
**Started:** 2025-10-20

**Goal:** Fix Build Report cultural filtering stats to avoid double-counting and mixing categories.

**Problem:**
- `OutsideRadius` includes both "outside distrito" and "outside GPS radius" events (mixing categories)
- `MissingCoords` is set to `missingBoth` (no distrito AND no coords), which is wrong
- `Kept` uses `len(filteredEvents) + pastEvents`, double-counting in some contexts

**Solution:** Track independent counters per FilterReason and populate stats correctly.

**Files to modify:**
- `cmd/buildsite/main.go` - Cultural events filtering stats calculation

**Implementation:**
1. ✅ Analyzed current filtering stats code
2. ✅ Replaced mixed counters with clean switch statement based on FilterResult.FilterReason
3. ✅ Fixed DistrictoFilterStats to use correct counters:
   - Input: len(allEvents) instead of len(merged)
   - Filtered: outsideDistrito (only "outside target distrito")
   - Kept: keptEvents (only "kept" events)
4. ✅ Fixed GeoFilterStats to use correct counters:
   - Input: len(allEvents)
   - MissingCoords: missingCoords (only "missing location data" reason)
   - OutsideRadius: outsideRadius (only "outside GPS radius")
   - Kept: keptEvents (only "kept" events)
5. ✅ Fixed TimeFilterStats to use correct counters:
   - Input: len(allEvents)
   - PastEvents: tooOld (only "event too old")
   - Kept: keptEvents (only "kept" events)
6. ✅ Fixed city pipeline stats similarly
7. ✅ All tests passing

**Key changes:**
- Removed mixed counters (outsideAll, missingBoth, pastEvents)
- Added switch statement to count events by FilterReason
- Each stat field now maps to exactly one FilterReason
- No double-counting: each event counted exactly once

**Before:**
```go
outsideAll++  // Mixed "outside distrito" AND "outside GPS radius"
Filtered: outsideAll  // Wrong: mixing categories
Kept: len(filteredEvents) + pastEvents  // Wrong: includes "too old" events
MissingCoords: missingBoth  // Wrong: "no distrito AND no coords"
```

**After:**
```go
case "outside target distrito": outsideDistrito++
case "outside GPS radius": outsideRadius++
case "event too old": tooOld++
case "kept": keptEvents++

Filtered: outsideDistrito  // Correct: only distrito-filtered events
Kept: keptEvents  // Correct: only kept events
MissingCoords: missingCoords  // Correct: "missing location data" reason
OutsideRadius: outsideRadius  // Correct: only GPS-filtered events
```

**Status:** ✅ Complete
**Completed:** 2025-10-20

**Validation:**
- Each event is counted exactly once based on its FilterReason
- Stats fields map 1:1 to filter reasons (no mixing)
- Percentages will add up to 100% (Input = sum of all categories)
- Build report accuracy dramatically improved

**Notes:**
- This was a major refactoring of stats calculation
- All existing tests continue to pass
- Build report will now show accurate, non-overlapping categories
