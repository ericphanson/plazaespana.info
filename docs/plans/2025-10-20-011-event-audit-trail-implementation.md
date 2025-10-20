# Event Audit Trail Implementation Plan

> **For Claude:** Follow TDD practices. Write log in docs/logs/. Commit after each task.

**Goal:** Implement comprehensive audit trail showing all events and filtering decisions

**Architecture:** Non-destructive filtering - tag events instead of removing them, export complete audit JSON with all event data + filter results

**Tech Stack:** Go 1.21, stdlib only (encoding/json, os)

---

## Implementation Log

Create: `docs/logs/2025-10-20-event-audit-trail-implementation.md`

Track progress after each task with timestamp, changes made, and status.

---

## Task 1: Add FilterResult Type (10 min)

**Goal:** Add FilterResult struct to track filter decisions per event

**Files:**
- Modify: `internal/event/types.go` (add FilterResult struct and fields to event types)

### Step 1: Add FilterResult struct

Add to `internal/event/types.go` after the existing type definitions:

```go
// FilterResult tracks all filter decisions for a single event.
// Used for audit trail to understand why events were kept or filtered.
type FilterResult struct {
	// Location filtering - distrito
	HasDistrito     bool   `json:"has_distrito"`
	DistritoMatched bool   `json:"distrito_matched"` // if has distrito, did it match target?
	Distrito        string `json:"distrito"`

	// Location filtering - GPS
	HasCoordinates bool    `json:"has_coordinates"`
	GPSDistanceKm  float64 `json:"gps_distance_km"` // km from reference point
	WithinRadius   bool    `json:"within_radius"`

	// Location filtering - text matching (fallback)
	TextMatched bool `json:"text_matched"` // matched location keywords?

	// Time filtering
	StartDate time.Time `json:"start_date"`
	EndDate   time.Time `json:"end_date,omitempty"`
	DaysOld   int       `json:"days_old"` // days since start (negative = future)
	TooOld    bool      `json:"too_old"`  // started more than cutoff days ago?

	// Final decision
	Kept         bool   `json:"kept"`          // true = will be rendered
	FilterReason string `json:"filter_reason"` // human-readable reason
}
```

### Step 2: Add FilterResult to CulturalEvent

Find the `CulturalEvent` struct in `internal/event/types.go` and add field:

```go
type CulturalEvent struct {
	// ... existing fields ...

	FilterResult FilterResult `json:"filter_result,omitempty"`
}
```

### Step 3: Add FilterResult to CityEvent

Find the `CityEvent` struct in `internal/event/types.go` and add field:

```go
type CityEvent struct {
	// ... existing fields ...

	FilterResult FilterResult `json:"filter_result,omitempty"`
}
```

### Step 4: Verify build

```bash
go build -o build/buildsite ./cmd/buildsite
```

Expected: No errors

### Step 5: Update log and commit

Update `docs/logs/2025-10-20-event-audit-trail-implementation.md`:

```markdown
### Task 1: Add FilterResult Type
**Status**: ✅ Complete
**Time**: [timestamp]

Added FilterResult struct to track filter decisions.
Added FilterResult field to CulturalEvent and CityEvent.
```

Commit:
```bash
git add internal/event/types.go docs/logs/2025-10-20-event-audit-trail-implementation.md
git commit -m "feat: add FilterResult type for audit trail

- Added FilterResult struct with all filter decision fields
- Added FilterResult to CulturalEvent and CityEvent
- Ready for filter tagging implementation

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Task 2: Create Audit Export Module (30 min)

**Goal:** Create module to export audit JSON with all events and filter results

**Files:**
- Create: `internal/audit/export.go`
- Create: `internal/audit/export_test.go`

### Step 1: Write test for audit export

Create `internal/audit/export_test.go`:

```go
package audit

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/ericphanson/madrid-events/internal/event"
)

func TestSaveAuditJSON(t *testing.T) {
	// Setup test data
	buildTime := time.Date(2025, 10, 20, 14, 30, 0, 0, time.UTC)
	duration := 2500 * time.Millisecond

	culturalEvents := []event.CulturalEvent{
		{
			ID:         "evt1",
			Title:      "Test Event",
			VenueName:  "Test Venue",
			Distrito:   "CENTRO",
			FilterResult: event.FilterResult{
				HasDistrito:     true,
				DistritoMatched: true,
				Kept:            true,
				FilterReason:    "kept",
			},
		},
		{
			ID:         "evt2",
			Title:      "Filtered Event",
			Distrito:   "VICALVARO",
			FilterResult: event.FilterResult{
				HasDistrito:     true,
				DistritoMatched: false,
				Kept:            false,
				FilterReason:    "outside target distrito",
			},
		},
	}

	cityEvents := []event.CityEvent{
		{
			ID:    "city1",
			Title: "City Event",
			FilterResult: event.FilterResult{
				Kept:         true,
				FilterReason: "kept",
			},
		},
	}

	// Test save
	tmpFile := t.TempDir() + "/audit.json"
	err := SaveAuditJSON(culturalEvents, cityEvents, tmpFile, buildTime, duration)
	if err != nil {
		t.Fatalf("SaveAuditJSON failed: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(tmpFile); os.IsNotExist(err) {
		t.Fatal("Audit file not created")
	}

	// Verify content
	data, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("Failed to read audit file: %v", err)
	}

	var audit AuditFile
	if err := json.Unmarshal(data, &audit); err != nil {
		t.Fatalf("Failed to parse audit JSON: %v", err)
	}

	// Verify structure
	if audit.TotalEvents != 3 {
		t.Errorf("Expected 3 total events, got %d", audit.TotalEvents)
	}
	if audit.CulturalEvents.Total != 2 {
		t.Errorf("Expected 2 cultural events, got %d", audit.CulturalEvents.Total)
	}
	if audit.CulturalEvents.Kept != 1 {
		t.Errorf("Expected 1 kept cultural event, got %d", audit.CulturalEvents.Kept)
	}
	if audit.CulturalEvents.Filtered != 1 {
		t.Errorf("Expected 1 filtered cultural event, got %d", audit.CulturalEvents.Filtered)
	}
	if audit.CityEvents.Total != 1 {
		t.Errorf("Expected 1 city event, got %d", audit.CityEvents.Total)
	}

	// Verify filter breakdown
	if count, ok := audit.CulturalEvents.FilterBreakdown["outside target distrito"]; !ok || count != 1 {
		t.Errorf("Expected 1 'outside target distrito' in breakdown, got %d", count)
	}
	if count, ok := audit.CulturalEvents.FilterBreakdown["kept"]; !ok || count != 1 {
		t.Errorf("Expected 1 'kept' in breakdown, got %d", count)
	}
}
```

### Step 2: Run test to verify it fails

```bash
go test ./internal/audit -v
```

Expected: FAIL - package internal/audit not found

### Step 3: Create audit package structure

Create `internal/audit/export.go`:

```go
package audit

import (
	"encoding/json"
	"os"
	"time"

	"github.com/ericphanson/madrid-events/internal/event"
)

// AuditFile represents the complete audit trail for a build.
type AuditFile struct {
	BuildTime     time.Time     `json:"build_time"`
	BuildDuration float64       `json:"build_duration_seconds"`
	TotalEvents   int           `json:"total_events"`
	CulturalEvents AuditPipeline `json:"cultural_events"`
	CityEvents     AuditPipeline `json:"city_events"`
}

// AuditPipeline represents audit data for one pipeline.
type AuditPipeline struct {
	Total           int                    `json:"total"`
	Kept            int                    `json:"kept"`
	Filtered        int                    `json:"filtered"`
	FilterBreakdown map[string]int         `json:"filter_breakdown"`
	Events          []json.RawMessage      `json:"events"`
}

// SaveAuditJSON exports complete audit trail to JSON file.
func SaveAuditJSON(culturalEvents []event.CulturalEvent, cityEvents []event.CityEvent, path string, buildTime time.Time, duration time.Duration) error {
	audit := AuditFile{
		BuildTime:      buildTime,
		BuildDuration:  duration.Seconds(),
		TotalEvents:    len(culturalEvents) + len(cityEvents),
		CulturalEvents: processEvents(culturalEvents),
		CityEvents:     processEvents(cityEvents),
	}

	// Marshal to JSON with indentation
	data, err := json.MarshalIndent(audit, "", "  ")
	if err != nil {
		return err
	}

	// Write to file
	return os.WriteFile(path, data, 0644)
}

// processEvents converts events to audit pipeline format.
func processEvents[T any](events []T) AuditPipeline {
	pipeline := AuditPipeline{
		Total:           len(events),
		FilterBreakdown: make(map[string]int),
		Events:          make([]json.RawMessage, len(events)),
	}

	for i, evt := range events {
		// Marshal full event to JSON
		data, _ := json.Marshal(evt)
		pipeline.Events[i] = data

		// Extract filter result (using reflection-like type assertion)
		// We need to count kept vs filtered and build breakdown
		var result event.FilterResult

		// Type assertion to get FilterResult
		switch e := any(evt).(type) {
		case event.CulturalEvent:
			result = e.FilterResult
		case event.CityEvent:
			result = e.FilterResult
		}

		// Count kept vs filtered
		if result.Kept {
			pipeline.Kept++
		} else {
			pipeline.Filtered++
		}

		// Build filter breakdown
		if result.FilterReason != "" {
			pipeline.FilterBreakdown[result.FilterReason]++
		}
	}

	return pipeline
}
```

### Step 4: Run test to verify it passes

```bash
go test ./internal/audit -v
```

Expected: PASS

### Step 5: Update log and commit

Update log:

```markdown
### Task 2: Create Audit Export Module
**Status**: ✅ Complete
**Time**: [timestamp]

Created internal/audit package with SaveAuditJSON function.
Includes filter breakdown calculation and full event serialization.
Tests passing.
```

Commit:
```bash
git add internal/audit/ docs/logs/
git commit -m "feat: create audit export module

- Created internal/audit package
- SaveAuditJSON exports all events with filter results
- Calculates filter breakdown (count by reason)
- Test coverage for export functionality

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Task 3: Refactor Cultural Events Filtering (45 min)

**Goal:** Update cultural events filtering to tag events instead of removing them

**Files:**
- Modify: `cmd/buildsite/main.go:248-288` (cultural events filter loop)

### Step 1: Create backup of current filtering logic

```bash
git diff HEAD cmd/buildsite/main.go > /tmp/before-filter-refactor.patch
```

### Step 2: Replace filter loop with tagging approach

In `cmd/buildsite/main.go`, find the cultural events filtering loop (around line 248).

Replace the entire loop from:
```go
for _, evt := range merged {
    // Priority 1: Filter by distrito...
    ...
    filteredEvents = append(filteredEvents, evt)
}
```

With:
```go
// AUDIT TRAIL: Evaluate all filters, tag events (don't remove)
allCulturalEvents := make([]event.CulturalEvent, 0, len(merged))

for _, evt := range merged {
    result := event.FilterResult{}

    // Record time information
    result.StartDate = evt.StartTime
    result.EndDate = evt.EndTime
    result.DaysOld = int(now.Sub(evt.StartTime).Hours() / 24)
    cutoffWeeksAgo := now.AddDate(0, 0, -7*cfg.Filter.PastEventsWeeks)
    result.TooOld = evt.StartTime.Before(cutoffWeeksAgo)

    // Priority 1: Evaluate distrito filter
    result.HasDistrito = (evt.Distrito != "")
    result.Distrito = evt.Distrito
    if result.HasDistrito {
        result.DistritoMatched = targetDistricts[evt.Distrito]
    }

    // Priority 2: Evaluate GPS filter
    result.HasCoordinates = (evt.Latitude != 0 && evt.Longitude != 0)
    if result.HasCoordinates {
        result.GPSDistanceKm = filter.HaversineDistance(
            cfg.Filter.Latitude, cfg.Filter.Longitude,
            evt.Latitude, evt.Longitude)
        result.WithinRadius = (result.GPSDistanceKm <= cfg.Filter.RadiusKm)
    }

    // Priority 3: Evaluate text matching
    result.TextMatched = filter.MatchesLocation(
        evt.VenueName, evt.Address, evt.Description, locationKeywords)

    // Decide if kept (SAME logic as before, just tagging instead of filtering)
    if result.HasDistrito && !result.DistritoMatched {
        result.Kept = false
        result.FilterReason = "outside target distrito"
        outsideAll++
    } else if result.HasCoordinates && !result.WithinRadius {
        result.Kept = false
        result.FilterReason = "outside GPS radius"
        outsideAll++
    } else if !result.HasDistrito && !result.HasCoordinates && !result.TextMatched {
        result.Kept = false
        result.FilterReason = "no location match"
        outsideAll++
        missingBoth++
    } else if result.TooOld {
        result.Kept = false
        result.FilterReason = "event too old"
        pastEvents++
    } else {
        result.Kept = true
        result.FilterReason = "kept"
        // Count by filter method
        if result.HasDistrito {
            byDistrito++
        } else if result.HasCoordinates {
            byRadius++
            missingDistr++
        } else {
            byTextMatch++
            missingBoth++
        }
    }

    evt.FilterResult = result
    allCulturalEvents = append(allCulturalEvents, evt)
}

// Separate kept events for rendering (same behavior as before)
filteredEvents := make([]event.CulturalEvent, 0, len(allCulturalEvents))
for _, evt := range allCulturalEvents {
    if evt.FilterResult.Kept {
        filteredEvents = append(filteredEvents, evt)
    }
}
```

### Step 3: Store allCulturalEvents for audit export

After the filtering loop, add before the log.Printf statements:

```go
// Store for audit export (will be saved after city events processing)
_ = allCulturalEvents // Will use this in Task 5
```

### Step 4: Build and verify

```bash
go build -o build/buildsite ./cmd/buildsite
```

Expected: No errors

### Step 5: Run build and verify same output

```bash
./build/buildsite -config config.toml 2>&1 | grep "Cultural events after filtering"
```

Expected: Same count as before (219 events)

### Step 6: Update log and commit

Update log:

```markdown
### Task 3: Refactor Cultural Events Filtering
**Status**: ✅ Complete
**Time**: [timestamp]

Refactored cultural events filtering to use tagging approach.
All events kept in memory with FilterResult attached.
Rendering output unchanged (219 events).
```

Commit:
```bash
git add cmd/buildsite/main.go docs/logs/
git commit -m "refactor: tag cultural events instead of filtering

- Evaluate all filters for all events
- Record filter decisions in FilterResult
- Keep all events in allCulturalEvents
- Separate kept events for rendering
- Same rendering output (219 events)

Prepares for audit trail export.

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Task 4: Refactor City Events Filtering (30 min)

**Goal:** Update city events filtering to tag events instead of removing them

**Files:**
- Modify: `cmd/buildsite/main.go` (city events section, around line 407-458)

### Step 1: Find city events filtering

Locate the city events filtering section in `cmd/buildsite/main.go`.

Currently it calls:
```go
filteredCityEvents := filter.FilterCityEvents(...)
```

### Step 2: Store all city events before filtering

Before the `filter.FilterCityEvents` call, add:

```go
// AUDIT TRAIL: Keep all city events for audit, tag them
allCityEvents := make([]event.CityEvent, len(cityEvents))
copy(allCityEvents, cityEvents)

// Track filtering counts
cityFilterStart := time.Now()
beforeFilterCount := len(cityEvents)
```

### Step 3: Add filter result tagging after filtering

After the `filter.FilterCityEvents` call and before sorting, add:

```go
// Tag all city events with filter results
for i := range allCityEvents {
    evt := &allCityEvents[i]
    result := event.FilterResult{}

    // Check if this event was kept
    kept := false
    for _, keptEvt := range filteredCityEvents {
        if keptEvt.ID == evt.ID {
            kept = true
            break
        }
    }

    // Record filter decision
    result.Kept = kept
    if kept {
        result.FilterReason = "kept"
    } else {
        // City events filtered by GPS radius or time
        // Determine reason by checking event properties
        result.HasCoordinates = (evt.Latitude != 0 && evt.Longitude != 0)
        if result.HasCoordinates {
            distance := filter.HaversineDistance(
                cfg.Filter.Latitude, cfg.Filter.Longitude,
                evt.Latitude, evt.Longitude)
            result.GPSDistanceKm = distance
            result.WithinRadius = (distance <= cfg.Filter.RadiusKm)

            if !result.WithinRadius {
                result.FilterReason = "outside GPS radius"
            }
        }

        // Check time filter
        result.StartDate = evt.StartDate
        if evt.EndDate != nil {
            result.EndDate = *evt.EndDate
        }
        result.DaysOld = int(now.Sub(evt.StartDate).Hours() / 24)
        cutoffWeeksAgo := now.AddDate(0, 0, -7*cfg.Filter.PastEventsWeeks)
        result.TooOld = evt.StartDate.Before(cutoffWeeksAgo)

        if result.TooOld && result.FilterReason == "" {
            result.FilterReason = "event too old"
        }

        // Default if no specific reason identified
        if result.FilterReason == "" {
            result.FilterReason = "filtered (reason unknown)"
        }
    }

    evt.FilterResult = result
}

// Store for audit export
_ = allCityEvents // Will use in Task 5
```

### Step 4: Build and verify

```bash
go build -o build/buildsite ./cmd/buildsite
```

Expected: No errors

### Step 5: Run build and verify same output

```bash
./build/buildsite -config config.toml 2>&1 | grep "City events after filtering"
```

Expected: Same count as before (19 events)

### Step 6: Update log and commit

Update log:

```markdown
### Task 4: Refactor City Events Filtering
**Status**: ✅ Complete
**Time**: [timestamp]

Refactored city events filtering to use tagging approach.
All city events kept in memory with FilterResult attached.
Rendering output unchanged (19 events).
```

Commit:
```bash
git add cmd/buildsite/main.go docs/logs/
git commit -m "refactor: tag city events instead of filtering

- Keep all city events in allCityEvents
- Tag each with FilterResult after filtering
- Determine filter reason by checking properties
- Same rendering output (19 events)

Prepares for audit trail export.

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude <noreply@anthropic.com)"
```

---

## Task 5: Integrate Audit Export (15 min)

**Goal:** Call SaveAuditJSON to export audit trail after filtering

**Files:**
- Modify: `cmd/buildsite/main.go` (add audit export call)

### Step 1: Add audit import

At top of `cmd/buildsite/main.go`, add to imports:

```go
import (
    // ... existing imports ...
    "github.com/ericphanson/madrid-events/internal/audit"
)
```

### Step 2: Call SaveAuditJSON after both pipelines complete

Find the section after city events filtering is done (before rendering starts).

Add:

```go
// =====================================================================
// AUDIT TRAIL: Export all events with filter results
// =====================================================================
log.Println("\n=== Exporting Audit Trail ===")
auditPath := filepath.Join(cfg.Snapshot.DataDir, "audit-events.json")
auditErr := audit.SaveAuditJSON(
    allCulturalEvents,
    allCityEvents,
    auditPath,
    buildReport.BuildTime,
    time.Since(buildReport.BuildTime),
)
if auditErr != nil {
    log.Printf("Warning: Failed to save audit trail: %v", auditErr)
} else {
    auditInfo, _ := os.Stat(auditPath)
    sizeKB := float64(auditInfo.Size()) / 1024
    log.Printf("Audit trail saved: %s (%.1f KB)", auditPath, sizeKB)
    log.Printf("  Cultural: %d total (%d kept, %d filtered)",
        len(allCulturalEvents),
        len(filteredEvents),
        len(allCulturalEvents)-len(filteredEvents))
    log.Printf("  City: %d total (%d kept, %d filtered)",
        len(allCityEvents),
        len(filteredCityEvents),
        len(allCityEvents)-len(filteredCityEvents))
}
```

### Step 3: Build

```bash
go build -o build/buildsite ./cmd/buildsite
```

Expected: No errors

### Step 4: Run build and verify audit file created

```bash
./build/buildsite -config config.toml 2>&1 | grep -A 10 "Exporting Audit"
```

Expected: "Audit trail saved" message with file size

### Step 5: Verify audit JSON structure

```bash
ls -lh data/audit-events.json
jq '.cultural_events.total, .cultural_events.kept, .cultural_events.filtered' data/audit-events.json
```

Expected: File exists, ~3-5MB, correct counts

### Step 6: Update log and commit

Update log:

```markdown
### Task 5: Integrate Audit Export
**Status**: ✅ Complete
**Time**: [timestamp]

Integrated audit.SaveAuditJSON into build pipeline.
Audit file saved to data/audit-events.json.
File size ~3-5MB with all events and filter results.
```

Commit:
```bash
git add cmd/buildsite/main.go docs/logs/
git commit -m "feat: export audit trail JSON after filtering

- Call audit.SaveAuditJSON after both pipelines complete
- Export to data/audit-events.json
- Include all events with filter results
- Log audit stats (total, kept, filtered)

Audit trail now fully functional.

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude <noreply@anthropic.com)"
```

---

## Task 6: Add .gitignore for audit file (2 min)

**Goal:** Ignore audit JSON file from git (data file, not code)

**Files:**
- Modify: `.gitignore`

### Step 1: Add audit file to .gitignore

Add to `.gitignore`:

```
# Audit trail (generated each build)
data/audit-events.json
```

### Step 2: Commit

```bash
git add .gitignore docs/logs/
git commit -m "chore: ignore audit-events.json in git

Audit file is generated data, not source code.

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude <noreply@anthropic.com)"
```

---

## Task 7: Testing & Validation (20 min)

**Goal:** Verify audit trail works correctly with real data

### Step 1: Run full build

```bash
./build/buildsite -config config.toml
```

Expected: Build succeeds, audit file created

### Step 2: Check audit file size

```bash
ls -lh data/audit-events.json
```

Expected: 3-5MB file

### Step 3: Verify JSON structure

```bash
jq 'keys' data/audit-events.json
```

Expected: ["build_duration_seconds", "build_time", "city_events", "cultural_events", "total_events"]

### Step 4: Check event counts

```bash
jq '{
  cultural_total: .cultural_events.total,
  cultural_kept: .cultural_events.kept,
  cultural_filtered: .cultural_events.filtered,
  city_total: .city_events.total,
  city_kept: .city_events.kept,
  city_filtered: .city_events.filtered
}' data/audit-events.json
```

Expected: Matches build output (219 cultural kept, 19 city kept)

### Step 5: Check filter breakdown

```bash
jq '.cultural_events.filter_breakdown' data/audit-events.json
```

Expected: Shows counts by filter reason ("kept", "outside target distrito", etc.)

### Step 6: Spot check specific events

```bash
# Find an event filtered by distrito
jq '.cultural_events.events[] | select(.filter_result.kept == false and .filter_result.filter_reason == "outside target distrito") | {id, title, distrito, filter_reason: .filter_result.filter_reason}' data/audit-events.json | head -20
```

Expected: Shows events with distrito like VICALVARO, LATINA, etc.

```bash
# Find kept events
jq '.cultural_events.events[] | select(.filter_result.kept == true) | {id, title, distrito, filter_reason: .filter_result.filter_reason}' data/audit-events.json | head -20
```

Expected: Shows events with distrito CENTRO/MONCLOA-ARAVACA or within radius

### Step 7: Verify all event fields present

```bash
jq '.cultural_events.events[0] | keys' data/audit-events.json
```

Expected: All CulturalEvent fields present (id, title, description, venue_name, etc.)

### Step 8: Update log

Update log:

```markdown
### Task 7: Testing & Validation
**Status**: ✅ Complete
**Time**: [timestamp]

Verified audit trail functionality:
- Audit file created (data/audit-events.json)
- File size: ~3-5MB
- Event counts match build output
- Filter breakdown shows correct reasons
- All event fields present in JSON
- Spot checks confirm filter decisions accurate
```

### Step 9: Final commit

```bash
git add docs/logs/
git commit -m "docs: complete audit trail implementation

All tasks complete and verified:
- FilterResult type added
- Audit export module created
- Both pipelines refactored to tag events
- Audit JSON exported with full event data
- Testing confirms accuracy

Total events in audit: 1001 cultural + 1157 city
File size: ~3-5MB
All filter decisions recorded

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude <noreply@anthropic.com)"
```

---

## Success Criteria

✅ All events (kept + filtered) in audit JSON
✅ Filter results recorded for every event
✅ Complete event data included (all fields)
✅ File size 3-5MB (reasonable)
✅ Same rendering output as before (219 cultural, 19 city)
✅ No performance regression
✅ All tests passing

---

## Example Audit Trail Usage

### Debug missing event:
```bash
jq '.cultural_events.events[] | select(.title | contains("Conde Duque")) | {title, kept: .filter_result.kept, reason: .filter_result.filter_reason}' data/audit-events.json
```

### Find all events filtered by distance:
```bash
jq '.cultural_events.events[] | select(.filter_result.filter_reason == "outside GPS radius") | {title, distance: .filter_result.gps_distance_km}' data/audit-events.json
```

### Count events by filter reason:
```bash
jq '.cultural_events.filter_breakdown' data/audit-events.json
```
