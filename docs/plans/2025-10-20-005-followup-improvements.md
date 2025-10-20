# Followup Improvements Plan

**Date:** 2025-10-20
**Goal:** Enhance event filtering, display, and reporting with post-canonicalization processing

---

## Architecture Principle

**CRITICAL:** All improvements must happen AFTER canonicalization.

```
Fetch (JSON/XML/CSV) → Parse → ToCanonical() → CanonicalEvent
                                                      ↓
                                        [All processing happens here]
                                                      ↓
                                    Filter → Enrich → Render
```

This ensures a clean separation: parsers handle format-specific logic, everything else operates on uniform CanonicalEvent.

---

## Task 1: Text-Based Fallback for Missing Coordinates (60 min)

**Problem:** Events without coordinates are currently dropped. Some relevant events might be near Plaza de España but lack GPS data.

**Solution:** Add text-based location matching as fallback.

**Implementation:**

### 1.1: Add Text Matching Function

**File:** `internal/filter/text.go` (new)

```go
package filter

import "strings"

// MatchesLocation checks if event text mentions target location.
// Used as fallback when coordinates are missing.
func MatchesLocation(venueName, address, description string, keywords []string) bool {
    // Combine all text fields
    text := strings.ToLower(venueName + " " + address + " " + description)

    // Check if any keyword appears
    for _, keyword := range keywords {
        if strings.Contains(text, strings.ToLower(keyword)) {
            return true
        }
    }

    return false
}
```

### 1.2: Update main.go Filtering

**File:** `cmd/buildsite/main.go`

Update the filtering loop:

```go
// Define location keywords (Plaza de España and variations)
locationKeywords := []string{
    "plaza de españa",
    "plaza españa",
    "templo de debod",  // Nearby landmark
    "parque del oeste", // Nearby park
}

for _, evt := range merged {
    // Text-based fallback if missing coordinates
    if evt.Latitude == 0 || evt.Longitude == 0 {
        if filter.MatchesLocation(evt.VenueName, evt.Address, evt.Description, locationKeywords) {
            missingCoordsKept++
            filteredEvents = append(filteredEvents, evt)
        } else {
            missingCoords++
        }
        continue
    }

    // Geographic filtering (existing code)
    if !filter.WithinRadius(*lat, *lon, evt.Latitude, evt.Longitude, *radiusKm) {
        outsideRadius++
        continue
    }

    // ... rest of filtering
}
```

### 1.3: Update Reporting

Add new stat: `missingCoordsKept` to show how many events were kept via text matching.

**Tests:**
- TestMatchesLocation (case insensitive, partial matches)
- Integration test with fixture events missing coordinates

---

## Task 2: Add Description to Event Cards (45 min)

**Problem:** Event cards only show title, time, venue. Description provides valuable context.

**Solution:** Add truncated description to cards.

### 2.1: Update Template Event Type

**File:** `internal/render/types.go`

```go
type TemplateEvent struct {
    IDEvento          string
    Titulo            string
    StartHuman        string
    NombreInstalacion string
    ContentURL        string
    Description       string // NEW: truncated description
}
```

### 2.2: Add Truncation Helper

**File:** `internal/render/helpers.go` (new)

```go
package render

import "strings"

// TruncateText truncates text to maxChars, adding ellipsis if truncated.
func TruncateText(text string, maxChars int) string {
    if len(text) <= maxChars {
        return text
    }

    // Find last space before maxChars to avoid cutting words
    truncated := text[:maxChars]
    lastSpace := strings.LastIndex(truncated, " ")
    if lastSpace > 0 {
        truncated = truncated[:lastSpace]
    }

    return truncated + "…"
}
```

### 2.3: Update main.go Conversion

```go
for _, evt := range filteredEvents {
    templateEvents = append(templateEvents, render.TemplateEvent{
        IDEvento:          evt.ID,
        Titulo:            evt.Title,
        StartHuman:        evt.StartTime.Format("02/01/2006 15:04"),
        NombreInstalacion: evt.VenueName,
        ContentURL:        evt.DetailsURL,
        Description:       render.TruncateText(evt.Description, 150), // NEW
    })
    // ...
}
```

### 2.4: Update HTML Template

**File:** `templates/index.tmpl.html`

Add description to event cards:

```html
<article class="event-card">
    <h2><a href="{{.ContentURL}}">{{.Titulo}}</a></h2>
    <p class="event-meta">
        <time>{{.StartHuman}}</time> · <span>{{.NombreInstalacion}}</span>
    </p>
    {{if .Description}}
    <p class="event-description">{{.Description}}</p>
    {{end}}
</article>
```

### 2.5: Update CSS

**File:** `assets/site.css`

```css
.event-description {
    color: #666;
    font-size: 0.9em;
    margin-top: 0.5em;
    line-height: 1.5;
}

@media (prefers-color-scheme: dark) {
    .event-description {
        color: #aaa;
    }
}
```

**Tests:**
- TestTruncateText (short text, long text, word boundaries)

---

## Task 3: Render Build Report as HTML (60 min)

**Problem:** Build report is markdown, not accessible from the site.

**Solution:** Generate HTML version of build report and link from main page.

### 3.1: Create HTML Report Renderer

**File:** `internal/report/html.go` (new)

```go
package report

import (
    "fmt"
    "io"
    "strings"
)

// WriteHTML writes an HTML-formatted report.
func (r *BuildReport) WriteHTML(w io.Writer) error {
    var b strings.Builder

    // HTML header
    b.WriteString("<!DOCTYPE html>\n")
    b.WriteString("<html lang=\"en\">\n")
    b.WriteString("<head>\n")
    b.WriteString("    <meta charset=\"UTF-8\">\n")
    b.WriteString("    <meta name=\"viewport\" content=\"width=device-width, initial-scale=1.0\">\n")
    b.WriteString("    <title>Build Report - Madrid Events</title>\n")
    b.WriteString("    <link rel=\"stylesheet\" href=\"assets/site.css\">\n")
    b.WriteString("    <style>\n")
    b.WriteString("        .report-container { max-width: 1000px; margin: 0 auto; padding: 2rem; }\n")
    b.WriteString("        .stat-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 1rem; margin: 1rem 0; }\n")
    b.WriteString("        .stat-card { background: #f5f5f5; padding: 1rem; border-radius: 8px; }\n")
    b.WriteString("        .stat-value { font-size: 2em; font-weight: bold; color: #2563eb; }\n")
    b.WriteString("        table { width: 100%; border-collapse: collapse; margin: 1rem 0; }\n")
    b.WriteString("        th, td { text-align: left; padding: 0.75rem; border-bottom: 1px solid #ddd; }\n")
    b.WriteString("        th { background: #f9f9f9; font-weight: 600; }\n")
    b.WriteString("        @media (prefers-color-scheme: dark) {\n")
    b.WriteString("            .stat-card { background: #2a2a2a; }\n")
    b.WriteString("            th { background: #2a2a2a; }\n")
    b.WriteString("            th, td { border-bottom-color: #444; }\n")
    b.WriteString("        }\n")
    b.WriteString("    </style>\n")
    b.WriteString("</head>\n")
    b.WriteString("<body>\n")
    b.WriteString("<div class=\"report-container\">\n")

    // Title
    b.WriteString("<h1>Build Report</h1>\n")
    b.WriteString(fmt.Sprintf("<p><strong>Build Time:</strong> %s</p>\n", r.BuildTime.Format("2006-01-02 15:04:05 MST")))
    b.WriteString(fmt.Sprintf("<p><strong>Duration:</strong> %.2fs</p>\n", r.Duration.Seconds()))
    b.WriteString(fmt.Sprintf("<p><strong>Status:</strong> %s</p>\n", r.ExitStatus))

    // Key stats
    b.WriteString("<div class=\"stat-grid\">\n")
    b.WriteString("    <div class=\"stat-card\">\n")
    b.WriteString("        <div class=\"stat-label\">Events Generated</div>\n")
    b.WriteString(fmt.Sprintf("        <div class=\"stat-value\">%d</div>\n", r.EventsCount))
    b.WriteString("    </div>\n")
    b.WriteString("    <div class=\"stat-card\">\n")
    b.WriteString("        <div class=\"stat-label\">Sources Merged</div>\n")
    b.WriteString(fmt.Sprintf("        <div class=\"stat-value\">%d</div>\n", r.Processing.Merge.TotalBeforeMerge))
    b.WriteString("    </div>\n")
    b.WriteString("    <div class=\"stat-card\">\n")
    b.WriteString("        <div class=\"stat-label\">Duplicates Removed</div>\n")
    b.WriteString(fmt.Sprintf("        <div class=\"stat-value\">%d</div>\n", r.Processing.Merge.Duplicates))
    b.WriteString("    </div>\n")
    b.WriteString("</div>\n")

    // Data sources table
    b.WriteString("<h2>Data Sources</h2>\n")
    b.WriteString("<table>\n")
    b.WriteString("<thead><tr><th>Source</th><th>Status</th><th>Events</th><th>Duration</th></tr></thead>\n")
    b.WriteString("<tbody>\n")

    for _, attempt := range []FetchAttempt{r.Fetching.JSON, r.Fetching.XML, r.Fetching.CSV} {
        if attempt.Source == "" {
            continue
        }
        status := attempt.Status
        if attempt.Status == "SUCCESS" {
            status = "✅ " + status
        } else {
            status = "❌ " + status
        }
        b.WriteString(fmt.Sprintf("<tr><td>%s</td><td>%s</td><td>%d</td><td>%.2fs</td></tr>\n",
            attempt.Source, status, attempt.EventCount, attempt.Duration.Seconds()))
    }

    b.WriteString("</tbody>\n</table>\n")

    // More sections...

    b.WriteString("<p><a href=\"index.html\">← Back to Events</a></p>\n")
    b.WriteString("</div>\n</body>\n</html>\n")

    _, err := w.Write([]byte(b.String()))
    return err
}
```

### 3.2: Update main.go

```go
// Write HTML report
htmlReportPath := filepath.Join(outputDir, "build-report.html")
if f, err := os.Create(htmlReportPath); err == nil {
    buildReport.WriteHTML(f)
    f.Close()
    log.Println("HTML build report written to:", htmlReportPath)
}
```

### 3.3: Add Link in Main Page

**File:** `templates/index.tmpl.html`

Add footer link:

```html
<footer>
    <p>Data source: Ayuntamiento de Madrid – datos.madrid.es</p>
    <p><a href="build-report.html">View Build Report</a></p>
</footer>
```

---

## Task 4: Investigate Data Enrichment (Research - 45 min)

**Goal:** Determine if we can make event cards richer with additional data.

### 4.1: Check CONTENT-URL Pattern

Examine a sample of CONTENT-URL links to see if they:
- Follow a consistent format
- Contain structured data we can scrape
- Have an API alternative

### 4.2: Investigate Madrid Open Data APIs

Check if Madrid has additional APIs for:
- Event images
- Categories/tags
- Venue details (photos, full address)
- Related events

### 4.3: Check for JSON-LD or Schema.org

If CONTENT-URLs are web pages, check if they have:
- JSON-LD embedded data
- Schema.org Event markup
- OpenGraph metadata

### 4.4: Document Findings

Create `docs/enrichment-investigation.md` with:
- What additional data is available
- How to access it (API, scraping, embedded data)
- Recommendations for implementation
- Effort estimate

**DO NOT IMPLEMENT** - this is research only. Will create follow-up tasks based on findings.

---

## Task 5: Verify Post-Canonicalization Architecture (30 min)

**Goal:** Ensure all new processing happens AFTER canonicalization, not during parsing.

### 5.1: Code Review Checklist

- [ ] Text matching: operates on CanonicalEvent fields ✓
- [ ] Description truncation: operates on CanonicalEvent.Description ✓
- [ ] No parser-specific logic in filter/render packages ✓
- [ ] All enrichment (future) will use CanonicalEvent interface ✓

### 5.2: Update Architecture Diagram

**File:** `docs/architecture.md`

Create clear diagram showing:

```
┌─────────────────────────────────────────────────────────┐
│ PARSING LAYER (Format-Specific)                        │
├─────────────────────────────────────────────────────────┤
│ JSON Parser → JSONEvent.ToCanonical() → CanonicalEvent │
│ XML Parser  → XMLEvent.ToCanonical()  → CanonicalEvent │
│ CSV Parser  → CSVEvent.ToCanonical()  → CanonicalEvent │
└─────────────────────────────────────────────────────────┘
                          ↓
┌─────────────────────────────────────────────────────────┐
│ PROCESSING LAYER (Format-Agnostic)                     │
├─────────────────────────────────────────────────────────┤
│ 1. Merge & Deduplicate                                 │
│ 2. Filter (geo + text + time)                          │
│ 3. Enrich (future: images, categories, etc.)           │
│ 4. Sort & Rank                                          │
└─────────────────────────────────────────────────────────┘
                          ↓
┌─────────────────────────────────────────────────────────┐
│ RENDERING LAYER (Output-Specific)                      │
├─────────────────────────────────────────────────────────┤
│ HTML Renderer  ← CanonicalEvent                        │
│ JSON Renderer  ← CanonicalEvent                        │
│ Report Renderer ← CanonicalEvent                       │
└─────────────────────────────────────────────────────────┘
```

### 5.3: Add Tests

Create integration test verifying that enrichment can be added without touching parsers:

```go
func TestEnrichmentInterface(t *testing.T) {
    // Mock enricher that adds data to canonical events
    type Enricher interface {
        Enrich(evt *event.CanonicalEvent) error
    }

    // Should be able to enrich without knowing source format
    events := []event.CanonicalEvent{/* ... */}

    for i := range events {
        // Enrichment operates on canonical form only
        // Never needs to know if event came from JSON, XML, or CSV
    }
}
```

---

## Execution Order

1. **Task 5** (Verify arch) - Ensure foundation is correct
2. **Task 1** (Text fallback) - Core functionality
3. **Task 2** (Descriptions) - UI improvement
4. **Task 3** (HTML report) - Visibility
5. **Task 4** (Investigation) - Research for future work

**Estimated Total Time:** 4 hours
**All changes post-canonicalization:** ✓
