# Implementation Plan: Parallel Data Pipelines with Late Deduplication

**Date:** 2025-10-20
**Goal:** Parse all three data sources independently, canonicalize to common format, then deduplicate with source tracking

## Current Architecture Problems

1. **Early Fallback:** JSON fails → XML fails → CSV succeeds (only 1 source used)
2. **Field Name Mismatch:** JSON/XML use different field names than CSV
3. **No Source Tracking:** Can't tell which sources provided which events
4. **Poor Reporting:** Can't show "Event X found in 2/3 sources"
5. **Fragile Parsing:** Single bad event crashes entire source parsing
6. **No Data Quality Tracking:** Don't know how many events failed to parse

## New Architecture: Independent Pipelines with Sequential Fetching

```
┌─────────────┐
│ JSON Source │  Fetch sequentially (polite to servers)
└──────┬──────┘  Each in try/catch (isolated errors)
       │
       ▼
  Parse JSON ─────────┐
  (id, title)         │
       │              │
       ▼              │
  Canonicalize        │
       │              │
       ▼              │
┌─────────────┐       │
│ XML Source  │       │  Independent pipelines
└──────┬──────┘       │  (JSON failure doesn't affect XML)
       │              │
       ▼              │
  Parse XML ──────────┤
  (ID-EVENTO)         │
       │              │
       ▼              │
  Canonicalize        │
       │              │
       ▼              │
┌─────────────┐       │
│ CSV Source  │       │
└──────┬──────┘       │
       │              │
       ▼              │
  Parse CSV ──────────┘
  (ID-EVENTO)
       │
       └────────┬─────────────────┘
                ▼
         Merge & Deduplicate
         (track sources)
                │
                ▼
         Geo + Time Filter
                │
                ▼
              Render
```

## Data Structures

### Task 1: Create Canonical Event Type (30 min)

**File:** `internal/event/event.go` (new package)

```go
package event

import "time"

// CanonicalEvent represents an event in our internal format.
// All parsers convert to this structure.
type CanonicalEvent struct {
    // Core fields
    ID          string
    Title       string
    Description string

    // Time
    StartTime   time.Time
    EndTime     time.Time

    // Location
    Latitude    float64
    Longitude   float64
    VenueName   string
    Address     string

    // Metadata
    DetailsURL  string

    // Source tracking
    Sources     []string  // ["JSON", "XML", "CSV"]
}

// SourcedEvent wraps an event with its source.
type SourcedEvent struct {
    Event  CanonicalEvent
    Source string  // "JSON", "XML", or "CSV"
}

// ParseResult tracks both successful parses and failures.
type ParseResult struct {
    Events  []SourcedEvent
    Errors  []ParseError
}

// ParseError records a single event that failed to parse.
type ParseError struct {
    Source      string  // "JSON", "XML", "CSV"
    Index       int     // Position in source data
    RawData     string  // Snippet of problematic data
    Error       error   // What went wrong
    RecoverType string  // "skipped", "partial", "defaulted"
}
```

**Tests:**
- TestCanonicalEvent_Creation
- TestSourcedEvent_Tracking

---

### Task 2: Update JSON Parser to Use Correct Field Names (45 min)

**File:** `internal/fetch/types.go`

Create separate struct for JSON-LD format:

```go
// JSONEvent represents Madrid's JSON-LD event structure.
type JSONEvent struct {
    ID          string  `json:"id"`
    Title       string  `json:"title"`
    Description string  `json:"description"`
    StartTime   string  `json:"dtstart"`
    EndTime     string  `json:"dtend"`
    Latitude    float64 `json:"latitude"`
    Longitude   float64 `json:"longitude"`
    Location    string  `json:"event-location"`
    Link        string  `json:"link"`
}

// JSONResponse wraps the Madrid API JSON-LD structure.
type JSONResponse struct {
    Context interface{} `json:"@context"`
    Graph   []JSONEvent `json:"@graph"`
}

// ToCanonical converts JSONEvent to CanonicalEvent.
func (e JSONEvent) ToCanonical() event.CanonicalEvent {
    return event.CanonicalEvent{
        ID:          e.ID,
        Title:       e.Title,
        Description: e.Description,
        // Parse dtstart to time.Time
        // Parse dtend to time.Time
        Latitude:    e.Latitude,
        Longitude:   e.Longitude,
        VenueName:   e.Location,
        DetailsURL:  e.Link,
        Sources:     []string{"JSON"},
    }
}
```

**File:** `internal/fetch/client.go`

Update `FetchJSON()` to return `event.ParseResult` with robust parsing:

```go
func (c *Client) FetchJSON(url string) event.ParseResult {
    var result event.ParseResult

    // Fetch and preprocess (existing code)
    resp, err := c.httpClient.Do(req)
    if err != nil {
        return event.ParseResult{
            Errors: []event.ParseError{{
                Source: "JSON",
                Error:  err,
            }},
        }
    }
    // ... existing fetch/preprocess code ...

    // Parse each event with individual error recovery
    for i, jsonEvent := range response.Graph {
        canonical, err := jsonEvent.ToCanonical()
        if err != nil {
            // Log parse error but continue processing other events
            result.Errors = append(result.Errors, event.ParseError{
                Source:      "JSON",
                Index:       i,
                RawData:     fmt.Sprintf("ID=%s Title=%s", jsonEvent.ID, jsonEvent.Title),
                Error:       err,
                RecoverType: "skipped",
            })
            continue
        }

        result.Events = append(result.Events, event.SourcedEvent{
            Event:  canonical,
            Source: "JSON",
        })
    }

    return result
}
```

**Key Robustness Features:**
- Each event parsed independently (one bad event doesn't kill the batch)
- Parse errors captured with context (index, snippet)
- Successful events still processed even if some fail
- Report shows: "JSON: 950/1000 events parsed, 50 failures"

**Tests:**
- TestFetchJSON_FieldMapping
- TestJSONEvent_ToCanonical
- TestFetchJSON_SourceTracking
- TestFetchJSON_PartialFailure (some events parse, some fail)
- TestFetchJSON_BadDateRecovery
- TestFetchJSON_MissingFieldRecovery

---

### Task 3: Update XML Parser for Correct Structure (45 min)

**File:** `internal/fetch/types.go`

```go
// XMLEvent represents Madrid's XML event structure.
type XMLEvent struct {
    IDEvento    string  `xml:"ID-EVENTO"`
    Titulo      string  `xml:"TITULO"`
    Descripcion string  `xml:"DESCRIPCION"`
    Fecha       string  `xml:"FECHA"`
    FechaFin    string  `xml:"FECHA-FIN"`
    Hora        string  `xml:"HORA"`
    Latitud     float64 `xml:"LATITUD"`
    Longitud    float64 `xml:"LONGITUD"`
    Instalacion string  `xml:"NOMBRE-INSTALACION"`
    Direccion   string  `xml:"DIRECCION"`
    ContentURL  string  `xml:"CONTENT-URL"`
}

// XMLResponse wraps the Madrid XML structure.
type XMLResponse struct {
    XMLName xml.Name   `xml:"Contenidos"`
    Events  []XMLEvent `xml:"contenido"`
}

// ToCanonical converts XMLEvent to CanonicalEvent.
func (e XMLEvent) ToCanonical(loc *time.Location) event.CanonicalEvent {
    startTime, _ := filter.ParseEventDateTime(e.Fecha, e.Hora, loc)
    endTime, _ := filter.ParseEventDateTime(e.FechaFin, "", loc)

    return event.CanonicalEvent{
        ID:          e.IDEvento,
        Title:       e.Titulo,
        Description: e.Descripcion,
        StartTime:   startTime,
        EndTime:     endTime,
        Latitude:    e.Latitud,
        Longitude:   e.Longitud,
        VenueName:   e.Instalacion,
        Address:     e.Direccion,
        DetailsURL:  e.ContentURL,
        Sources:     []string{"XML"},
    }
}
```

**Tests:**
- TestXMLEvent_ToCanonical
- TestFetchXML_FieldMapping
- TestXMLResponse_Parsing

---

### Task 4: Update CSV Parser for Canonical Format (30 min)

**File:** `internal/fetch/client.go`

Update `FetchCSV()` to return `[]event.SourcedEvent`:

```go
func (c *Client) FetchCSV(url string) ([]event.SourcedEvent, error) {
    // ... existing CSV parsing ...

    var result []event.SourcedEvent
    for _, row := range records[1:] {
        csvEvent := parseCSVRow(row, headerMap)
        canonical := csvEvent.ToCanonical(loc)
        result = append(result, event.SourcedEvent{
            Event:  canonical,
            Source: "CSV",
        })
    }
    return result, nil
}
```

**Tests:**
- TestFetchCSV_ToCanonical
- TestCSVEvent_EncodingConversion

---

### Task 5: Create Pipeline Orchestrator (60 min)

**File:** `internal/pipeline/pipeline.go` (new package)

```go
package pipeline

import (
    "github.com/ericphanson/madrid-events/internal/event"
    "github.com/ericphanson/madrid-events/internal/fetch"
)

// Pipeline coordinates parallel data source fetching.
type Pipeline struct {
    jsonURL string
    xmlURL  string
    csvURL  string
    client  *fetch.Client
}

// PipelineResult tracks events from all sources.
type PipelineResult struct {
    JSONEvents []event.SourcedEvent
    XMLEvents  []event.SourcedEvent
    CSVEvents  []event.SourcedEvent

    JSONError error
    XMLError  error
    CSVError  error
}

// FetchAll fetches from all three sources sequentially.
// Each source is isolated - errors in one don't affect others.
func (p *Pipeline) FetchAll() PipelineResult {
    var result PipelineResult

    // Fetch JSON (isolated - errors captured, don't crash)
    result.JSONEvents, result.JSONError = p.fetchJSONIsolated()

    // Fetch XML (isolated - JSON failure doesn't prevent this)
    result.XMLEvents, result.XMLError = p.fetchXMLIsolated()

    // Fetch CSV (isolated - previous failures don't prevent this)
    result.CSVEvents, result.CSVError = p.fetchCSVIsolated()

    return result
}

// fetchJSONIsolated fetches JSON with panic recovery.
func (p *Pipeline) fetchJSONIsolated() (events []event.SourcedEvent, err error) {
    defer func() {
        if r := recover(); r != nil {
            err = fmt.Errorf("JSON fetch panic: %v", r)
        }
    }()
    return p.client.FetchJSON(p.jsonURL)
}

// fetchXMLIsolated fetches XML with panic recovery.
func (p *Pipeline) fetchXMLIsolated() (events []event.SourcedEvent, err error) {
    defer func() {
        if r := recover(); r != nil {
            err = fmt.Errorf("XML fetch panic: %v", r)
        }
    }()
    return p.client.FetchXML(p.xmlURL)
}

// fetchCSVIsolated fetches CSV with panic recovery.
func (p *Pipeline) fetchCSVIsolated() (events []event.SourcedEvent, err error) {
    defer func() {
        if r := recover(); r != nil {
            err = fmt.Errorf("CSV fetch panic: %v", r)
        }
    }()
    return p.client.FetchCSV(p.csvURL)
}

// Merge combines events from all sources and deduplicates.
func (p *Pipeline) Merge(result PipelineResult) []event.CanonicalEvent {
    // Combine all events
    var all []event.SourcedEvent
    all = append(all, result.JSONEvents...)
    all = append(all, result.XMLEvents...)
    all = append(all, result.CSVEvents...)

    // Deduplicate by ID, tracking sources
    seen := make(map[string]*event.CanonicalEvent)

    for _, sourced := range all {
        if existing, found := seen[sourced.Event.ID]; found {
            // Event already exists, add this source
            existing.Sources = append(existing.Sources, sourced.Source)
        } else {
            // New event
            evt := sourced.Event
            seen[evt.ID] = &evt
        }
    }

    // Convert map to slice
    var merged []event.CanonicalEvent
    for _, evt := range seen {
        merged = append(merged, *evt)
    }

    return merged
}
```

**Tests:**
- TestPipeline_FetchAll_Sequential
- TestPipeline_FetchAll_ErrorIsolation (JSON failure doesn't prevent CSV)
- TestPipeline_FetchAll_PanicRecovery
- TestPipeline_Merge_Deduplication
- TestPipeline_Merge_SourceTracking
- TestPipeline_Merge_HandlesFailures

---

### Task 6: Update Build Reporting (45 min)

**File:** `internal/report/types.go`

Update to track all three sources independently:

```go
type FetchReport struct {
    JSON FetchAttempt
    XML  FetchAttempt
    CSV  FetchAttempt

    TotalDuration time.Duration
}

type MergeStats struct {
    JSONEvents     int
    XMLEvents      int
    CSVEvents      int

    TotalBeforeMerge int
    UniqueEvents     int
    Duplicates       int

    // Source coverage
    InAllThree     int  // Events found in all 3 sources
    InTwoSources   int  // Events found in 2 sources
    InOneSource    int  // Events found in only 1 source

    Duration time.Duration
}

type ProcessingReport struct {
    Merge      MergeStats
    GeoFilter  GeoFilterStats
    TimeFilter TimeFilterStats
}
```

**File:** `internal/report/markdown.go`

Update report to show:
- All three fetch attempts (not fallback chain)
- Merge statistics with source overlap
- Venn diagram-style visualization of source coverage

---

### Task 7: Update main.go Orchestration (60 min)

**File:** `cmd/buildsite/main.go`

Replace sequential fallback with parallel pipeline:

```go
func main() {
    buildReport := report.NewBuildReport()
    // ... flag parsing ...

    // Create pipeline
    pipe := pipeline.New(*jsonURL, *xmlURL, *csvURL, client)

    // Fetch all sources in parallel
    fetchStart := time.Now()
    pipeResult := pipe.FetchAll()
    buildReport.Fetching.TotalDuration = time.Since(fetchStart)

    // Track individual fetch results
    buildReport.Fetching.JSON = reportFetchAttempt("JSON", pipeResult.JSONEvents, pipeResult.JSONError)
    buildReport.Fetching.XML = reportFetchAttempt("XML", pipeResult.XMLEvents, pipeResult.XMLError)
    buildReport.Fetching.CSV = reportFetchAttempt("CSV", pipeResult.CSVEvents, pipeResult.CSVError)

    // Merge and deduplicate
    mergeStart := time.Now()
    merged := pipe.Merge(pipeResult)
    buildReport.Processing.Merge = calculateMergeStats(pipeResult, merged)
    buildReport.Processing.Merge.Duration = time.Since(mergeStart)

    // Apply filters
    filtered := applyFilters(merged, *lat, *lon, *radiusKm, now)

    // Render outputs
    // ...
}
```

---

### Task 8: Handle Snapshot Fallback (30 min)

**Strategy:** Only use snapshot if ALL three sources fail

```go
// After parallel fetch
if len(merged) == 0 && allSourcesFailed(pipeResult) {
    log.Println("All sources failed, loading snapshot...")
    snapshot, err := snapMgr.LoadSnapshot()
    // Convert snapshot to canonical format
    // ...
}

// Save snapshot from merged results
if len(merged) > 0 {
    snapMgr.SaveSnapshot(merged)
}
```

---

### Task 9: Update Tests (60 min)

**New test files:**
- `internal/event/event_test.go` - Canonical event tests
- `internal/pipeline/pipeline_test.go` - Pipeline orchestration tests
- `cmd/buildsite/main_integration_test.go` - Update for new flow

**Test scenarios:**
- All three sources succeed
- One source fails
- Two sources fail
- All three sources fail (snapshot fallback)
- Deduplication with source tracking
- Source coverage statistics

---

### Task 10: Add Data Quality Validation (45 min)

**File:** `internal/validate/validate.go` (new package)

Add defensive validation for canonical events:

```go
package validate

import "github.com/ericphanson/madrid-events/internal/event"

// ValidateEvent checks if canonical event has required fields.
// Returns error if critical data is missing or invalid.
func ValidateEvent(evt event.CanonicalEvent) error {
    var issues []string

    // Required fields
    if evt.ID == "" {
        issues = append(issues, "missing ID")
    }
    if evt.Title == "" {
        issues = append(issues, "missing title")
    }
    if evt.StartTime.IsZero() {
        issues = append(issues, "missing start time")
    }

    // Coordinate sanity checks
    if evt.Latitude != 0 || evt.Longitude != 0 {
        if evt.Latitude < -90 || evt.Latitude > 90 {
            issues = append(issues, fmt.Sprintf("invalid latitude: %.5f", evt.Latitude))
        }
        if evt.Longitude < -180 || evt.Longitude > 180 {
            issues = append(issues, fmt.Sprintf("invalid longitude: %.5f", evt.Longitude))
        }
    }

    if len(issues) > 0 {
        return fmt.Errorf("validation failed: %s", strings.Join(issues, ", "))
    }
    return nil
}

// SanitizeEvent fixes common data quality issues.
func SanitizeEvent(evt *event.CanonicalEvent) {
    // Trim whitespace
    evt.ID = strings.TrimSpace(evt.ID)
    evt.Title = strings.TrimSpace(evt.Title)
    evt.VenueName = strings.TrimSpace(evt.VenueName)

    // Fix end time if missing (use start time)
    if evt.EndTime.IsZero() && !evt.StartTime.IsZero() {
        evt.EndTime = evt.StartTime.Add(2 * time.Hour) // Default 2hr event
    }

    // Deduplicate sources
    evt.Sources = uniqueStrings(evt.Sources)
}
```

Apply in ToCanonical() methods:

```go
func (e JSONEvent) ToCanonical() (event.CanonicalEvent, error) {
    canonical := event.CanonicalEvent{
        ID:    e.ID,
        Title: e.Title,
        // ... other fields ...
    }

    // Sanitize and validate
    validate.SanitizeEvent(&canonical)
    if err := validate.ValidateEvent(canonical); err != nil {
        return event.CanonicalEvent{}, err
    }

    return canonical, nil
}
```

**Tests:**
- TestValidateEvent_RequiredFields
- TestValidateEvent_CoordinateBounds
- TestSanitizeEvent_Whitespace
- TestSanitizeEvent_DefaultEndTime

---

### Task 11: Remove Debug Logging (15 min)

Clean up temporary debug code from:
- `internal/filter/dedupe.go` - Remove debug prints
- `internal/fetch/client.go` - Remove JSON sample saving

---

## Implementation Order

**Phase 1: Foundation (2.5 hours)**
1. Task 1: Create canonical event type + ParseResult ✓
2. Task 10: Add data quality validation ✓
3. Task 2: Update JSON parser with robust parsing ✓
4. Task 3: Update XML parser with robust parsing ✓
5. Task 4: Update CSV parser with robust parsing ✓

**Phase 2: Pipeline (2 hours)**
6. Task 5: Create pipeline orchestrator ✓
7. Task 8: Handle snapshot fallback ✓

**Phase 3: Integration (2 hours)**
8. Task 6: Update build reporting ✓
9. Task 7: Update main.go orchestration ✓

**Phase 4: Testing & Cleanup (2 hours)**
10. Task 9: Update tests (including data quality tests) ✓
11. Task 11: Remove debug logging ✓

**Total Estimated Time:** 8.5 hours

---

## Success Criteria

1. ✅ All three sources fetch sequentially (polite to servers)
2. ✅ Each source parsed with correct field names
3. ✅ **Data integrity:** Single bad event doesn't crash entire source
4. ✅ **Parse recovery:** Capture errors, continue processing
5. ✅ **Validation:** Sanitize and validate all canonical events
6. ✅ Deduplication tracks which sources provided each event
7. ✅ Report shows: "JSON: 950/1000 parsed (50 errors), CSV: 1000/1000 parsed"
8. ✅ Report shows: "Event X found in JSON, CSV (2/3 sources)"
9. ✅ Build succeeds if ANY source has valid events
10. ✅ All existing tests pass
11. ✅ Build report shows detailed source statistics + parse errors

---

## Benefits

- **Robustness:** Works if 1-2 sources fail OR have partial data corruption
- **Data Quality:** Individual event failures don't kill entire batch
- **Visibility:** Know exactly which sources are working AND parse success rate
- **Recovery:** Graceful degradation with detailed error reporting
- **Validation:** Catch bad coordinates, missing fields, invalid dates
- **Debugging:** See exact events that failed with error context
- **Reporting:** "999/1000 events parsed" vs. "source totally failed"
