# Multi-Venue Plaza de España Implementation Plan

**Date:** 2025-10-20
**Status:** Planning
**Reference:** docs/plans/2025-10-20-014-include-multivenue-plaza-espana.md

## Objective

Include city events that explicitly mention "Plaza de España" in their program copy, even if their canonical coordinates are outside the strict 350m radius. This captures multi-venue events (e.g., Christmas markets, Pride festivals) that include Plaza de España as one of several locations.

## Scope

- **Apply only to city events** (not cultural events)
- **Keep existing geo filtering** for plaza-core events (within 350m radius)
- **Add text-based matching** for Plaza de España mentions in Title/Venue/Address/Description
- **Time window**: Already filtered to "last weekend onwards" by existing pipeline
- **Audit tracking**: Add counters for multi-venue kept events

## Priority Assessment

**High Priority:**
- Captures important multi-venue events currently excluded
- Examples: Christmas markets (ID 93553), Pride festivals (ID 71133)
- Low implementation risk (additive, backward compatible)

## Implementation Plan

### Phase 1: Text Matching Infrastructure

#### Task 1.1: Add Text Normalization and Matching Helper (30 min)

**Goal:** Create utility function to match "Plaza de España" variants (accent-insensitive).

**Files to create:**
- `internal/filter/text.go` - Text matching utilities

**Implementation:**

```go
// internal/filter/text.go
package filter

import (
	"strings"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

// normalizeText removes accents, converts to lowercase, collapses whitespace
func normalizeText(s string) string {
	// Remove diacritics
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	result, _, _ := transform.String(t, s)

	// Lowercase
	result = strings.ToLower(result)

	// Collapse whitespace
	result = strings.Join(strings.Fields(result), " ")

	return result
}

// plazaEspanaVariants returns all normalized variants to search for
func plazaEspanaVariants() []string {
	return []string{
		"plaza de espana",
		"plaza espana",
		"pza espana",
		"pza de espana",
		"pl espana",
		"pl de espana",
		"plz espana",
		"pza. espana",
		"pza. de espana",
		"pl. espana",
		"pl. de espana",
	}
}

// MatchesPlazaEspana checks if any field mentions Plaza de España (accent-insensitive)
// Returns true if Plaza de España is mentioned in any of the provided fields
func MatchesPlazaEspana(title, venue, address, description string) bool {
	// Combine all fields into searchable text
	combined := strings.Join([]string{title, venue, address, description}, " ")

	// Normalize
	normalized := normalizeText(combined)

	// Check all variants
	variants := plazaEspanaVariants()
	for _, variant := range variants {
		if strings.Contains(normalized, variant) {
			return true
		}
	}

	return false
}
```

**Tests:**

```go
// internal/filter/text_test.go
package filter

import "testing"

func TestNormalizeText(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Plaza de España", "plaza de espana"},
		{"PLAZA   DE   ESPAÑA", "plaza de espana"},
		{"Pza. España", "pza. espana"},
		{"  extra   spaces  ", "extra spaces"},
	}

	for _, tt := range tests {
		got := normalizeText(tt.input)
		if got != tt.want {
			t.Errorf("normalizeText(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestMatchesPlazaEspana(t *testing.T) {
	tests := []struct {
		name        string
		title       string
		venue       string
		address     string
		description string
		want        bool
	}{
		{
			name:        "title_with_accent",
			title:       "Mercadillo en Plaza de España",
			want:        true,
		},
		{
			name:        "title_without_accent",
			title:       "Mercadillo en Plaza de Espana",
			want:        true,
		},
		{
			name:        "description_abbreviated",
			description: "Varios puntos: Pza. España, Sol, Cibeles",
			want:        true,
		},
		{
			name:        "venue_field",
			venue:       "Pl. de España",
			want:        true,
		},
		{
			name:        "no_mention",
			title:       "Evento en Plaza Mayor",
			description: "Cerca de Sol",
			want:        false,
		},
		{
			name:        "historical_reference",
			description: "Historia de la Plaza de España en el museo",
			want:        true, // Will match (filtering out historical refs is optional refinement)
		},
		{
			name:        "multi_venue",
			description: "Fiestas en Plaza de Pedro Zerolo, Plaza del Rey, Plaza de España, y Sol",
			want:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MatchesPlazaEspana(tt.title, tt.venue, tt.address, tt.description)
			if got != tt.want {
				t.Errorf("MatchesPlazaEspana() = %v, want %v", got, tt.want)
			}
		})
	}
}
```

---

### Phase 2: FilterResult Extension

#### Task 2.1: Extend FilterResult with Text Matching Fields (15 min)

**Goal:** Add fields to track text-based Plaza de España matching.

**Files to modify:**
- `internal/event/types.go` - Add fields to FilterResult

**Implementation:**

```go
// internal/event/types.go
type FilterResult struct {
	// Existing fields...
	HasDistrito      bool
	DistritoMatched  bool
	Distrito         string
	HasCoordinates   bool
	GPSDistanceKm    float64
	WithinRadius     bool
	TextMatched      bool   // NEW: Text-based location matching (existing)
	PlazaEspanaText  bool   // NEW: Specifically matched Plaza de España mention
	MultiVenueKept   bool   // NEW: Kept due to multi-venue Plaza de España mention
	StartDate        time.Time
	EndDate          time.Time
	DaysOld          int
	TooOld           bool
	Kept             bool
	FilterReason     string
}
```

**Notes:**
- `TextMatched` already exists (for general text matching)
- `PlazaEspanaText` tracks specific Plaza de España mentions
- `MultiVenueKept` indicates event kept via text (not geo)

---

### Phase 3: City Events Filtering Logic

#### Task 3.1: Update City Events Filtering to Include Text Matching (45 min)

**Goal:** Modify city events filtering to keep events that mention Plaza de España, even if outside radius.

**Files to modify:**
- `cmd/buildsite/main.go` - City events filtering section

**Current logic:**
```go
// Decide if kept
if !hasCoords {
	result.Kept = false
	result.FilterReason = "missing location data"
} else if !result.WithinRadius {
	result.Kept = false
	result.FilterReason = "outside GPS radius"
} else if result.TooOld {
	result.Kept = false
	result.FilterReason = "event too old"
} else {
	result.Kept = true
	result.FilterReason = "kept"
}
```

**New logic:**
```go
// Check for Plaza de España text mention (city events only)
result.PlazaEspanaText = filter.MatchesPlazaEspana(
	evt.Title,
	evt.Venue,
	evt.Address,
	evt.Description,
)

// Decide if kept (priority: missing coords -> geo/text -> too old -> kept)
if !hasCoords {
	// No coordinates - check text matching
	if result.PlazaEspanaText {
		result.Kept = true
		result.FilterReason = "kept (multi-venue: Plaza de España)"
		result.MultiVenueKept = true
	} else {
		result.Kept = false
		result.FilterReason = "missing location data"
		cityMissingCoords++
	}
} else {
	// Have coordinates - check geo first, then text
	result.GPSDistanceKm = filter.HaversineDistance(
		cfg.Filter.Latitude, cfg.Filter.Longitude,
		evt.Latitude, evt.Longitude)
	result.WithinRadius = (result.GPSDistanceKm <= cfg.Filter.RadiusKm)

	if result.WithinRadius {
		// Kept by geo (preferred)
		if result.TooOld {
			result.Kept = false
			result.FilterReason = "event too old"
			cityTooOld++
		} else {
			result.Kept = true
			result.FilterReason = "kept"
		}
	} else if result.PlazaEspanaText {
		// Outside radius but mentions Plaza de España
		if result.TooOld {
			result.Kept = false
			result.FilterReason = "event too old"
			cityTooOld++
		} else {
			result.Kept = true
			result.FilterReason = "kept (multi-venue: Plaza de España)"
			result.MultiVenueKept = true
		}
	} else {
		// Outside radius and no text match
		result.Kept = false
		result.FilterReason = "outside GPS radius"
		cityOutsideRadius++
	}
}
```

**Stats tracking:**
```go
// Add counter for multi-venue kept events
cityMultiVenueKept := 0

// In the loop, increment when result.MultiVenueKept is true
if result.MultiVenueKept {
	cityMultiVenueKept++
}

// Log after filtering
log.Printf("City events: %d kept by geo, %d kept by Plaza de España text match",
	len(filteredCityEvents) - cityMultiVenueKept, cityMultiVenueKept)
```

---

### Phase 4: Build Report Integration

#### Task 4.1: Add Multi-Venue Stats to Build Report (20 min)

**Goal:** Track and display multi-venue kept events in build report.

**Files to modify:**
- `internal/report/types.go` - Add MultiVenueKept field
- `cmd/buildsite/main.go` - Populate multi-venue stats

**Implementation:**

```go
// internal/report/types.go
type GeoFilterStats struct {
	RefLat           float64       `json:"ref_lat"`
	RefLon           float64       `json:"ref_lon"`
	Radius           float64       `json:"radius_km"`
	Input            int           `json:"input"`
	MissingCoords    int           `json:"missing_coords"`
	OutsideRadius    int           `json:"outside_radius"`
	Kept             int           `json:"kept"`
	MultiVenueKept   int           `json:"multi_venue_kept,omitempty"` // NEW: City events only
	Duration         time.Duration `json:"duration"`
}
```

**In main.go:**
```go
// Update geo filter stats for city pipeline
buildReport.CityPipeline.Filtering.GeoFilter = &report.GeoFilterStats{
	RefLat:         cfg.Filter.Latitude,
	RefLon:         cfg.Filter.Longitude,
	Radius:         cfg.Filter.RadiusKm,
	Input:          len(allCityEvents),
	MissingCoords:  cityMissingCoords,
	OutsideRadius:  cityOutsideRadius,
	Kept:           len(filteredCityEvents),
	MultiVenueKept: cityMultiVenueKept, // NEW
	Duration:       cityFilterDuration,
}
```

---

### Phase 5: Testing

#### Task 5.1: Add Integration Tests (30 min)

**Goal:** Verify multi-venue filtering works correctly.

**Test cases:**

1. **City event with Plaza de España in description, outside radius** → kept (multi-venue)
2. **City event within radius** → kept (geo, not multi-venue)
3. **City event with Plaza de España mention, but too old** → excluded
4. **City event outside radius, no Plaza de España mention** → excluded
5. **Cultural event with Plaza de España mention, outside radius** → excluded (not applied to cultural)

**Files to modify:**
- `cmd/buildsite/main_test.go` or create integration test

---

## Testing Strategy

### Unit Tests
- ✅ Text normalization (accents, case, whitespace)
- ✅ Plaza de España variant matching
- ✅ Positive matches (title, venue, address, description)
- ✅ Negative matches (no mention, different plaza)

### Integration Tests
- ✅ Multi-venue city event kept despite distance
- ✅ Geo filtering still works (within radius)
- ✅ Time filtering still works (too old excluded)
- ✅ Cultural events not affected by text matching
- ✅ Build report shows multi-venue count

### Manual Validation
- Check audit file for specific IDs (93553, 71133)
- Verify FilterResult.multi_venue_kept = true
- Verify build report stats

---

## Success Criteria

- ✅ Events within 350m radius continue to be kept via geo
- ✅ City events mentioning "Plaza de España" are kept even if outside radius
- ✅ Cultural events NOT included via text matching (city events only)
- ✅ Audit file shows `multi_venue_kept: true` for text-matched events
- ✅ Build report shows count of multi-venue kept events
- ✅ Filter reason clearly indicates "kept (multi-venue: Plaza de España)"
- ✅ All existing tests continue to pass

---

## Estimated Time

- Phase 1 (Text Matching): 30 minutes
- Phase 2 (FilterResult): 15 minutes
- Phase 3 (City Filtering): 45 minutes
- Phase 4 (Build Report): 20 minutes
- Phase 5 (Testing): 30 minutes
- **Total: ~2.5 hours**

---

## Risk Assessment

**Low Risk:**
- Additive change (no breaking changes)
- Only affects city events (cultural events unchanged)
- Existing geo filtering preserved
- Backward compatible audit format

**Mitigation:**
- Comprehensive tests
- Monitor multi-venue count in build report
- If > 30% of kept events are text-only, review heuristics

---

## Dependencies

- `golang.org/x/text` package (for Unicode normalization)
- Existing filter infrastructure
- Existing audit system

---

## Documentation Updates

- Update CLAUDE.md with multi-venue filtering behavior
- Update docs/dataflow.md with new filter reason
- Add examples to README.md if relevant

---

## Deployment

All changes are backward compatible. Deploy as normal:
1. Run full test suite
2. Build FreeBSD binary
3. Deploy to production
4. Monitor build report for multi-venue count
5. Check audit file for expected IDs (93553, 71133)

---

**Ready for implementation when approved.**
