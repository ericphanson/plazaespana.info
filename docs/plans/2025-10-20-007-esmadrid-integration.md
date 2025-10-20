# ESMadrid.com Integration Plan

**Date:** 2025-10-20
**Goal:** Add esmadrid.com tourism feed as a second, parallel event pipeline
**Status:** 📋 Planning

---

## Executive Summary

Add esmadrid.com feed to capture city-wide festivals and special events that datos.madrid.es doesn't include (Plaza de España outdoor events, gaming tournaments, seasonal festivals). Implement as a **separate pipeline** to maintain architectural clarity.

**Key Decision:** Two independent pipelines rather than forced canonicalization
- Pipeline 1: datos.madrid.es → Cultural events (museums, theaters, libraries)
- Pipeline 2: esmadrid.com → City events (festivals, outdoor events, gaming)

---

## Architecture Overview

### Current State (Single Pipeline)
```
datos.madrid.es (JSON/XML/CSV)
  ↓ fetch/client.go
  ↓ parse to event.CanonicalEvent
  ↓ filter by distrito/GPS
  ↓ render to HTML/JSON
  → public/index.html, public/events.json
```

### Target State (Dual Pipeline)
```
┌─────────────────────────────────────────────────────────┐
│ Pipeline 1: Cultural Events (datos.madrid.es)           │
├─────────────────────────────────────────────────────────┤
│ JSON/XML/CSV → fetch/client.go                          │
│               → event.CulturalEvent                      │
│               → filter by distrito/GPS                   │
│               → []CulturalEvent                          │
└─────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────┐
│ Pipeline 2: City Events (esmadrid.com)                   │
├─────────────────────────────────────────────────────────┤
│ XML → fetch/esmadrid.go                                  │
│     → event.CityEvent                                    │
│     → filter by GPS/category                             │
│     → []CityEvent                                        │
└─────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────┐
│ Render Layer (handles both)                              │
├─────────────────────────────────────────────────────────┤
│ render/html.go → Sections for each type                  │
│ render/json.go → Separate arrays or tagged events        │
└─────────────────────────────────────────────────────────┘
```

---

## Configuration (TOML)

**File:** `config.toml`

```toml
[cultural_events]
# datos.madrid.es cultural programming
json_url = "https://datos.madrid.es/egob/catalogo/300107-0-agenda-actividades-eventos.json"
xml_url = "https://datos.madrid.es/egob/catalogo/300107-0-agenda-actividades-eventos.xml"
csv_url = "https://datos.madrid.es/egob/catalogo/300107-0-agenda-actividades-eventos.csv"

[city_events]
# esmadrid.com tourism/city events
xml_url = "https://www.esmadrid.com/opendata/agenda_v1_es.xml"

[filter]
# Plaza de España coordinates
latitude = 40.42338
longitude = -3.71217
radius_km = 0.35

# Distrito filtering
distritos = ["CENTRO", "MONCLOA-ARAVACA"]

# Time filtering
past_events_weeks = 2  # Exclude events started >2 weeks ago

[output]
html_path = "public/index.html"
json_path = "public/events.json"

[snapshot]
data_dir = "data"

[server]
# For development only
port = 8080
```

---

## Implementation Tasks

### Phase 1: Configuration & Foundation (Tasks 1-3)

#### Task 1: Create TOML Configuration System
**Estimated:** 30 minutes
**Files:** `config.toml`, `internal/config/config.go`

**Subtasks:**
1.1. Create `config.toml` in project root with URLs and settings
1.2. Create `internal/config/config.go` with structs:
```go
type Config struct {
    CulturalEvents CulturalEventsConfig
    CityEvents     CityEventsConfig
    Filter         FilterConfig
    Output         OutputConfig
    Snapshot       SnapshotConfig
}
```
1.3. Add TOML parsing using `github.com/BurntSushi/toml`
1.4. Add validation (URLs not empty, coordinates valid, etc.)
1.5. Write tests for config parsing and validation

**Acceptance:**
- [ ] Config file loads successfully
- [ ] All URLs and settings accessible
- [ ] Invalid config returns clear errors
- [ ] Tests pass

---

#### Task 2: Refactor Existing Event Type
**Estimated:** 20 minutes
**Files:** `internal/event/event.go` → `internal/event/cultural.go`

**Subtasks:**
2.1. Rename `CanonicalEvent` to `CulturalEvent`
2.2. Add `EventType() string` method returning "cultural"
2.3. Update all references in existing code
2.4. Ensure all existing tests still pass

**Acceptance:**
- [ ] `CulturalEvent` replaces `CanonicalEvent`
- [ ] All 22 existing tests pass
- [ ] No functional changes to current pipeline

---

#### Task 3: Create City Event Type
**Estimated:** 30 minutes
**Files:** `internal/event/city.go`, `internal/event/city_test.go`

**Subtasks:**
3.1. Create `CityEvent` struct:
```go
type CityEvent struct {
    ID          string
    Title       string
    Description string
    StartDate   time.Time
    EndDate     time.Time
    Venue       string
    Address     string
    Latitude    float64
    Longitude   float64
    Category    string
    Subcategory string
    WebURL      string
    ImageURL    string
    Price       string
}
```
3.2. Add `EventType() string` method returning "city"
3.3. Add `Distance(lat, lon float64) float64` method (reuse geo.go)
3.4. Write tests for CityEvent creation and methods

**Acceptance:**
- [ ] `CityEvent` struct defined with all fields
- [ ] Methods implemented
- [ ] Tests pass

---

### Phase 2: ESMadrid Fetching & Parsing (Tasks 4-6)

#### Task 4: Create ESMadrid XML Parser
**Estimated:** 45 minutes
**Files:** `internal/fetch/esmadrid.go`, `internal/fetch/esmadrid_test.go`

**Subtasks:**
4.1. Create `EsmadridEvent` raw struct matching XML schema:
```go
type EsmadridService struct {
    ID              string `xml:"id,attr"`
    UpdateDate      string `xml:"fechaActualizacion,attr"`
    Name            string `xml:"basicData>name"`
    Title           string `xml:"basicData>title"`
    Body            string `xml:"basicData>body"`
    Web             string `xml:"basicData>web"`
    VenueID         string `xml:"basicData>idrt"`
    VenueName       string `xml:"basicData>nombrert"`
    Address         string `xml:"geoData>address"`
    Latitude        string `xml:"geoData>latitude"`
    Longitude       string `xml:"geoData>longitude"`
    ImageURL        string `xml:"multimedia>media>url"`
    Category        string // Extract from extradata>categorias
    Subcategory     string // Extract from extradata>categorias>subcategorias
    Price           string // Extract from extradata "Servicios de pago"
    StartDate       string // Extract from extradata>fechas>rango>inicio
    EndDate         string // Extract from extradata>fechas>rango>fin
}
```
4.2. Implement XML parsing (handle CDATA, nested extradata)
4.3. Implement `ToCityEvent() (*event.CityEvent, error)` method
4.4. Write tests with sample XML fixtures

**Acceptance:**
- [ ] Parses esmadrid XML successfully
- [ ] Extracts all key fields (including nested extradata)
- [ ] Converts to CityEvent correctly
- [ ] Tests pass with fixture data

---

#### Task 5: Create ESMadrid HTTP Client
**Estimated:** 30 minutes
**Files:** `internal/fetch/esmadrid.go`, `internal/fetch/esmadrid_test.go`

**Subtasks:**
5.1. Add `FetchEsmadridEvents(url string) ([]EsmadridService, error)` function
5.2. Set User-Agent header (reuse pattern from client.go)
5.3. Handle HTTP errors and timeouts
5.4. Parse XML response body
5.5. Write tests with httptest mock server

**Acceptance:**
- [ ] Fetches from esmadrid.com URL
- [ ] Sets proper User-Agent
- [ ] Handles network errors gracefully
- [ ] Tests pass with mocked responses

---

#### Task 6: Implement City Event Filtering
**Estimated:** 30 minutes
**Files:** `internal/filter/city.go`, `internal/filter/city_test.go`

**Subtasks:**
6.1. Create `FilterCityEvents()` function:
```go
func FilterCityEvents(
    events []event.CityEvent,
    centerLat, centerLon, radiusKM float64,
    categories []string,
    pastWeeks int,
) []event.CityEvent
```
6.2. Implement GPS radius filtering (reuse geo.go)
6.3. Implement category filtering ("Eventos de ciudad", etc.)
6.4. Implement time filtering (exclude old events)
6.5. Write tests with known events

**Acceptance:**
- [ ] Filters by GPS radius correctly
- [ ] Filters by category correctly
- [ ] Filters by time correctly
- [ ] Tests pass

---

### Phase 3: Pipeline Integration (Tasks 7-9)

#### Task 7: Create Dual Pipeline Orchestrator
**Estimated:** 45 minutes
**Files:** `cmd/buildsite/main.go`

**Subtasks:**
7.1. Load config from TOML file
7.2. Run cultural events pipeline (existing):
   - Fetch datos.madrid.es (JSON/XML/CSV)
   - Parse and merge
   - Filter by distrito/GPS
   - Sort chronologically
7.3. Run city events pipeline (new):
   - Fetch esmadrid.com (XML)
   - Parse to CityEvent
   - Filter by GPS/category
   - Sort chronologically
7.4. Pass both event lists to renderer
7.5. Update logging to show both pipeline stats

**Acceptance:**
- [ ] Both pipelines run successfully
- [ ] Each pipeline independently filters and sorts
- [ ] Logs show stats for both (e.g., "Cultural: 142, City: 15")
- [ ] No conflicts between pipelines

---

#### Task 8: Update HTML Rendering
**Estimated:** 45 minutes
**Files:** `internal/render/html.go`, `internal/render/types.go`, `templates/index.tmpl.html`

**Subtasks:**
8.1. Update `TemplateData` struct:
```go
type TemplateData struct {
    CulturalEvents []TemplateEvent
    CityEvents     []TemplateEvent
    UpdateTime     string
    TotalEvents    int
}
```
8.2. Update HTML template with two sections:
   - "Cultural Events" section
   - "City Festivals & Special Events" section
8.3. Add visual distinction (icons, colors, badges)
8.4. Update atomic write to handle new template
8.5. Write test for dual-section rendering

**Acceptance:**
- [ ] HTML shows both event types in separate sections
- [ ] Events clearly labeled by type
- [ ] Responsive design maintained
- [ ] Tests pass

---

#### Task 9: Update JSON API Output
**Estimated:** 30 minutes
**Files:** `internal/render/json.go`, `internal/render/types.go`

**Subtasks:**
9.1. Update JSON structure:
```json
{
  "cultural_events": [...],
  "city_events": [...],
  "meta": {
    "update_time": "...",
    "total_cultural": 142,
    "total_city": 15
  }
}
```
9.2. Or tagged approach:
```json
{
  "events": [
    {"type": "cultural", ...},
    {"type": "city", ...}
  ]
}
```
9.3. Update atomic write
9.4. Write tests

**Acceptance:**
- [ ] JSON includes both event types
- [ ] Schema clearly distinguishes types
- [ ] Valid JSON output
- [ ] Tests pass

---

### Phase 4: Testing & Refinement (Tasks 10-12)

#### Task 10: Integration Testing
**Estimated:** 30 minutes
**Files:** `cmd/buildsite/main_integration_test.go`

**Subtasks:**
10.1. Create integration test with mock servers for both feeds
10.2. Test full dual pipeline execution
10.3. Verify both event types appear in output
10.4. Test fallback behavior (if esmadrid fails, cultural still works)
10.5. Test snapshot behavior for both pipelines

**Acceptance:**
- [ ] Integration test runs both pipelines
- [ ] Outputs contain both event types
- [ ] Fallback works if one source fails
- [ ] All tests pass

---

#### Task 11: Update CLI Flags & Help
**Estimated:** 20 minutes
**Files:** `cmd/buildsite/main.go`

**Subtasks:**
11.1. Add `-config` flag for TOML file path (default: `config.toml`)
11.2. Keep backward compatibility with individual URL flags
11.3. Update help text
11.4. Add version flag with new "dual pipeline" mention

**Acceptance:**
- [ ] `-config` flag works
- [ ] Backward compatible with old flags
- [ ] Help text is clear
- [ ] Version shows "dual pipeline" support

---

#### Task 12: Documentation & Examples
**Estimated:** 30 minutes
**Files:** `README.md`, `docs/design.md`, `config.toml.example`

**Subtasks:**
12.1. Update README with dual pipeline explanation
12.2. Add config.toml.example with comments
12.3. Update design.md with architecture diagrams
12.4. Add examples of running with config file
12.5. Document new JSON output schema

**Acceptance:**
- [ ] README explains dual pipeline clearly
- [ ] Example config file provided
- [ ] Design doc updated
- [ ] Examples work as documented

---

### Phase 5: Deployment Preparation (Tasks 13-15)

#### Task 13: Update Build & Deploy Scripts
**Estimated:** 20 minutes
**Files:** `justfile`, `scripts/build-freebsd.sh`, `ops/deploy-notes.md`

**Subtasks:**
13.1. Update justfile with config file handling
13.2. Add `just config` command to validate config
13.3. Update FreeBSD build to include config.toml
13.4. Update deploy notes with config setup instructions

**Acceptance:**
- [ ] `just config` validates TOML
- [ ] Build includes config file
- [ ] Deploy instructions updated

---

#### Task 14: Update Firewall for ESMadrid
**Estimated:** 5 minutes
**Files:** `.devcontainer/init-firewall.sh` (already done!)

**Subtasks:**
14.1. Verify esmadrid.com and www.esmadrid.com are allowed ✅ DONE
14.2. Test connectivity ✅ DONE
14.3. Commit firewall changes

**Acceptance:**
- [x] esmadrid.com accessible ✅ VERIFIED
- [ ] Changes committed

---

#### Task 15: Final End-to-End Test
**Estimated:** 30 minutes
**Files:** All

**Subtasks:**
15.1. Build FreeBSD binary
15.2. Run with live datos.madrid.es and esmadrid.com feeds
15.3. Verify Plaza de España events appear (ice rink, Christmas ball, etc.)
15.4. Verify cultural events still work (142 events)
15.5. Verify HTML and JSON output
15.6. Check performance (should be <5 seconds for both pipelines)

**Acceptance:**
- [ ] FreeBSD binary builds successfully
- [ ] Both pipelines fetch live data
- [ ] Plaza de España city events visible
- [ ] Cultural events unchanged
- [ ] Build time reasonable (<10 sec)
- [ ] All 22+ tests pass

---

## Task Summary

**Total Tasks:** 15
**Estimated Time:** ~6-7 hours
**Phases:** 5

**Phase Breakdown:**
- Phase 1 (Config & Foundation): 1h 20min - Tasks 1-3
- Phase 2 (ESMadrid Integration): 1h 45min - Tasks 4-6
- Phase 3 (Pipeline Integration): 2h - Tasks 7-9
- Phase 4 (Testing): 1h 20min - Tasks 10-12
- Phase 5 (Deployment): 55min - Tasks 13-15

---

## Dependencies & Risks

### Dependencies
- TOML library: `github.com/BurntSushi/toml` (add to go.mod)
- esmadrid.com uptime and API stability
- Existing tests must continue passing

### Risks & Mitigations

**Risk 1:** ESMadrid feed format changes
- **Mitigation:** Comprehensive parsing tests, graceful degradation

**Risk 2:** Performance with 2 pipelines
- **Mitigation:** Run pipelines concurrently, cache/snapshot both

**Risk 3:** Breaking existing deployment
- **Mitigation:** Backward compatible flags, existing pipeline unchanged

**Risk 4:** ESMadrid feed is slow/unreliable
- **Mitigation:** Timeouts, fallback to cultural events only, snapshot caching

---

## Success Criteria

✅ **Functional:**
- Both pipelines run independently
- Plaza de España city events visible (ice rink, Pride, etc.)
- Cultural events continue working (142 events)
- HTML shows both event types clearly
- JSON API includes both types

✅ **Quality:**
- All existing 22 tests pass
- New tests for city events pass
- Integration test passes
- FreeBSD binary builds

✅ **Performance:**
- Build time <10 seconds for both pipelines
- No memory leaks
- Graceful degradation if one source fails

✅ **Maintainability:**
- Clean separation between pipelines
- Config-driven (no hardcoded URLs)
- Well-documented architecture
- Easy to add 3rd pipeline in future

---

## Future Enhancements (Out of Scope)

- [ ] Add more city event sources (TimeOut Madrid, etc.)
- [ ] Machine learning to detect outdoor vs indoor events
- [ ] Event deduplication across sources (same event in both feeds)
- [ ] Map view showing event locations
- [ ] RSS/iCal feed generation
- [ ] Email notifications for new Plaza events

---

## Rollback Plan

If integration fails or causes issues:

1. **Revert to single pipeline:**
   - Keep cultural events code
   - Remove esmadrid.go and city.go
   - Remove config.toml requirement

2. **Fallback behavior:**
   - If config.toml missing, use CLI flags (backward compatible)
   - If esmadrid fetch fails, show cultural events only
   - Log errors but don't crash

3. **Git strategy:**
   - Each task is a separate commit
   - Can cherry-pick or revert individual features

---

## Next Steps

1. **Review this plan** with user
2. **Execute Phase 1** (Tasks 1-3) - Config & Foundation
3. **Test after each phase** before proceeding
4. **Update log file** after each task completion
5. **Commit with Claude coauthor** after each successful task

---

## Log File

Track progress in: `docs/logs/2025-10-20-esmadrid-integration-log.md`

Format:
```markdown
## Task N: [Name]
**Status:** ✅ Complete / 🚧 In Progress / ❌ Blocked
**Time:** HH:MM - HH:MM
**Commit:** [hash]

### What was done:
- [changes]

### Issues encountered:
- [if any]

### Next steps:
- [next task]
```

---

**End of Plan**
