# Data Format Debugging & Fix Plan

**Date:** 2025-10-20
**Issue:** Multiple data format and encoding problems preventing proper event display

## Current State

**Problems:**
1. Unicode corruption: `Dèco` → `D�co`, `época` → `�poca`
2. Only 1 event showing (should be 42+ with reasonable radius)
3. JSON parsing fails: `invalid character '\n' in string literal`
4. XML parsing fails: `expected element type <response> but have <Contenidos>`
5. CSV works but has encoding issues

**Current behavior:**
- JSON fails → XML fails → CSV succeeds
- CSV parses 1001 events → 1 event after filtering (0.35km radius too restrictive)
- Unicode characters corrupted in output

---

## PHASE 0: Build Reporting Infrastructure (FIRST!)

**Goal:** Implement structured build report BEFORE investigating issues

**Why first:**
- Makes all subsequent debugging easier
- Provides visibility into each step
- Groups failures meaningfully
- Tracks timing automatically
- Shows data quality issues clearly

**See:** `docs/plans/2025-10-20-build-report-design.md` for full design

### Task 0.1: Implement Basic Report Structure

**Files to create:**
- `internal/report/report.go` - Core report types
- `internal/report/writer.go` - Text/JSON output

**Core types:**
```go
type BuildReport struct {
    BuildTime   time.Time
    Duration    time.Duration
    Fetching    FetchReport
    Processing  ProcessingReport
    DataQuality []DataQualityIssue
    Output      OutputReport
    Warnings    []Warning
}
```

**Success criteria:**
- Types defined
- Basic text writer works
- Can serialize to JSON

---

### Task 0.2: Integrate Report into main.go

**Changes:**
```go
func main() {
    report := report.NewBuildReport()
    defer report.WriteReports("./public")

    // Track each fetch attempt
    attempt := report.TrackFetch("JSON", jsonURL)
    events, err := client.FetchJSON(jsonURL)
    attempt.Complete(err, len(events))

    // Track processing
    report.TrackDeduplication(before, after)
    report.TrackGeoFilter(input, output, radius)
    report.TrackTimeFilter(input, output)

    // Track output
    report.TrackOutput("HTML", path, size)
}
```

**Success criteria:**
- Report written to `./public/build-report.txt`
- Contains all fetch attempts
- Shows filter statistics
- Includes timing info

---

### Task 0.3: Add Data Quality Checks

**Detection points:**
```go
// During CSV parsing
if !utf8.ValidString(event.Titulo) {
    report.AddDataQualityIssue(EncodingIssue{
        EventID: event.IDEvento,
        Field: "TITULO",
        Value: event.Titulo,
    })
}

// After geo filtering
if kept < input * 0.01 {  // Less than 1%
    report.AddWarning("Very restrictive radius - only %.1f%% kept", percent)
}

// After parsing
if parseFailures > input * 0.10 {  // More than 10%
    report.AddWarning("High parse failure rate: %d/%d", failures, total)
}
```

**Success criteria:**
- Encoding issues detected and reported
- Radius warnings trigger correctly
- Parse failures tracked

---

### Task 0.4: Use Report to Debug Current Issues

**Workflow:**
```bash
# Build with current code
just build-report

# Check report
cat ./public/build-report.txt

# Should show:
# - All 3 fetch attempts and failures
# - Encoding issue count and examples
# - Radius warning (0.35km too small)
# - Exact byte offsets for JSON errors
# - XML root element mismatch
```

**Success criteria:**
- Can see exactly why JSON fails
- Can see exactly why XML fails
- Can see encoding corruption examples
- Can see filter statistics

**Output from this task feeds into all other investigations!**

---

## Investigation Tasks

*Note: These tasks now use the build report for diagnosis*

### Task 1: Understand Current Radius Configuration
**Goal:** Determine why default radius reverted to 0.35km

**Steps:**
1. Check `justfile` dev command
2. Check default flag value in main.go
3. Document recommended radius for production

**Success criteria:**
- Understand where 0.35km comes from
- Document path to change default

---

### Task 2: Unicode/Encoding Root Cause Analysis
**Goal:** Identify where UTF-8 encoding breaks

**Investigation:**
1. Check if CSV source data is UTF-8
2. Verify Go's CSV reader encoding handling
3. Check JSON/HTML rendering preserves UTF-8
4. Test with known special characters

**Hypothesis:**
- CSV may not be UTF-8 (possibly ISO-8859-1/Windows-1252)
- Or CSV reader not configured for UTF-8
- Or HTML not declaring UTF-8 charset

**Success criteria:**
- Identify exact point where encoding corrupts
- Understand source encoding format

---

### Task 3: JSON Format Analysis
**Goal:** Understand why JSON parsing fails

**Investigation:**
1. Download JSON sample (first 1000 lines)
2. Identify the `\n` in string literal issue
3. Determine if it's:
   - Malformed JSON from source
   - Encoding issue (CRLF vs LF)
   - Unescaped newlines in string fields
4. Check if JSON is actually valid with `jq`

**Steps:**
```bash
# Get sample
curl -s 'URL' | head -1000 > sample.json

# Validate
jq empty sample.json 2>&1

# Check for literal newlines in strings
grep -P '"\S*\n' sample.json
```

**Success criteria:**
- Understand JSON structure
- Identify specific malformation
- Determine if preprocessing is viable

---

### Task 4: XML Format Analysis
**Goal:** Understand why XML parsing fails

**Investigation:**
1. Download XML sample (first 1000 lines)
2. Check actual root element name
3. Compare with expected structure in types.go
4. Validate XML syntax

**Steps:**
```bash
# Get sample
curl -s 'URL' | head -1000 > sample.xml

# Check root element
head -20 sample.xml | grep -E '<[A-Z]'

# Validate XML
xmllint --noout sample.xml 2>&1
```

**Expected vs Actual:**
- Code expects: `<response><event>...</event></response>`
- Likely actual: `<Contenidos>...</Contenidos>`

**Success criteria:**
- Confirm XML structure
- Map field names to RawEvent struct

---

## Fix Tasks

### Fix 1: Correct Default Radius
**Priority:** High (quick win)

**Changes:**
1. Update default `-radius-km` flag to 2.0
2. Update `justfile` dev command to use 2.0km
3. Document radius setting in README

**Files:**
- `cmd/buildsite/main.go` (flag default)
- `justfile` (dev command)
- `README.md` or `CLAUDE.md` (documentation)

**Testing:**
- `just dev` shows 42+ events
- Default build shows 42+ events
- All tests still pass

---

### Fix 2: UTF-8 Encoding Handling
**Priority:** High (user-visible corruption)

**Root cause analysis required first (Task 2)**

**Potential fixes:**
1. **If source is Latin-1/Windows-1252:**
   - Detect encoding and convert to UTF-8
   - Use `golang.org/x/text/encoding` package

2. **If HTML missing charset:**
   - Add `<meta charset="UTF-8">` to template

3. **If CSV reader issue:**
   - Configure reader for correct encoding

**Files to modify:**
- `internal/fetch/client.go` (CSV reading)
- `templates/index.tmpl.html` (charset declaration)
- Possibly add encoding detection library

**Testing:**
```bash
# Build and check specific event
./build/buildsite [flags]
grep "Madrid Art Dèco" ./public/index.html  # Should match correctly

# Check JSON
cat ./public/events.json | jq '.[0].title'  # Should show proper UTF-8
```

**Success criteria:**
- `Dèco` renders correctly (not `D�co`)
- `época` renders correctly (not `�poca`)
- All Spanish characters (á, é, í, ó, ú, ñ, ¿, ¡) render correctly

---

### Fix 3: JSON Parser
**Priority:** Medium (fallback to CSV works)

**Root cause analysis required first (Task 3)**

**Potential approaches:**

**Option A: Preprocess JSON**
```go
// Clean invalid JSON before parsing
body = bytes.ReplaceAll(body, []byte("\n"), []byte("\\n"))
// Then parse
json.Unmarshal(cleaned, &result)
```

**Option B: Lenient parsing**
- Use streaming parser
- Skip/replace invalid characters
- Log warnings for malformed fields

**Option C: Report upstream**
- Document that Madrid's JSON is malformed
- Keep CSV as primary source
- JSON/XML as best-effort fallback

**Files:**
- `internal/fetch/client.go` (FetchJSON)
- `internal/fetch/client_test.go` (add JSON malformation tests)

**Testing:**
```bash
# Test JSON parsing
go test -v ./internal/fetch/ -run TestClient_FetchJSON

# Integration test
./build/buildsite -json-url URL -xml-url "" -csv-url "" [...]
# Should either succeed or fail gracefully
```

---

### Fix 4: XML Parser
**Priority:** Medium (fallback to CSV works)

**Root cause analysis required first (Task 4)**

**Changes:**
1. Update XMLResponse struct to match actual structure
2. Map `<Contenidos>` to events list
3. Update field mappings if different from JSON/CSV

**Files:**
- `internal/fetch/types.go` (XMLResponse struct)
- `internal/fetch/client.go` (FetchXML)
- `internal/fetch/client_test.go` (XML test data)

**Example fix:**
```go
type XMLResponse struct {
    XMLName xml.Name   `xml:"Contenidos"`  // Not "response"!
    Events  []RawEvent `xml:"contenido"`   // Check actual element name
}
```

**Testing:**
```bash
# Test XML parsing
go test -v ./internal/fetch/ -run TestClient_FetchXML

# Integration test
./build/buildsite -json-url "" -xml-url URL -csv-url "" [...]
# Should parse events from XML
```

---

## Testing Strategy

### Unit Tests

**Encoding tests:**
```go
func TestCSV_UTF8Handling(t *testing.T) {
    csvData := `ID-EVENTO;TITULO
TEST-001;Madrid Art Dèco, 1925: época`

    events := parseCSV(csvData)
    if events[0].Titulo != "Madrid Art Dèco, 1925: época" {
        t.Errorf("UTF-8 not preserved: got %s", events[0].Titulo)
    }
}
```

**JSON preprocessing tests:**
```go
func TestJSON_NewlineInString(t *testing.T) {
    // Test JSON with literal newline in string field
    invalidJSON := `{"title":"Line 1\nLine 2"}`
    // Should either parse or fail gracefully
}
```

**XML structure tests:**
```go
func TestXML_ContenidosRoot(t *testing.T) {
    xmlData := `<Contenidos><contenido>...</contenido></Contenidos>`
    events, err := parseXML(xmlData)
    // Should parse successfully
}
```

### Integration Tests

**Three-source fallback:**
```go
func TestFallbackPriority(t *testing.T) {
    // Mock: JSON fails, XML fails, CSV succeeds
    // Verify CSV data is used

    // Mock: JSON fails, XML succeeds
    // Verify XML data is used

    // Mock: JSON succeeds
    // Verify JSON data is used (highest priority)
}
```

### Manual Testing Checklist

**Encoding verification:**
- [ ] Build site with actual Madrid data
- [ ] Check `public/index.html` for proper Spanish characters
- [ ] Check `public/events.json` for proper UTF-8
- [ ] View site in browser, verify no �  characters

**Data source verification:**
- [ ] Force JSON-only: `go run ./cmd/buildsite -json-url URL -xml-url "" -csv-url ""`
- [ ] Force XML-only: `go run ./cmd/buildsite -json-url "" -xml-url URL -csv-url ""`
- [ ] Force CSV-only: `go run ./cmd/buildsite -json-url "" -xml-url "" -csv-url URL`
- [ ] All three: Verify JSON tried first, falls back correctly

**Radius configuration:**
- [ ] `just dev` shows 42+ events (not 1)
- [ ] Default build (no flags) shows reasonable number of events
- [ ] `-radius-km 0.35` shows 0-1 events (confirms filter works)
- [ ] `-radius-km 10` shows 181+ events (confirms broader search)

---

## Acceptance Criteria

### Must Have (blocking issues)
✅ No unicode corruption (Dèco renders correctly)
✅ Default radius shows reasonable events (42+ not 1)
✅ At least one data source (JSON/XML/CSV) works perfectly
✅ All existing tests pass
✅ UTF-8 properly declared in HTML

### Should Have (quality improvements)
- JSON parsing works or fails gracefully with clear error
- XML parsing works or fails gracefully with clear error
- Encoding detection/conversion automatic
- Tests for all three data formats

### Nice to Have (future work)
- Contribute fixes upstream to Madrid open data portal
- Support multiple encodings automatically
- Streaming JSON parser for robustness

---

## Execution Order

**REVISED - Report-First Approach:**

1. **Phase 0: Build Reporting (1 hour) - DO THIS FIRST:**
   - Task 0.1: Implement report structure (30 min)
   - Task 0.2: Integrate into main.go (20 min)
   - Task 0.3: Add data quality checks (10 min)
   - Task 0.4: Generate first report and review (10 min)

   **Deliverable:** `./public/build-report.txt` showing all issues clearly

2. **Quick wins (30 min) - Now informed by report:**
   - Fix 1: Default radius (guided by report warnings)
   - Add UTF-8 charset to HTML template
   - Quick test with report

3. **Investigation (30 min) - Report makes this faster:**
   - Task 2: Encoding analysis (report shows examples)
   - Task 3: JSON analysis (report shows error locations)
   - Task 4: XML analysis (report shows structure mismatch)

4. **Fixes (2 hours) - Targeted by report data:**
   - Fix 2: UTF-8 encoding (based on report findings)
   - Fix 3: JSON parser (based on report error details)
   - Fix 4: XML parser (based on report structure info)

5. **Testing (1 hour) - Report validates success:**
   - Unit tests for encoding/parsing
   - Integration tests
   - Manual verification
   - Final report review (should show all SUCCESS)

---

## Open Questions

1. What encoding does Madrid's CSV actually use?
2. Is the JSON malformation consistent or intermittent?
3. What is the exact XML structure? (`<Contenidos>` vs `<response>`)
4. Should we report data quality issues to Madrid?
5. What's the right default radius for "near Plaza de España"?

---

## Success Metrics

**Before:**
- 1 event visible
- Unicode corruption in titles
- JSON/XML parsing fails
- Only CSV works

**After:**
- 42+ events visible (with 2km radius)
- Perfect UTF-8 rendering
- All 3 formats parse successfully OR fail with clear error messages
- Tests cover encoding edge cases
- Documentation explains radius configuration
