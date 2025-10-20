# Build Report Dual Pipeline Update

**Date:** 2025-10-20
**Goal:** Update HTML build report to track both cultural and city event pipelines
**Status:** 📋 Planning

---

## Problem Statement

The HTML build report (`internal/report/`) currently only tracks the cultural events pipeline (datos.madrid.es with JSON/XML/CSV sources). After implementing the dual pipeline architecture, the build report is incomplete and doesn't show:

1. City events pipeline statistics (esmadrid.com)
2. Separate event counts for each pipeline
3. City events fetching/parsing metrics
4. City events filtering metrics

**Current limitations:**
- `FetchReport` only has JSON/XML/CSV (all cultural events)
- `ProcessingReport.Merge` assumes three sources being merged
- `EventsCount` is a single number (doesn't distinguish types)
- No section for city events pipeline in HTML output

---

## Proposed Solution

Update the build report structure to support dual pipelines with clear separation:

### 1. Update Data Structures (`internal/report/types.go`)

**Add new types:**
```go
// PipelineReport tracks a single data pipeline
type PipelineReport struct {
    Name     string // "Cultural Events" or "City Events"
    Source   string // "datos.madrid.es" or "esmadrid.com"
    Fetching PipelineFetchReport
    Merging  *MergeStats // Only for cultural events (3 sources)
    Filtering PipelineFilterReport
    EventCount int
    Duration time.Duration
}

// PipelineFetchReport tracks fetching for one pipeline
type PipelineFetchReport struct {
    Attempts []FetchAttempt  // One or more sources
    TotalDuration time.Duration
}

// PipelineFilterReport tracks filtering for one pipeline
type PipelineFilterReport struct {
    GeoFilter  *GeoFilterStats  // Optional
    TimeFilter *TimeFilterStats // Optional
    CategoryFilter *CategoryFilterStats // Optional (for city events)
    DistrictoFilter *DistrictoFilterStats // Optional (for cultural events)
}

// CategoryFilterStats tracks category-based filtering
type CategoryFilterStats struct {
    AllowedCategories []string
    Input int
    Filtered int
    Kept int
    Duration time.Duration
}

// DistrictoFilterStats tracks distrito-based filtering
type DistrictoFilterStats struct {
    AllowedDistricts []string
    Input int
    Filtered int
    Kept int
    Duration time.Duration
}
```

**Update BuildReport:**
```go
type BuildReport struct {
    BuildTime   time.Time
    Duration    time.Duration
    ExitStatus  string

    // Dual pipeline tracking
    CulturalPipeline PipelineReport
    CityPipeline     PipelineReport

    TotalEvents int // Sum of both pipelines

    DataQuality []DataQualityIssue
    Output      OutputReport

    Warnings        []string
    Recommendations []string
}
```

**Remove old types:**
- Remove `FetchReport` (replaced by `PipelineFetchReport`)
- Remove `ProcessingReport` (replaced by `PipelineFilterReport`)
- Remove `MergeStats` (move to `CulturalPipelineReport` specific field)
- Remove `DeduplicationStats` (unused)
- Keep `GeoFilterStats`, `TimeFilterStats` (used in `PipelineFilterReport`)

### 2. Update HTML Rendering (`internal/report/html.go`)

**Add sections:**
- **Pipeline Overview** - Summary card showing both pipelines side-by-side
- **Cultural Events Pipeline** - Detailed section for datos.madrid.es
  - Source: datos.madrid.es
  - Fetching: JSON (137 events) → fallback to XML if needed
  - Merging: Deduplication across 3 sources
  - Filtering: Distrito + GPS radius + Time
  - Output: 137 events
- **City Events Pipeline** - Detailed section for esmadrid.com
  - Source: esmadrid.com
  - Fetching: XML only (1,158 events)
  - Parsing: Nested extradata extraction
  - Filtering: GPS radius + Category + Time
  - Output: 19 events

**Visual design:**
- Use same color scheme as main site (purple for cultural, orange for city)
- Pipeline comparison table showing side-by-side metrics
- Collapsible sections for detailed stats

### 3. Update main.go Population Logic

**Cultural pipeline tracking:**
```go
buildReport.CulturalPipeline = report.PipelineReport{
    Name: "Cultural Events",
    Source: "datos.madrid.es",
    Fetching: report.PipelineFetchReport{
        Attempts: []report.FetchAttempt{
            createFetchAttempt("JSON", ...),
            createFetchAttempt("XML", ...),
            createFetchAttempt("CSV", ...),
        },
        TotalDuration: fetchDuration,
    },
    Filtering: report.PipelineFilterReport{
        DistrictoFilter: &report.DistrictoFilterStats{...},
        GeoFilter: &report.GeoFilterStats{...},
        TimeFilter: &report.TimeFilterStats{...},
    },
    EventCount: len(filteredEvents),
    Duration: culturalPipelineDuration,
}
```

**City pipeline tracking:**
```go
buildReport.CityPipeline = report.PipelineReport{
    Name: "City Events",
    Source: "esmadrid.com",
    Fetching: report.PipelineFetchReport{
        Attempts: []report.FetchAttempt{
            createFetchAttempt("XML", cfg.CityEvents.XMLURL, ...),
        },
        TotalDuration: cityFetchDuration,
    },
    Filtering: report.PipelineFilterReport{
        GeoFilter: &report.GeoFilterStats{...},
        TimeFilter: &report.TimeFilterStats{...},
        CategoryFilter: &report.CategoryFilterStats{
            AllowedCategories: cfg.Filter.CityCategories,
            Input: len(rawCityEvents),
            Kept: len(filteredCityEvents),
        },
    },
    EventCount: len(filteredCityEvents),
    Duration: cityPipelineDuration,
}
```

---

## Implementation Tasks

### Task 1: Update Data Structures (30 min)
- Add `PipelineReport`, `PipelineFetchReport`, `PipelineFilterReport` to `types.go`
- Add `CategoryFilterStats`, `DistrictoFilterStats` to `types.go`
- Update `BuildReport` struct with dual pipeline fields
- Keep legacy fields for backward compatibility
- Write tests for new structs

**Files:** `internal/report/types.go`, `internal/report/types_test.go` (new)

### Task 2: Update HTML Rendering (45 min)
- Add pipeline overview section showing both pipelines side-by-side
- Add cultural pipeline detailed section
- Add city pipeline detailed section
- Update CSS for dual pipeline styling (purple/orange accents)
- Add collapsible sections for detailed stats
- Update existing sections to work with new structure

**Files:** `internal/report/html.go`

### Task 3: Update main.go Population Logic (30 min)
- Track cultural pipeline timing separately
- Track city pipeline timing separately
- Populate `CulturalPipeline` struct with fetch/filter stats
- Populate `CityPipeline` struct with fetch/filter stats
- Keep `EventsCount` as total for backward compat
- Add tracking for distrito filtering (cultural events)
- Add tracking for category filtering (city events - currently not implemented, note as TODO)

**Files:** `cmd/buildsite/main.go`

### Task 4: Testing & Verification (20 min)
- Run full build and generate HTML report
- Verify both pipelines appear in report
- Verify metrics are accurate
- Check responsive design
- Verify dark mode works
- Test with no city events (fallback case)

### Task 5: Documentation (15 min)
- Update CLAUDE.md build report section
- Add example report screenshot to docs
- Document dual pipeline structure

---

## Design Mockup

### Pipeline Overview Section

```
┌─────────────────────────────────────────────────────────────┐
│ Build Summary                                                │
├─────────────────────────────────────────────────────────────┤
│ Duration: 2.52s    Status: ✅ SUCCESS    Total Events: 156  │
└─────────────────────────────────────────────────────────────┘

┌──────────────────────────┬──────────────────────────────────┐
│  Cultural Events (🎭)    │  City Events (🎉)                │
├──────────────────────────┼──────────────────────────────────┤
│  Source: datos.madrid.es │  Source: esmadrid.com            │
│  Events: 137             │  Events: 19                      │
│  Duration: 1.58s         │  Duration: 0.94s                 │
│  Status: ✅ SUCCESS      │  Status: ✅ SUCCESS              │
└──────────────────────────┴──────────────────────────────────┘
```

### Cultural Events Pipeline (Detailed)

```
┌─────────────────────────────────────────────────────────────┐
│ Cultural Events Pipeline (datos.madrid.es)                  │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│ 📡 Data Fetching                                             │
│   ✅ JSON: 1,055 events (142ms)                             │
│   ⏭️  XML: Skipped (JSON succeeded)                         │
│   ⏭️  CSV: Skipped (JSON succeeded)                         │
│                                                              │
│ 🔄 Deduplication                                             │
│   Input: 1,055 events                                        │
│   Duplicates: 64 (6.1%)                                      │
│   Output: 991 unique events                                  │
│                                                              │
│ 🗺️  Geographic Filtering                                     │
│   Input: 991 events                                          │
│   Distrito filter: 156 events in CENTRO/MONCLOA-ARAVACA     │
│   Radius filter: 137 events within 0.35km of Plaza          │
│   Missing coords: 835 events (84.3%)                         │
│                                                              │
│ ⏰ Time Filtering                                             │
│   Input: 137 events                                          │
│   Past events: 0 events removed                              │
│   Output: 137 events                                         │
└─────────────────────────────────────────────────────────────┘
```

### City Events Pipeline (Detailed)

```
┌─────────────────────────────────────────────────────────────┐
│ City Events Pipeline (esmadrid.com)                         │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│ 📡 Data Fetching                                             │
│   ✅ XML: 1,158 events (871ms)                              │
│   Parse errors: 1 event (0.1%)                               │
│                                                              │
│ 🗺️  Geographic Filtering                                     │
│   Input: 1,157 events                                        │
│   Radius filter: 24 events within 0.35km of Plaza           │
│   Missing coords: 0 events (0%)                              │
│                                                              │
│ 🏷️  Category Filtering                                       │
│   Input: 24 events                                           │
│   Note: No category filter configured (all categories kept)  │
│   Output: 24 events                                          │
│                                                              │
│ ⏰ Time Filtering                                             │
│   Input: 24 events                                           │
│   Past events: 5 events removed                              │
│   Output: 19 events                                          │
└─────────────────────────────────────────────────────────────┘
```

---

## Backward Compatibility

**Decision: ❌ BREAK COMPATIBILITY**

- Remove all legacy fields (`EventsCount`, `Fetching`, `Processing`, `MergeStats`, etc.)
- Replace with clean dual pipeline structure
- Simpler, cleaner code
- No migration path needed

**Requirement:** Just ensure `just build` works - that's all we need!

---

## Success Criteria

- [ ] Build report shows both cultural and city pipelines
- [ ] Metrics are accurate for both pipelines
- [ ] Visual distinction between pipelines (colors/icons)
- [ ] HTML report is responsive and works in dark mode
- [ ] Report shows correct event counts (137 cultural + 19 city = 156 total)
- [ ] Report shows performance metrics (2.52s total, breakdown per pipeline)
- [ ] Fallback case handled (if city pipeline fails, report still generated)
- [ ] Tests pass
- [ ] Documentation updated

---

## Estimated Time

- Task 1: 30 minutes (data structures)
- Task 2: 45 minutes (HTML rendering)
- Task 3: 30 minutes (main.go population)
- Task 4: 20 minutes (testing)
- Task 5: 15 minutes (documentation)

**Total:** ~2.5 hours

---

## Notes

- **Build report should ALWAYS be generated** - not behind a flag
- Output path: `/workspace/public/build-report.html`
- Report is written alongside HTML/JSON output every build
- Useful for debugging and understanding pipeline performance

---

## Follow-up Enhancements (Out of Scope)

- Add chart.js for visual graphs (event distribution over time, sources breakdown)
- Add historical comparison (compare current build to previous builds)
- Add detailed parse error list (which events failed to parse from esmadrid)
- Add data quality metrics per pipeline
- Export report as JSON for programmatic access
