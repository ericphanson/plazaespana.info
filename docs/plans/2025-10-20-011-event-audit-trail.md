# Event Audit Trail System

**Date:** 2025-10-20
**Goal:** Create comprehensive audit trail for all events showing filtering decisions
**Status:** 📋 Planning

---

## Problem Statement

Currently, when events are filtered out during processing, we have no visibility into:

1. **Which events were filtered** - Once removed, we can't see what was excluded
2. **Why they were filtered** - Was it distrito? GPS radius? Time? Multiple reasons?
3. **What data they had** - Hard to debug "why didn't this event show up?"
4. **Filter performance** - Can't track patterns (e.g., "80% rejected by distrito")

This makes it difficult to:
- Debug user reports ("Event X should be showing but isn't")
- Tune filter criteria ("Are we being too restrictive?")
- Understand data quality issues from upstream sources
- Manually review filtering decisions

**Recent example:** We discovered 93 events with incomplete location data were being filtered out, some of which were actually relevant (e.g., "Actividades del distrito Centro"). Without an audit trail, we only found this through manual investigation.

---

## Proposed Solution

Implement **non-destructive filtering with comprehensive audit trail**:

1. **Tag events instead of removing them** - Keep all events in memory, mark which should be rendered
2. **Evaluate all filters for all events** - Record every filter decision
3. **Export complete audit JSON** - One file per build with full event data + filter results
4. **Separate rendering** - Filter for display at the end based on tags

### Architecture: Filter Tags (Non-destructive)

```
┌─────────────────────────────────────────────────────────┐
│ Current: Events removed at each filter stage            │
├─────────────────────────────────────────────────────────┤
│ Fetch → Merge → [Filter: remove] → [Filter: remove] →  │
│                  Render (219 events)                     │
└─────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────┐
│ Proposed: Events tagged, all kept in memory             │
├─────────────────────────────────────────────────────────┤
│ Fetch → Merge → [Tag all] → Save audit JSON (1001) →   │
│                  Filter for render → Render (219)       │
└─────────────────────────────────────────────────────────┘
```

---

## Data Structures

### 1. FilterResult Type

```go
// FilterResult tracks all filter decisions for a single event
type FilterResult struct {
    // Location filtering - distrito
    HasDistrito      bool
    DistritoMatched  bool   // if has distrito, did it match target?
    Distrito         string

    // Location filtering - GPS
    HasCoordinates   bool
    GPSDistance      float64 // km from reference point (Plaza de España)
    WithinRadius     bool

    // Location filtering - text matching (fallback)
    TextMatched      bool   // matched location keywords in venue/address/description?

    // Time filtering
    StartDate        time.Time
    EndDate          time.Time
    DaysOld          int    // days since start (negative = future)
    TooOld           bool   // started more than cutoff days ago?

    // Final decision
    Kept             bool   // true = will be rendered, false = filtered out
    FilterReason     string // human-readable: "outside distrito", "too old", "kept", etc.
}
```

### 2. Update Event Types

```go
// CulturalEvent (in internal/event/types.go)
type CulturalEvent struct {
    // ... existing fields ...

    FilterResult FilterResult `json:"filter_result"` // ADDED
}

// CityEvent (in internal/event/types.go)
type CityEvent struct {
    // ... existing fields ...

    FilterResult FilterResult `json:"filter_result"` // ADDED
}
```

### 3. Audit File Format

**File location:** `data/audit-events.json` (overwritten each build)

**Structure:**
```json
{
  "build_time": "2025-10-20T14:30:00Z",
  "build_duration_seconds": 2.5,
  "total_events": 2158,
  "cultural_events": {
    "total": 1001,
    "kept": 219,
    "filtered": 782,
    "filter_breakdown": {
      "outside_distrito": 750,
      "outside_radius": 0,
      "no_location_match": 0,
      "too_old": 32
    },
    "events": [
      {
        // COMPLETE event data (all fields from CulturalEvent)
        "id": "12345",
        "title": "Concierto en Plaza de España",
        "description": "Concierto de música clásica al aire libre...",
        "start_time": "2025-10-25T20:00:00Z",
        "end_time": "2025-10-25T22:00:00Z",
        "venue_name": "Plaza de España",
        "address": "Plaza de España, s/n, 28008 Madrid",
        "distrito": "CENTRO",
        "latitude": 40.42338,
        "longitude": -3.71217,
        "details_url": "https://datos.madrid.es/...",
        "sources": ["JSON", "XML"],

        // Filter decisions
        "filter_result": {
          "has_distrito": true,
          "distrito_matched": true,
          "has_coordinates": true,
          "gps_distance_km": 0.05,
          "within_radius": true,
          "text_matched": false,
          "start_date": "2025-10-25T20:00:00Z",
          "days_old": -5,
          "too_old": false,
          "kept": true,
          "filter_reason": "kept"
        }
      },
      {
        "id": "67890",
        "title": "Evento en Vicálvaro",
        "description": "...",
        "venue_name": "Centro Cultural Vicálvaro",
        "distrito": "VICALVARO",
        "latitude": 40.40275,
        "longitude": -3.60723,

        "filter_result": {
          "has_distrito": true,
          "distrito_matched": false,
          "distrito": "VICALVARO",
          "has_coordinates": true,
          "gps_distance_km": 8.5,
          "within_radius": false,
          "kept": false,
          "filter_reason": "outside target distrito"
        }
      }
    ]
  },
  "city_events": {
    "total": 1157,
    "kept": 19,
    "filtered": 1138,
    "filter_breakdown": {
      "outside_radius": 1100,
      "too_old": 38
    },
    "events": [
      // Same structure, with all CityEvent fields
    ]
  }
}
```

---

## Implementation Changes

### 1. Filtering Pipeline Refactor

**Current approach (cmd/buildsite/main.go):**
```go
for _, evt := range merged {
    if evt.Distrito != "" {
        if !targetDistricts[evt.Distrito] {
            outsideAll++
            continue  // REMOVES event from pipeline
        }
    }
    // ... more filters that remove events
    filteredEvents = append(filteredEvents, evt)
}
```

**New approach (non-destructive tagging):**
```go
// Step 1: Evaluate ALL filters, record results (but keep all events)
allEvents := make([]event.CulturalEvent, 0, len(merged))
for _, evt := range merged {
    result := FilterResult{}

    // Evaluate distrito filter
    result.HasDistrito = (evt.Distrito != "")
    result.Distrito = evt.Distrito
    result.DistritoMatched = targetDistricts[evt.Distrito]

    // Evaluate GPS filter
    result.HasCoordinates = (evt.Latitude != 0 && evt.Longitude != 0)
    if result.HasCoordinates {
        result.GPSDistance = filter.HaversineDistance(
            cfg.Filter.Latitude, cfg.Filter.Longitude,
            evt.Latitude, evt.Longitude)
        result.WithinRadius = (result.GPSDistance <= cfg.Filter.RadiusKm)
    }

    // Evaluate text matching
    result.TextMatched = filter.MatchesLocation(
        evt.VenueName, evt.Address, evt.Description, locationKeywords)

    // Evaluate time filter
    result.StartDate = evt.StartTime
    result.EndDate = evt.EndTime
    result.DaysOld = int(now.Sub(evt.StartTime).Hours() / 24)
    cutoffWeeksAgo := now.AddDate(0, 0, -7*cfg.Filter.PastEventsWeeks)
    result.TooOld = evt.StartTime.Before(cutoffWeeksAgo)

    // Decide if kept (SAME logic as before)
    if result.HasDistrito && !result.DistritoMatched {
        result.Kept = false
        result.FilterReason = "outside target distrito"
    } else if result.HasCoordinates && !result.WithinRadius {
        result.Kept = false
        result.FilterReason = "outside GPS radius"
    } else if !result.HasDistrito && !result.HasCoordinates && !result.TextMatched {
        result.Kept = false
        result.FilterReason = "no location match"
    } else if result.TooOld {
        result.Kept = false
        result.FilterReason = "event too old"
    } else {
        result.Kept = true
        result.FilterReason = "kept"
    }

    evt.FilterResult = result
    allEvents = append(allEvents, evt)  // Keep ALL events
}

// Step 2: Save audit JSON with ALL events
audit.SaveAuditJSON(allEvents, cityAllEvents, "data/audit-events.json")

// Step 3: Filter for rendering (only kept events)
keptEvents := make([]event.CulturalEvent, 0, len(allEvents))
for _, evt := range allEvents {
    if evt.FilterResult.Kept {
        keptEvents = append(keptEvents, evt)
    }
}

// Step 4: Continue with rendering (same as before)
// ... render keptEvents to HTML/JSON
```

### 2. Audit Export Module

**New file:** `internal/audit/export.go`

```go
package audit

import (
    "encoding/json"
    "os"
    "time"

    "github.com/ericphanson/madrid-events/internal/event"
)

type AuditFile struct {
    BuildTime     time.Time `json:"build_time"`
    BuildDuration float64   `json:"build_duration_seconds"`
    TotalEvents   int       `json:"total_events"`

    CulturalEvents AuditPipeline `json:"cultural_events"`
    CityEvents     AuditPipeline `json:"city_events"`
}

type AuditPipeline struct {
    Total           int                       `json:"total"`
    Kept            int                       `json:"kept"`
    Filtered        int                       `json:"filtered"`
    FilterBreakdown map[string]int            `json:"filter_breakdown"`
    Events          []json.RawMessage         `json:"events"` // Full event data
}

func SaveAuditJSON(culturalEvents []event.CulturalEvent, cityEvents []event.CityEvent, path string, buildTime time.Time, duration time.Duration) error {
    audit := AuditFile{
        BuildTime:     buildTime,
        BuildDuration: duration.Seconds(),
        TotalEvents:   len(culturalEvents) + len(cityEvents),
    }

    // Process cultural events
    audit.CulturalEvents = processEvents(culturalEvents)

    // Process city events
    audit.CityEvents = processEvents(cityEvents)

    // Write JSON
    data, err := json.MarshalIndent(audit, "", "  ")
    if err != nil {
        return err
    }

    return os.WriteFile(path, data, 0644)
}

func processEvents[T any](events []T) AuditPipeline {
    pipeline := AuditPipeline{
        Total:           len(events),
        FilterBreakdown: make(map[string]int),
        Events:          make([]json.RawMessage, len(events)),
    }

    for i, evt := range events {
        // Count kept vs filtered
        // Extract filter reason and count it
        // Marshal full event to JSON
        data, _ := json.Marshal(evt)
        pipeline.Events[i] = data
    }

    return pipeline
}
```

---

## Implementation Tasks

### Task 1: Add FilterResult Type (15 min)
- Add `FilterResult` struct to `internal/event/types.go`
- Add `FilterResult` field to `CulturalEvent`
- Add `FilterResult` field to `CityEvent`
- Add JSON tags for serialization

**Files:** `internal/event/types.go`

### Task 2: Create Audit Export Module (30 min)
- Create `internal/audit/export.go`
- Implement `AuditFile`, `AuditPipeline` structs
- Implement `SaveAuditJSON()` function
- Add filter breakdown calculation
- Write unit tests

**Files:** `internal/audit/export.go`, `internal/audit/export_test.go` (new)

### Task 3: Refactor Cultural Events Filtering (45 min)
- Update cultural events filter loop in `cmd/buildsite/main.go`
- Evaluate all filters and record results
- Set `FilterResult` fields
- Keep all events (don't filter out)
- Separate kept events for rendering

**Files:** `cmd/buildsite/main.go`

### Task 4: Refactor City Events Filtering (30 min)
- Update city events filter logic
- Record filter results for each event
- Keep all events
- Separate kept events for rendering

**Files:** `cmd/buildsite/main.go`

### Task 5: Integrate Audit Export (15 min)
- Call `SaveAuditJSON()` after filtering
- Pass all events (cultural + city)
- Save to `data/audit-events.json`
- Add error handling

**Files:** `cmd/buildsite/main.go`

### Task 6: Update Build Report (Optional, 20 min)
- Add link to audit JSON from build-report.html
- Show filter breakdown stats (how many filtered by each reason)
- Add "View Full Audit" link

**Files:** `internal/report/html.go`

### Task 7: Testing & Validation (30 min)
- Run full build, verify audit JSON created
- Check file size (~3-5MB expected)
- Verify all 1001+ cultural events present
- Verify all 1157+ city events present
- Spot-check filter results are accurate
- Test with filtered events (verify reasons correct)

---

## Design Trade-offs

### Memory Usage

**Impact:** Keeping all events in memory instead of filtering them out.

- Before: ~219 cultural events in memory after filtering
- After: ~1001 cultural events in memory (5x more)

**Analysis:** Still manageable - 1001 events × ~2KB per event = ~2MB in memory. Modern systems can easily handle this.

### Performance

**Impact:** Evaluating all filters for all events (even if early filter would have rejected).

- Before: Event fails distrito check → skip GPS/time checks
- After: Always evaluate all filters

**Analysis:** Extra computation is minimal:
- Distrito check: O(1) map lookup
- GPS check: One distance calculation
- Time check: One date comparison
- Total overhead: <1ms per event × 1000 events = ~1 second

**Acceptable:** Build currently takes ~2.5s, adding 1s is fine.

### Disk Space

**Impact:** 3-5MB JSON file per build (if kept, but we're only keeping last 1).

**Analysis:** Single file, overwritten each build. No disk space concern.

---

## Benefits

1. **Complete transparency** - See every event that came in and what happened to it
2. **Debugging made easy** - User reports missing event? Check audit JSON for exact reason
3. **Filter tuning** - Analyze filter breakdown to optimize criteria
4. **Data quality visibility** - See which events have incomplete data
5. **Manual review** - Export JSON to spreadsheet for human analysis
6. **No data loss** - Never lose information about filtered events

---

## Example Use Cases

### Use Case 1: Debug Missing Event

User: "Why isn't the concert at Conde Duque showing?"

1. Open `data/audit-events.json`
2. Search for event title or venue
3. Check `filter_result.kept` and `filter_result.filter_reason`
4. See exact coordinates, distrito, etc.
5. Understand why it was filtered

### Use Case 2: Tune Filter Criteria

Question: "Are we being too restrictive with the 60-day cutoff?"

1. Open audit JSON
2. Filter events where `filter_reason = "event too old"`
3. See how many exhibitions started 61-90 days ago
4. Review event titles/venues to assess relevance
5. Decide whether to adjust cutoff

### Use Case 3: Monitor Data Quality

Question: "How many events have missing location data?"

1. Check `filter_breakdown["no_location_match"]`
2. Filter events with `has_distrito = false` and `has_coordinates = false`
3. Review event titles/descriptions
4. Report issues to datos.madrid.es if needed

---

## Future Enhancements (Not in Scope)

- Web UI to browse audit file (searchable table)
- Track audit files over time (trend analysis)
- Alert when filter rejection rate spikes
- Compress audit JSON (gzip)
- Export to CSV for Excel analysis

---

## Success Criteria

✅ All events (kept + filtered) saved to audit JSON
✅ Filter results recorded for every event
✅ Complete event data included (all fields)
✅ File size reasonable (3-5MB)
✅ Can debug any filtering decision
✅ No performance regression (build time < 5s)
✅ Same rendering output as before (219 cultural events)
