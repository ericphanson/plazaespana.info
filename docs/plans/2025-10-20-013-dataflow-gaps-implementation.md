# Dataflow Gaps Implementation Plan

**Date:** 2025-10-20
**Status:** Planning
**Reference:** docs/dataflow.md "Noted gaps and mismatches" section

## Objective

Address 7 identified gaps in the data pipeline where data does not fully align with intent or metrics may be misleading. Improve audit coverage, fix reporting inaccuracies, and handle edge cases properly.

## Overview

The dataflow document identified specific actionable issues:

1. **Parse failures not audited** - Missing events invisible in audit
2. **Cultural filtering stats mixed** - Misleading Build Report percentages
3. **City coordinates assumed present** - False negatives with missing coords
4. **Source coverage inflated** - Duplicate source labels not deduped
5. **Snapshot fallback not implemented** - Site fails completely during outage
6. **City fetch counts wrong** - Services fetched vs events parsed mismatch
7. **JSON output missing EndTime** - Consumers can't see end times

## Priority Assessment

### High Priority (Accuracy & Completeness)
- Issue 1: Parse failures not audited (data visibility gap)
- Issue 2: Build Report stats incorrect (misleading metrics)
- Issue 3: City coordinates assumed (false negatives)

### Medium Priority (Quality of Life)
- Issue 4: Source coverage inflated (metric accuracy)
- Issue 6: City fetch counts wrong (reporting clarity)
- Issue 7: JSON missing EndTime (API completeness)

### Low Priority (Edge Case Resilience)
- Issue 5: Snapshot fallback not implemented (outage handling)

## Implementation Plan

### Phase 1: Audit Completeness (Issues 1, 6)

#### Task 1.1: Extend Audit File with Parse Errors (30 min)

**Problem:** Events that fail to parse are invisible in audit file.

**Solution:** Add `ParseErrors` section to audit file.

**Files to modify:**
- `internal/audit/types.go` - Add ParseErrors field
- `internal/audit/export.go` - Include parse errors in export
- `cmd/buildsite/main.go` - Pass parse errors to audit

**Implementation:**

```go
// internal/audit/types.go
type AuditFile struct {
    BuildTime    time.Time       `json:"build_time"`
    Duration     time.Duration   `json:"duration"`
    TotalEvents  int             `json:"total_events"`
    CulturalEvents AuditPipeline `json:"cultural_events"`
    CityEvents   AuditPipeline   `json:"city_events"`
    ParseErrors  ParseErrorsAudit `json:"parse_errors"` // NEW
}

type ParseErrorsAudit struct {
    CulturalErrors []AuditParseError `json:"cultural"`
    CityErrors     []AuditParseError `json:"city"`
    TotalErrors    int               `json:"total_errors"`
}

type AuditParseError struct {
    Source      string `json:"source"`      // "JSON", "XML", "CSV", "ESMadrid"
    Index       int    `json:"index"`       // Row/record index
    ID          string `json:"id,omitempty"` // If available
    RawData     string `json:"raw_data,omitempty"`
    Error       string `json:"error"`
    RecoverType string `json:"recover_type"` // "skipped"
}
```

**Changes to main.go:**
```go
// Collect parse errors from pipeline
culturalParseErrors := []audit.AuditParseError{}
for _, e := range pipeResult.JSONErrors {
    culturalParseErrors = append(culturalParseErrors, audit.AuditParseError{
        Source: "JSON",
        Index: e.Index,
        RawData: e.RawData,
        Error: e.Error.Error(),
        RecoverType: e.RecoverType,
    })
}
// Same for XMLErrors, CSVErrors

// Collect city parse errors
cityParseErrors := []audit.AuditParseError{}
for _, e := range parseErrors {
    cityParseErrors = append(cityParseErrors, audit.AuditParseError{
        Source: "ESMadrid",
        Index: e.Index,
        RawData: e.RawData,
        Error: e.Error.Error(),
        RecoverType: e.RecoverType,
    })
}

// Pass to SaveAuditJSON
audit.SaveAuditJSON(
    allEvents, allCityEvents,
    culturalParseErrors, cityParseErrors, // NEW
    auditPath, buildReport.BuildTime, buildReport.Duration,
)
```

**Tests:**
- Test audit file includes parse errors
- Test parse errors from each source (JSON, XML, CSV, ESMadrid)
- Test empty parse errors (all successful)

---

#### Task 1.2: Fix City Event Count Reporting (15 min)

**Problem:** City fetch attempt shows services fetched, not events parsed.

**Solution:** Track both fetched services and parsed city events.

**Files to modify:**
- `cmd/buildsite/main.go` - City events fetch attempt creation

**Implementation:**

```go
// After city events parsing
cityFetchAttempt := report.FetchAttempt{
    Source: "ESMadrid",
    URL:    cfg.CityEvents.XMLURL,
}

if err != nil {
    cityFetchAttempt.Status = "FAILED"
    cityFetchAttempt.Error = err.Error()
} else {
    cityFetchAttempt.Status = "SUCCESS"
    cityFetchAttempt.EventCount = len(allCityEvents) // CHANGED: was len(esmadridServices)
    cityFetchAttempt.HTTPStatus = 200
}

// Add note about parsing
if len(parseErrors) > 0 {
    cityFetchAttempt.Error = fmt.Sprintf("Parsed %d/%d services successfully",
        len(allCityEvents), len(esmadridServices))
}
```

**Tests:**
- Test fetch attempt shows parsed count, not fetched count
- Test error message includes both counts when there are parse errors

---

### Phase 2: Reporting Accuracy (Issue 2)

#### Task 2.1: Fix Cultural Filtering Stats (45 min)

**Problem:** Build Report mixing categories and double-counting.

**Solution:** Track independent counters per filter reason.

**Files to modify:**
- `cmd/buildsite/main.go` - Cultural events filtering stats calculation

**Current problematic code:**
```go
// WRONG: mixes outside distrito + outside radius
outsideAll := outsideDistrito + outsideRadius

// WRONG: double-counting kept events
geo := report.GeoFilterStats{
    TotalEvaluated: totalCultural,
    WithinRadius:   len(filteredEvents) + pastEvents, // WRONG
    OutsideRadius:  outsideAll,                       // WRONG
    MissingCoords:  missingBoth,                      // WRONG
}
```

**Corrected implementation:**
```go
// Track separate counters
var (
    keptEvents       = 0
    outsideDistrito  = 0
    outsideRadius    = 0
    missingCoords    = 0
    tooOld           = 0
)

for _, evt := range allEvents {
    switch evt.FilterResult.FilterReason {
    case "kept":
        keptEvents++
    case "outside distrito":
        outsideDistrito++
    case "outside GPS radius":
        outsideRadius++
    case "missing location data":
        missingCoords++
    case "too old":
        tooOld++
    }
}

// Populate stats correctly
geo := report.GeoFilterStats{
    TotalEvaluated: len(allEvents),
    WithinRadius:   keptEvents,           // FIXED
    OutsideRadius:  outsideRadius,        // FIXED
    MissingCoords:  missingCoords,        // FIXED
}

distrito := report.DistrictoFilterStats{
    TotalEvaluated: len(allEvents),
    InDistritos:    keptEvents,           // FIXED
    OutsideDistrito: outsideDistrito,     // FIXED
}

time := report.TimeFilterStats{
    TotalEvaluated: len(allEvents),
    Current:        keptEvents,           // FIXED
    TooOld:         tooOld,               // FIXED
}
```

**Tests:**
- Test each filter reason counted independently
- Test no double-counting
- Test percentages add up to 100%

---

### Phase 3: City Events Edge Cases (Issue 3)

#### Task 3.1: Fix City Coordinates Assumption (30 min)

**Problem:** City events assume coordinates always present (set HasCoordinates=true unconditionally).

**Solution:** Check for non-zero coordinates before setting HasCoordinates.

**Files to modify:**
- `cmd/buildsite/main.go` - City events filtering

**Current code:**
```go
result.HasCoordinates = true // WRONG: assumes always present
```

**Corrected code:**
```go
// Check if coordinates are actually present
hasCoords := evt.Latitude != 0.0 && evt.Longitude != 0.0
result.HasCoordinates = hasCoords

// If no coordinates, mark as missing location data
if !hasCoords {
    result.Kept = false
    result.FilterReason = "missing location data"
} else {
    // Continue with geo filtering
    ...
}
```

**Alternative approach (if city events should support text fallback):**
```go
hasCoords := evt.Latitude != 0.0 && evt.Longitude != 0.0
result.HasCoordinates = hasCoords

if !hasCoords {
    // Try text matching fallback (like cultural events)
    textMatch := strings.Contains(strings.ToLower(evt.Venue), "plaza")
    result.TextMatched = textMatch

    if textMatch {
        result.Kept = true
        result.FilterReason = "kept"
    } else {
        result.Kept = false
        result.FilterReason = "missing location data"
    }
} else {
    // Geo filtering
    ...
}
```

**Tests:**
- Test city event with lat=0, lon=0 marked as missing coords
- Test city event with valid coords proceeds to geo filtering
- Test filter reason correctly set

---

### Phase 4: Data Quality (Issues 4, 7)

#### Task 4.1: Deduplicate Source Labels (20 min)

**Problem:** Same event from same source multiple times inflates coverage.

**Solution:** Deduplicate Sources slice after merging.

**Files to modify:**
- `internal/pipeline/pipeline.go` - Merge function

**Implementation:**
```go
// After merging all sources
for _, evt := range seen {
    // Deduplicate Sources
    evt.Sources = deduplicateStrings(evt.Sources)
    merged = append(merged, *evt)
}

// Helper function
func deduplicateStrings(input []string) []string {
    seen := make(map[string]bool)
    result := []string{}
    for _, s := range input {
        if !seen[s] {
            seen[s] = true
            result = append(result, s)
        }
    }
    return result
}
```

**Tests:**
- Test duplicate sources from same source are deduped
- Test sources from different sources are preserved
- Test coverage stats (InTwoSources, InAllThree) are accurate

---

#### Task 4.2: Add EndTime to JSON Output (15 min)

**Problem:** JSON API omits EndTime even when available.

**Solution:** Populate EndTime in JSONEvent serialization.

**Files to modify:**
- `internal/render/json.go` - JSONEvent population

**Implementation:**
```go
// For cultural events
func culturalToJSON(evt event.CulturalEvent) JSONEvent {
    return JSONEvent{
        ID:         evt.ID,
        Title:      evt.Title,
        StartTime:  evt.StartTime,
        EndTime:    &evt.EndTime,    // ADDED (use pointer for omitempty)
        VenueName:  evt.VenueName,
        DetailsURL: evt.DetailsURL,
    }
}

// For city events
func cityToJSON(evt event.CityEvent) JSONEvent {
    return JSONEvent{
        ID:         evt.ID,
        Title:      evt.Title,
        StartTime:  evt.StartDate,
        EndTime:    &evt.EndDate,     // ADDED
        VenueName:  evt.Venue,
        DetailsURL: evt.WebURL,
    }
}

// Update JSONEvent struct
type JSONEvent struct {
    ID         string     `json:"id"`
    Title      string     `json:"title"`
    StartTime  time.Time  `json:"start_time"`
    EndTime    *time.Time `json:"end_time,omitempty"` // CHANGED: pointer for omitempty
    VenueName  string     `json:"venue_name"`
    DetailsURL string     `json:"details_url"`
}
```

**Tests:**
- Test EndTime included when available
- Test EndTime omitted when nil
- Test both cultural and city events

---

### Phase 5: Resilience (Issue 5)

#### Task 5.1: Implement Snapshot Fallback (45 min)

**Problem:** When all sources fail, site renders nothing despite having snapshots.

**Solution:** Load snapshot when all sources fail.

**Files to modify:**
- `cmd/buildsite/main.go` - After FetchAll failure detection

**Implementation:**
```go
// After checking if all sources failed
if allSourcesFailed(pipeResult) {
    log.Println("All fetch sources failed. Attempting to load snapshot...")

    snapshot, err := snapMgr.LoadSnapshot()
    if err != nil {
        log.Printf("Warning: Failed to load snapshot: %v", err)
        // Continue with empty result
    } else {
        log.Printf("Loaded snapshot with %d events from %s", len(snapshot.Events), snapshot.FetchedAt)

        // Convert RawEvent back to CulturalEvent
        snapshotEvents := make([]event.CulturalEvent, 0, len(snapshot.Events))
        for _, raw := range snapshot.Events {
            canonical, err := raw.ToCanonical(loc)
            if err != nil {
                log.Printf("Warning: Failed to convert snapshot event %s: %v", raw.IDEvento, err)
                continue
            }

            // Mark as from snapshot
            canonical.Sources = []string{"SNAPSHOT"}
            snapshotEvents = append(snapshotEvents, canonical)
        }

        merged = snapshotEvents
        buildReport.AddWarning("Using snapshot data - all fetch attempts failed")
    }
}
```

**Additional changes:**
- Mark snapshot-sourced events in audit file
- Add "data_source" field to audit: "live" vs "snapshot"
- Update build report to show snapshot usage

**Tests:**
- Test snapshot loads when all sources fail
- Test snapshot events are properly converted
- Test snapshot source is marked in audit
- Test build report shows snapshot warning

---

## Testing Strategy

### Unit Tests
- Test each fix independently with targeted unit tests
- Test edge cases (zero coords, missing data, empty sources)
- Test audit file structure and content

### Integration Tests
- Test full pipeline with parse errors
- Test full pipeline with all sources failing (snapshot)
- Test build report accuracy with various filter combinations
- Test JSON API completeness

### Validation
- Run against real Madrid APIs
- Verify audit file includes all events (kept + filtered + parse errors)
- Verify build report stats add up to 100%
- Verify JSON API includes EndTime

## Success Criteria

- ✅ Audit file includes parse errors (100% data visibility)
- ✅ Build Report stats are accurate (no double-counting, no mixing)
- ✅ City events handle missing coordinates correctly (no false negatives)
- ✅ Source coverage metrics are accurate (no inflation)
- ✅ City fetch counts show parsed events (not raw services)
- ✅ JSON API includes EndTime (API completeness)
- ✅ Snapshot fallback works during outages (resilience)

## Estimated Time

- Phase 1 (Audit): 45 minutes
- Phase 2 (Reporting): 45 minutes
- Phase 3 (City Events): 30 minutes
- Phase 4 (Data Quality): 35 minutes
- Phase 5 (Resilience): 45 minutes
- Testing & Validation: 60 minutes

**Total: ~4 hours**

## Risk Assessment

**Low Risk:**
- All changes are additive or corrective
- No breaking changes to external APIs
- Backward compatible audit file (new fields only)

**Medium Risk:**
- Snapshot fallback (new code path, needs testing)
- Filtering stats changes (need to verify percentages)

**Mitigation:**
- Comprehensive test coverage
- Test against real APIs before deployment
- Keep old behavior available via feature flag if needed

## Dependencies

- Existing audit system (internal/audit)
- Existing snapshot system (internal/snapshot)
- Existing event types (internal/event)

## Documentation Updates

- Update docs/dataflow.md to mark issues as resolved
- Update CLAUDE.md with snapshot fallback behavior
- Update README.md if user-facing changes

## Deployment

All changes are backward compatible. Deploy as normal:
1. Run full test suite
2. Build FreeBSD binary
3. Deploy to production
4. Monitor audit file and build report

## Future Enhancements

Beyond this plan:
- Add parse error recovery strategies (fuzzy matching, partial parse)
- Add configurable filter priorities
- Add audit file viewer/query tool
- Add historical audit file comparison

---

**Ready for implementation when approved.**
