# Build Report Design

**Purpose:** Structured, human-readable report of what happened during site generation

## Why We Need This

**Current problem:** Minimal logging makes debugging hard
- "After filtering: 0 events" - but why?
- Which source was used?
- What failed and why?
- How long did each step take?

**Solution:** Comprehensive report showing:
- What was attempted
- What succeeded/failed
- Why failures occurred
- Performance metrics
- Data quality issues

## Report Format

### Output Location
- File: `./public/build-report.txt` (human-readable)
- Optional JSON: `./public/build-report.json` (machine-readable)

### Report Structure

```
================================================================================
MADRID EVENTS SITE BUILD REPORT
================================================================================
Build Time: 2025-10-20 01:45:32 CEST
Duration: 2.3s
Exit Status: SUCCESS
Events Generated: 42

================================================================================
1. DATA FETCHING
================================================================================

Attempt 1: JSON Source
  URL: https://datos.madrid.es/egob/.../eventos.json
  Status: FAILED
  Duration: 1.2s
  Error: decoding JSON: invalid character '\n' in string literal
  Details:
    - HTTP Status: 200 OK
    - Content-Type: application/json
    - Response Size: 1.2 MB
    - Encoding: UTF-8
    - Issue: Unescaped newlines in string fields
    - Sample error location: byte offset 45231

Attempt 2: XML Source
  URL: https://datos.madrid.es/egob/.../eventos.xml
  Status: FAILED
  Duration: 0.8s
  Error: decoding XML: expected element type <response> but have <Contenidos>
  Details:
    - HTTP Status: 200 OK
    - Content-Type: application/xml
    - Response Size: 1.5 MB
    - Root Element: <Contenidos>
    - Expected: <response>
    - Recommendation: Update XMLResponse struct mapping

Attempt 3: CSV Source
  URL: https://datos.madrid.es/egob/.../eventos.csv
  Status: SUCCESS
  Duration: 0.3s
  Details:
    - HTTP Status: 200 OK
    - Content-Type: text/csv
    - Response Size: 850 KB
    - Encoding: ISO-8859-1 (detected)
    - Delimiter: ; (semicolon)
    - Rows parsed: 1001
    - Columns: 29

Data Source Used: CSV
Total Fetch Duration: 2.3s

================================================================================
2. DATA PROCESSING
================================================================================

Deduplication (by ID-EVENTO):
  Input: 1001 events
  Duplicates removed: 0
  Output: 1001 events
  Duration: 0.01s

Geographic Filtering (Haversine):
  Reference Point: Plaza de España (40.42338, -3.71217)
  Radius: 0.35 km

  Filtered Out:
    - Missing coordinates: 94 events (9.4%)
    - Outside radius: 906 events (90.5%)

  Kept: 1 event (0.1%)
  Duration: 0.05s

  WARNING: Very few events in radius - consider increasing to 1-2km

Time Filtering (Future Events):
  Reference Time: 2025-10-20 01:45:32 CEST
  Timezone: Europe/Madrid

  Filtered Out:
    - Parse failures: 0 events (0.0%)
    - Past events: 0 events (0.0%)

  Kept: 1 event (100.0%)
  Duration: 0.02s

Total Events After Filtering: 1

================================================================================
3. DATA QUALITY ISSUES
================================================================================

Encoding Issues Detected:
  ⚠ 15 events with corrupted UTF-8 characters
    Examples:
      - Event 12800051: "Madrid Art D�co" (should be "Dèco")
      - Event 50042085: "nueva �poca" (should be "época")
    Root Cause: CSV source is ISO-8859-1, not UTF-8
    Recommendation: Add encoding conversion step

Missing Data:
  ⚠ 94 events (9.4%) missing GPS coordinates
    - Cannot be filtered geographically
    - Consider adding text-based location matching

  ⚠ 234 events (23.4%) missing HORA field
    - Treated as all-day events (00:00)

Field Truncation:
  ⚠ 12 events have DESCRIPCION > 500 chars
    - May affect display formatting

================================================================================
4. OUTPUT GENERATION
================================================================================

HTML Generation:
  Template: templates/index.tmpl.html
  Output: ./public/index.html
  Events rendered: 1
  File size: 8.2 KB
  Duration: 0.01s
  Status: SUCCESS

JSON API Generation:
  Output: ./public/events.json
  Events serialized: 1
  File size: 412 bytes
  Duration: 0.00s
  Status: SUCCESS

Snapshot Saved:
  Path: ./data/snapshot.json
  Events: 1001 (pre-filter)
  File size: 1.1 MB
  Duration: 0.02s
  Status: SUCCESS
  Purpose: Fallback for future builds if fetch fails

================================================================================
5. SUMMARY
================================================================================

Build Status: ✓ SUCCESS

Events Pipeline:
  Fetched: 1001 (from CSV)
  After dedup: 1001
  After geo filter: 1 (0.35km radius)
  After time filter: 1
  Final output: 1

Performance:
  Total duration: 2.3s
  - Fetching: 2.3s (100%)
  - Processing: 0.08s (3.5%)
  - Rendering: 0.03s (1.3%)

Warnings: 3
  1. Geographic radius very restrictive (0.35km) - only 0.1% of events kept
  2. Encoding issues detected - 15 events with corrupted characters
  3. 94 events missing coordinates - excluded from results

Recommendations:
  1. Increase -radius-km flag to 1.0-2.0 for better coverage
  2. Implement UTF-8 encoding conversion for CSV source
  3. Fix XML struct mapping to support <Contenidos> root element
  4. Add JSON preprocessing to handle unescaped newlines
  5. Consider secondary text-based filtering for events without coordinates

================================================================================
```

## Implementation Design

### Code Structure

```go
// internal/report/report.go

package report

import (
    "fmt"
    "time"
)

type BuildReport struct {
    BuildTime     time.Time
    Duration      time.Duration
    ExitStatus    string // SUCCESS, PARTIAL, FAILED

    Fetching      FetchReport
    Processing    ProcessingReport
    DataQuality   []DataQualityIssue
    Output        OutputReport

    Warnings      []Warning
    Recommendations []string
}

type FetchReport struct {
    Attempts []FetchAttempt
    SourceUsed string
    TotalDuration time.Duration
}

type FetchAttempt struct {
    Source      string  // "JSON", "XML", "CSV"
    URL         string
    Status      string  // "SUCCESS", "FAILED", "SKIPPED"
    Duration    time.Duration
    Error       string
    HTTPStatus  int
    ContentType string
    Size        int64
    Details     map[string]interface{}
}

type ProcessingReport struct {
    Deduplication DeduplicationStats
    GeoFilter     GeoFilterStats
    TimeFilter    TimeFilterStats
}

type DeduplicationStats struct {
    Input      int
    Duplicates int
    Output     int
    Duration   time.Duration
}

type GeoFilterStats struct {
    RefLat    float64
    RefLon    float64
    Radius    float64

    MissingCoords int
    OutsideRadius int
    Kept          int

    Duration  time.Duration
}

type TimeFilterStats struct {
    ReferenceTime time.Time
    Timezone      string

    ParseFailures int
    PastEvents    int
    Kept          int

    Duration      time.Duration
}

type DataQualityIssue struct {
    Type        string  // "ENCODING", "MISSING_DATA", "TRUNCATION"
    Severity    string  // "WARNING", "ERROR", "INFO"
    Count       int
    Description string
    Examples    []string
    Recommendation string
}

type OutputReport struct {
    HTML     OutputFile
    JSON     OutputFile
    Snapshot OutputFile
}

type OutputFile struct {
    Path     string
    Size     int64
    Status   string
    Duration time.Duration
}

type Warning struct {
    Message string
    Context string
}

// Generate human-readable report
func (r *BuildReport) WriteText(w io.Writer) error {
    // Implementation
}

// Generate JSON report
func (r *BuildReport) WriteJSON(w io.Writer) error {
    return json.NewEncoder(w).Encode(r)
}
```

### Integration Points

**In main.go:**

```go
func main() {
    // Initialize report
    report := report.NewBuildReport()
    report.BuildTime = time.Now()
    defer func() {
        report.Duration = time.Since(report.BuildTime)
        report.WriteReports("./public")
    }()

    // Fetch with reporting
    attempt := report.Fetching.StartAttempt("JSON", jsonURL)
    rawEvents, err := client.FetchJSON(jsonURL)
    attempt.Complete(err, rawEvents)

    // ... more steps with reporting

    // Processing with reporting
    report.Processing.Deduplication.Start()
    dedupedEvents := filter.DeduplicateByID(rawEvents)
    report.Processing.Deduplication.Complete(len(rawEvents), len(dedupedEvents))

    // Track warnings
    if report.Processing.GeoFilter.Kept < 10 {
        report.AddWarning("Very few events after geo filter - radius may be too small")
    }
}
```

## Benefits

**For Development:**
- Understand exactly what's happening at each step
- Identify performance bottlenecks
- Catch encoding/format issues early
- Validate assumptions about data

**For Debugging:**
- See full pipeline in one place
- Understand why event counts drop
- Get specific error locations (byte offsets, line numbers)
- Track data quality over time

**For Operations:**
- Monitor build health
- Alert on unusual patterns (too few events, high parse failures)
- Performance tracking
- Automated validation

**For Users:**
- Transparency about data sources
- Understand why events may be missing
- See when data was last updated

## Testing Strategy

```go
func TestBuildReport_TextOutput(t *testing.T) {
    report := &BuildReport{
        BuildTime: time.Now(),
        Duration: 2 * time.Second,
        ExitStatus: "SUCCESS",
        // ... populate test data
    }

    var buf bytes.Buffer
    err := report.WriteText(&buf)

    require.NoError(t, err)
    output := buf.String()

    // Verify structure
    assert.Contains(t, output, "MADRID EVENTS SITE BUILD REPORT")
    assert.Contains(t, output, "DATA FETCHING")
    assert.Contains(t, output, "DATA PROCESSING")
    assert.Contains(t, output, "SUMMARY")

    // Verify metrics
    assert.Contains(t, output, "Duration: 2.0s")
    assert.Contains(t, output, "Exit Status: SUCCESS")
}
```

## Future Enhancements

1. **Diff from previous build:**
   - Show change in event count
   - New events, removed events
   - Data quality improvements/regressions

2. **Interactive HTML report:**
   - Collapsible sections
   - Charts for metrics
   - Links to specific events with issues

3. **Alerting:**
   - Exit non-zero if critical issues
   - Email/webhook notifications
   - Integration with monitoring systems

4. **Historical tracking:**
   - Store reports over time
   - Trend analysis
   - Detect anomalies

## Implementation Priority

**Phase 1 (Now - for debugging):**
- Basic text report structure
- Fetch attempt tracking
- Filter statistics
- Data quality warnings
- Write to `./public/build-report.txt`

**Phase 2 (After fixes):**
- JSON output for automation
- Performance metrics
- Detailed error context
- Recommendations engine

**Phase 3 (Future):**
- HTML report
- Historical comparison
- Alerting integration
