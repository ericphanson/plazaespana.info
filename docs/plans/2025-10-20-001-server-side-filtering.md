# Implementation Plan 001: District-Based Server-Side Filtering

**Date:** 2025-10-20
**Plan ID:** 2025-10-20-001
**Status:** ✅ Network Access Resolved - Ready to Implement

**Goal:** Use Madrid's distrito filter to reduce data transfer by 96.8% while maintaining full event data.

**Philosophy:** Let the server do coarse filtering (distrito), we handle precision (radius).

**Critical Discovery:** Radius API returns minimal schema (only 3 fields - no dates/times/descriptions). District filter is the right approach. See `docs/radius-api-investigation.md` for details.

**Network Issue:** ✅ RESOLVED - Fixed DNS CNAME resolution in init-firewall.sh

---

## Current Architecture (Inefficient)

```
┌─────────────────────────────────────────────────────────┐
│ Madrid API (All Events)                                │
│ https://datos.madrid.es/.../300107-0-agenda...json     │
│ Returns: 1055 unique events (ALL Madrid)               │
└─────────────────────────────────────────────────────────┘
                        ↓ (large download)
┌─────────────────────────────────────────────────────────┐
│ Our Server: Parse ALL events                           │
│ - Decode 3 formats (JSON/XML/CSV)                      │
│ - Merge & deduplicate (3003 → 1055 events)             │
│ - Filter by GPS radius (1055 → 13 events)              │
│ - 98.8% of data discarded after download!              │
└─────────────────────────────────────────────────────────┘
                        ↓
┌─────────────────────────────────────────────────────────┐
│ Output: 13 events                                       │
└─────────────────────────────────────────────────────────┘

Problems:
❌ Download 1055 events, use 13 (1.2% efficiency)
❌ Parse 3000+ events across 3 formats
❌ Client-side Haversine calculations
❌ Slow builds (important for hourly cron)
❌ High memory usage
```

---

## Target Architecture (Efficient)

```
┌─────────────────────────────────────────────────────────┐
│ Madrid API (Distrito-Filtered)                         │
│ ?distrito_nombre=MONCLOA-ARAVACA                       │
│ Returns: 34 events (MONCLOA-ARAVACA district only)     │
│ Schema: FULL (dates, times, descriptions, etc.)        │
└─────────────────────────────────────────────────────────┘
                        ↓ (small download - 96.8% reduction)
┌─────────────────────────────────────────────────────────┐
│ Our Server: Targeted Processing                        │
│ - Parse district events (34 events)                    │
│ - Merge & deduplicate across formats                   │
│ - Filter by 0.35km radius (34 → ~13 events)            │
│ - Enrich with descriptions                             │
│ - Generate HTML/JSON                                   │
└─────────────────────────────────────────────────────────┘
                        ↓
┌─────────────────────────────────────────────────────────┐
│ Output: ~13 events (same as current, but faster)       │
└─────────────────────────────────────────────────────────┘

Benefits:
✅ Download 34 events, use ~13 (38% efficiency vs 1.2%)
✅ Parse ~102 events total (34×3 formats vs 3000+)
✅ 96.8% data reduction (1055 → 34)
✅ Full event schema (no secondary API calls needed)
✅ Fast builds (3-5x faster)
✅ Low memory usage
✅ Same functionality, better performance
```

---

## Implementation Plan

### Phase 1: Network Access ✅ COMPLETE

**Goal:** Fix network/firewall issues to test API filtering

**Issue:** datos.madrid.es uses CNAME records that weren't being followed by DNS resolution in init-firewall.sh

**Solution:** Changed `dig +noall +answer A` to `dig +short A` to follow CNAME chains
- datos.madrid.es → madridw.edgekey.net → e101210.dscb.akamaiedge.net → [2.18.188.31, 2.18.188.10]

**Result:** ✅ API access working
- District filter tested: 34 events
- Radius filter tested: 1 event (minimal schema - unusable)

**Files Changed:** `.devcontainer/init-firewall.sh`
**Documentation:** `docs/radius-api-investigation.md`

**Estimated Time:** ✅ 1 hour (completed)

**Next:** Rebuild container for firewall changes to take effect permanently

#### Task 1.1: Test Distrito Filter API ✅ COMPLETE
**Location:** Once network access works

**Test A: Radius Search (Primary)**
```bash
curl "https://datos.madrid.es/egob/catalogo/300107-0-agenda-actividades-eventos.json?latitud=40.42338&longitud=-3.71217&distancia=350" -o /tmp/radius-test.json

# Verify response
cat /tmp/radius-test.json | head -100
```

**Expected:** Valid JSON with events near Plaza de España

**Test B: Distrito Search (Backup)**
```bash
curl "https://datos.madrid.es/egob/catalogo/300107-0-agenda-actividades-eventos.json?distrito_nombre=MONCLOA-ARAVACA" -o /tmp/distrito-test.json
```

**Expected:** Valid JSON with Moncloa-Aravaca events

**Test C: Compare Results**
```bash
# Count events in each
grep -o '"ID-EVENTO"' /tmp/radius-test.json | wc -l
grep -o '"ID-EVENTO"' /tmp/distrito-test.json | wc -l

# Extract IDs for comparison
grep -o '"ID-EVENTO":"[^"]*"' /tmp/radius-test.json > /tmp/radius-ids.txt
grep -o '"ID-EVENTO":"[^"]*"' /tmp/distrito-test.json > /tmp/distrito-ids.txt

# Check overlap
comm -12 <(sort /tmp/radius-ids.txt) <(sort /tmp/distrito-ids.txt)
```

**Success Criteria:**
- ✅ Both methods return valid JSON
- ✅ Radius returns ≤ distrito (radius is subset)
- ✅ Radius results are actually within 350m (verify coordinates)

**Estimated Time:** 20 minutes

#### Task 1.3: Validate Accuracy
**Goal:** Ensure radius search uses proper distance calculation

**Method:**
```bash
# Download radius results
# For each event, verify actual distance from Plaza de España
# Using Haversine formula

# Python one-liner for validation:
python3 << 'EOF'
import json, math

def haversine(lat1, lon1, lat2, lon2):
    R = 6371000  # Earth radius in meters
    φ1, φ2 = math.radians(lat1), math.radians(lat2)
    Δφ = math.radians(lat2 - lat1)
    Δλ = math.radians(lon2 - lon1)
    a = math.sin(Δφ/2)**2 + math.cos(φ1) * math.cos(φ2) * math.sin(Δλ/2)**2
    return R * 2 * math.atan2(math.sqrt(a), math.sqrt(1-a))

plaza_lat, plaza_lon = 40.42338, -3.71217
with open('/tmp/radius-test.json') as f:
    data = json.load(f)
    for event in data.get('@graph', []):
        lat = event.get('location', {}).get('latitude', 0)
        lon = event.get('location', {}).get('longitude', 0)
        if lat and lon:
            dist = haversine(plaza_lat, plaza_lon, lat, lon)
            status = "✓" if dist <= 350 else "✗ OUTSIDE"
            print(f"{status} {dist:.0f}m - {event.get('title', 'Unknown')}")
EOF
```

**Success Criteria:**
- ✅ All events are within 350m
- ✅ No events outside radius (proves API is accurate)

**Estimated Time:** 15 minutes

---

### Phase 2: Implement Server-Side Filtering

**Outcome of Phase 1 determines approach:**

#### Option A: Radius Search Works (PREFERRED)

**Implementation:**
```go
// internal/fetch/urls.go (new file)
package fetch

import (
    "fmt"
    "net/url"
)

// BuildEventURL constructs event API URL with radius filter
func BuildEventURL(baseURL string, lat, lon float64, radiusM int, format string) string {
    params := url.Values{}
    params.Add("latitud", fmt.Sprintf("%.6f", lat))
    params.Add("longitud", fmt.Sprintf("%.6f", lon))
    params.Add("distancia", fmt.Sprintf("%d", radiusM))

    return fmt.Sprintf("%s.%s?%s", baseURL, format, params.Encode())
}

// URLs for Plaza de España (0.35km = 350m)
const (
    BasePlazaURL = "https://datos.madrid.es/egob/catalogo/300107-0-agenda-actividades-eventos"
    PlazaLat     = 40.42338
    PlazaLon     = -3.71217
    PlazaRadiusM = 350
)

func GetPlazaEventURLs() (jsonURL, xmlURL, csvURL string) {
    jsonURL = BuildEventURL(BasePlazaURL, PlazaLat, PlazaLon, PlazaRadiusM, "json")
    xmlURL  = BuildEventURL(BasePlazaURL, PlazaLat, PlazaLon, PlazaRadiusM, "xml")
    csvURL  = BuildEventURL(BasePlazaURL, PlazaLat, PlazaLon, PlazaRadiusM, "csv")
    return
}
```

**Update pipeline:**
```go
// internal/pipeline/pipeline.go
func NewPipeline(lat, lon float64, radiusM int, client *fetch.Client, loc *time.Location) *Pipeline {
    jsonURL, xmlURL, csvURL := fetch.BuildRadiusURLs(lat, lon, radiusM)

    return &Pipeline{
        jsonURL: jsonURL,
        xmlURL:  xmlURL,
        csvURL:  csvURL,
        client:  client,
        loc:     loc,
    }
}
```

**Update main.go:**
```go
// cmd/buildsite/main.go
func main() {
    // ... flag parsing ...

    // No longer need to pass full URLs as flags!
    // API does the filtering for us
    pipe := pipeline.NewPipeline(*lat, *lon, int(*radiusKm*1000), client, loc)

    pipeResult := pipe.FetchAll()
    merged := pipe.Merge(pipeResult)

    // Optional: Safety check (should be unnecessary if API is accurate)
    // Can remove after validation period
    for _, evt := range merged {
        if evt.Latitude != 0 && evt.Longitude != 0 {
            if !filter.WithinRadius(*lat, *lon, evt.Latitude, evt.Longitude, *radiusKm) {
                log.Printf("WARNING: API returned event outside radius: %s", evt.ID)
            }
        }
    }

    // Continue with rendering...
}
```

**Benefits:**
- ✅ API does ALL geographic filtering
- ✅ Remove filter.WithinRadius() calls (except optional safety check)
- ✅ Remove text-based location fallback (API has GPS data)
- ✅ 95%+ less data downloaded
- ✅ Essentially a "Plaza de España event feed" proxy

**Estimated Time:** 1-2 hours

#### Option B: Radius Search Doesn't Work (FALLBACK)

**Implementation:**
```go
// Use distrito filter instead
const DistritoFilter = "?distrito_nombre=MONCLOA-ARAVACA"

jsonURL := "https://datos.madrid.es/egob/catalogo/300107-0-agenda-actividades-eventos.json" + DistritoFilter
xmlURL  := "https://datos.madrid.es/egob/catalogo/300107-0-agenda-actividades-eventos.xml" + DistritoFilter
csvURL  := "https://datos.madrid.es/egob/catalogo/300107-0-agenda-actividades-eventos.csv" + DistritoFilter
```

**Keep existing filtering:**
```go
// Still need client-side radius check
for _, evt := range merged {
    if filter.WithinRadius(*lat, *lon, evt.Latitude, evt.Longitude, *radiusKm) {
        filteredEvents = append(filteredEvents, evt)
    }
}
```

**Benefits:**
- ✅ 90% less data downloaded
- ✅ Faster than current approach
- ⚠️ Still need client-side filtering

**Estimated Time:** 30 minutes

---

### Phase 3: Optimize Data Pipeline

**Goal:** Minimal processing, maximum efficiency

#### Task 3.1: Remove Redundant Filtering

**If radius search works:**
```diff
- // Filter by location and time
- for _, evt := range merged {
-     if evt.Latitude == 0 || evt.Longitude == 0 {
-         if filter.MatchesLocation(...) {
-             // text fallback
-         }
-     } else {
-         if !filter.WithinRadius(...) {
-             continue
-         }
-     }
- }
+ // API already filtered by radius!
+ // Just filter by time
+ for _, evt := range merged {
+     if filter.IsInFuture(evt.EndTime, now) {
+         filteredEvents = append(filteredEvents, evt)
+     }
+ }
```

**Data Flow:**
```
Before: Fetch(1055) → Merge(1055) → GeoFilter(13) → TimeFilter(13) → Render(13)
After:  Fetch(15)   → Merge(15)   → TimeFilter(13) → Render(13)
```

**Estimated Time:** 30 minutes

#### Task 3.2: Consider Single-Format Fetching

**Question:** Do we need all 3 formats if API filters for us?

**Analysis:**
- **Current:** Fetch 3 formats for redundancy (JSON fails → XML → CSV)
- **With filtering:** All 3 formats return same ~15 events
- **Deduplication benefit:** Minimal (events already filtered)

**Options:**

**A. Keep 3 formats (conservative)**
```go
// Benefit: Redundancy if one source fails
// Cost: 3x network requests (but small data now)
pipe := pipeline.NewPipeline(lat, lon, radiusM, client, loc)
```

**B. Use JSON only (efficient)**
```go
// Benefit: Single request, fastest
// Cost: No fallback if JSON fails
events := fetch.FetchJSONFiltered(jsonURL, client, loc)
```

**C. JSON + fallback (balanced)**
```go
// Try JSON first
events := fetch.FetchJSONFiltered(jsonURL, client, loc)
if len(events) == 0 {
    // Fallback to XML
    events = fetch.FetchXMLFiltered(xmlURL, client, loc)
}
```

**Recommendation:** **Keep 3 formats initially**, can optimize later based on reliability.

**Estimated Time:** 15 minutes (decision) or 1 hour (if implementing single format)

---

### Phase 4: Make Configuration Flexible

**Goal:** Easy to deploy for other locations

#### Task 4.1: Parameterize Everything

```go
// cmd/buildsite/main.go
func main() {
    // Geographic configuration
    lat := flag.Float64("lat", 40.42338, "Reference latitude")
    lon := flag.Float64("lon", -3.71217, "Reference longitude")
    radiusKm := flag.Float64("radius-km", 0.35, "Filter radius in kilometers")

    // Optional: District fallback
    distrito := flag.String("distrito", "", "District filter (fallback if radius fails)")

    // API configuration
    apiBase := flag.String("api-base",
        "https://datos.madrid.es/egob/catalogo/300107-0-agenda-actividades-eventos",
        "Madrid events API base URL")

    useRadiusSearch := flag.Bool("use-radius-search", true,
        "Use server-side radius search (disable to use distrito)")

    flag.Parse()

    // Build URLs based on configuration
    var jsonURL, xmlURL, csvURL string
    if *useRadiusSearch {
        jsonURL, xmlURL, csvURL = fetch.BuildRadiusURLs(*apiBase, *lat, *lon, int(*radiusKm*1000))
    } else if *distrito != "" {
        jsonURL, xmlURL, csvURL = fetch.BuildDistritoURLs(*apiBase, *distrito)
    } else {
        // No filtering (fetch all)
        jsonURL = *apiBase + ".json"
        xmlURL  = *apiBase + ".xml"
        csvURL  = *apiBase + ".csv"
    }
}
```

**Benefits:**
- ✅ Can switch between radius/distrito filtering via flag
- ✅ Easy to deploy for other Madrid locations
- ✅ Backwards compatible (can disable filtering)

**Example Usage:**
```bash
# Plaza de España with radius search (default)
./buildsite

# Plaza de España with distrito fallback
./buildsite -use-radius-search=false -distrito=MONCLOA-ARAVACA

# Retiro Park (different location)
./buildsite -lat=40.4153 -lon=-3.6844 -radius-km=0.5

# All Madrid (no filtering)
./buildsite -use-radius-search=false
```

**Estimated Time:** 1 hour

---

### Phase 5: Enhanced Reporting

**Goal:** Show filtering efficiency in build reports

#### Task 5.1: Add Filtering Statistics

```go
// internal/report/types.go
type FetchReport struct {
    JSON FetchAttempt
    XML  FetchAttempt
    CSV  FetchAttempt
    TotalDuration time.Duration

    // NEW: Filtering metadata
    FilterMethod   string  // "radius", "distrito", "none"
    FilterDetails  string  // "350m radius" or "MONCLOA-ARAVACA district"
    ServerFiltered bool    // true if API did filtering
}
```

**Update build report:**
```markdown
## 1. Data Fetching (Server-Side Filtered)

**Filter Method:** Radius Search
**Location:** 40.42338°N, 3.71217°W
**Radius:** 350 meters
**Server-Side:** ✅ Yes (API filtered)

| Source | Status | Events | Duration |
|--------|--------|--------|----------|
| JSON   | ✅ SUCCESS | 5 | 0.23s |
| XML    | ✅ SUCCESS | 5 | 0.31s |
| CSV    | ✅ SUCCESS | 5 | 0.28s |

**Efficiency:** 100% (all fetched events are relevant)
**Data Transfer:** 15 events fetched vs 1055 without filtering (98.6% reduction)
```

**Estimated Time:** 30 minutes

---

### Phase 6: Optional POI Enrichment

**Goal:** Add context from related datasets

#### Task 6.1: Fetch Nearby Points of Interest

**Since distrito filter works on other datasets:**

```go
// internal/poi/fetcher.go (new package)
package poi

import "github.com/ericphanson/madrid-events/internal/fetch"

type POIFetcher struct {
    distrito string
    client   *fetch.Client
}

func (f *POIFetcher) FetchNearbyMonuments() ([]Monument, error) {
    url := fmt.Sprintf(
        "https://datos.madrid.es/egob/catalogo/300356-0-monumentos-ciudad-madrid.json?distrito_nombre=%s",
        f.distrito,
    )
    // Fetch and parse...
}

func (f *POIFetcher) FetchNearbyParks() ([]Park, error) {
    url := fmt.Sprintf(
        "https://datos.madrid.es/egob/catalogo/200761-0-parques-jardines.json?distrito_nombre=%s",
        f.distrito,
    )
    // Fetch and parse...
}
```

**Enrich events:**
```go
// After filtering events
poiFetcher := poi.NewFetcher("MONCLOA-ARAVACA", client)
monuments := poiFetcher.FetchNearbyMonuments()
parks := poiFetcher.FetchNearbyParks()

// Cross-reference
for i := range filteredEvents {
    // Find nearest monument/park
    filteredEvents[i].NearbyPOIs = findNearby(filteredEvents[i], monuments, parks)
}
```

**Display in HTML:**
```html
<article>
    <h2>{{.Titulo}}</h2>
    <p class="when">{{.StartHuman}}</p>
    <p class="where">{{.NombreInstalacion}}</p>
    {{if .Description}}<p class="description">{{.Description}}</p>{{end}}

    <!-- NEW: Nearby context -->
    {{if .NearbyPOIs}}
    <p class="nearby">
        📍 Near: {{range .NearbyPOIs}}{{.Name}}{{end}}
    </p>
    {{end}}
</article>
```

**Estimated Time:** 2-3 hours

**Priority:** Low (nice-to-have, not essential)

---

## Phased Rollout

### Week 1: Validation & Core Implementation
- ✅ Deploy to production (Task 1.1)
- ✅ Test radius/distrito filtering (Task 1.2)
- ✅ Validate accuracy (Task 1.3)
- ✅ Implement chosen filtering (Task 2)
- ✅ Remove redundant code (Task 3.1)

**Deliverable:** Working "Plaza de España feed" with 90-95% less data transfer

### Week 2: Optimization & Configuration
- ✅ Flexible configuration (Task 4.1)
- ✅ Enhanced reporting (Task 5.1)
- ✅ Performance measurements

**Deliverable:** Configurable, well-documented system

### Week 3: Enhancement (Optional)
- 🤔 POI enrichment (Task 6.1)
- 🤔 Additional datasets
- 🤔 Multi-location deployment

**Deliverable:** Rich context for events

---

## Success Metrics

### Performance
- **Data Transfer:** 95%+ reduction (1055 events → 15 events)
- **Build Time:** 5-10x faster (less parsing)
- **Memory Usage:** 90%+ reduction
- **Network Requests:** Same (3 formats) but smaller responses

### Code Quality
- **Lines Removed:** ~100 lines of filtering logic
- **Complexity:** Lower (API does heavy lifting)
- **Maintainability:** Higher (less custom logic)

### Functionality
- **Accuracy:** Same or better (API filtering)
- **Reliability:** Same (still have fallbacks)
- **User Experience:** Identical output, faster updates

---

## Risk Mitigation

### Risk 1: Radius Search Doesn't Work
**Mitigation:** Fall back to distrito filter (still 90% improvement)

### Risk 2: API Changes/Breaks
**Mitigation:**
- Keep fallback to unfiltered URLs
- Monitor build reports for errors
- Alert on data count anomalies

### Risk 3: Different Event Counts
**Mitigation:**
- Compare before/after event counts
- Validate first week in parallel
- Keep old code in git history

### Risk 4: Server-Side Filter Inaccurate
**Mitigation:**
- Add safety check (verify coordinates client-side)
- Log warnings for out-of-radius events
- Report issues to datos.madrid.es

---

## Migration Strategy

### Safe Migration Path
```go
// 1. Add new filtered URLs alongside old
filteredJSON := buildRadiusURL(...)
unfilteredJSON := oldURL

// 2. Fetch both in parallel (temporarily)
filtered := fetch(filteredJSON)
unfiltered := fetch(unfilteredJSON)

// 3. Compare and log
log.Printf("Filtered: %d events, Unfiltered: %d events",
    len(filtered), len(unfiltered))

// 4. Validate filtered is subset of unfiltered
for _, evt := range filtered {
    if !contains(unfiltered, evt) {
        log.Printf("WARNING: Filtered has event not in unfiltered: %s", evt.ID)
    }
}

// 5. After validation period, remove unfiltered fetch
```

**Validation Period:** 1 week (7 builds)

**Rollback:** Revert commit if issues found

---

## Implementation Checklist

### Phase 1: Testing ✅
- [ ] Deploy current code to NFSN production
- [ ] Test radius search API from production
- [ ] Test distrito search API from production
- [ ] Validate accuracy (all events within radius)
- [ ] Measure performance (response time, data size)
- [ ] Document findings

### Phase 2: Implementation ✅
- [ ] Create fetch/urls.go with URL builders
- [ ] Update pipeline to use filtered URLs
- [ ] Update main.go to pass lat/lon/radius
- [ ] Add configuration flags
- [ ] Test locally (if container works) or on production

### Phase 3: Validation ✅
- [ ] Run parallel builds (filtered + unfiltered)
- [ ] Compare event counts
- [ ] Verify event IDs match
- [ ] Check build times
- [ ] Review build reports

### Phase 4: Cleanup ✅
- [ ] Remove unnecessary filtering code
- [ ] Update documentation
- [ ] Update CLAUDE.md
- [ ] Write migration notes

### Phase 5: Optimization 🤔
- [ ] Consider single-format fetching
- [ ] Add POI enrichment (optional)
- [ ] Multi-location support (optional)

---

## Documentation Updates Needed

### CLAUDE.md
```markdown
## Data Fetching Strategy

**Server-Side Filtering:** The Madrid Open Data API does the geographic filtering for us!

**Query:** Events within 350m of Plaza de España
**Method:** Radius search API
**URL:** `...?latitud=40.42338&longitud=-3.71217&distancia=350`

**Benefits:**
- Fetches ~15 events instead of 1055 (95% reduction)
- No client-side geographic filtering needed
- Essentially a "Plaza de España event feed" proxy

**Architecture:**
```
Madrid API (filtered) → Parse → Time Filter → Enrich → Render
```

No geographic filtering code! API handles it.
```

### README.md
```markdown
## How It Works

This site is essentially a specialized view of Madrid's Open Data API, filtered to show only events near Plaza de España.

**Data Source:** Madrid Ayuntamiento Open Data Portal
**Filtering:** Server-side (API handles geographic filtering)
**Updates:** Hourly via cron

The site doesn't store data—it's a real-time feed from Madrid's API.
```

---

## Expected Outcome

**Before:**
```bash
$ time ./buildsite
Fetching JSON from: https://datos.madrid.es/egob/catalogo/300107-0-agenda-actividades-eventos.json
Fetched 1001 events from JSON
Fetched 1001 events from XML
Fetched 1001 events from CSV
After merge: 1055 unique events from 3003 total
After filtering: 13 events
Build complete!

real    0m8.342s
user    0m2.103s
sys     0m0.234s
```

**After (radius search):**
```bash
$ time ./buildsite
Fetching JSON from: https://datos.madrid.es/egob/catalogo/300107-0-agenda-actividades-eventos.json?latitud=40.42338&longitud=-3.71217&distancia=350
Fetched 5 events from JSON
Fetched 5 events from XML
Fetched 5 events from CSV
After merge: 5 unique events from 15 total
After time filtering: 5 events (API already geo-filtered!)
Build complete!

real    0m1.532s
user    0m0.421s
sys     0m0.067s
```

**5x faster builds, 95% less data, same results!**

---

## Summary

This plan transforms your site from:
- ❌ "Download all Madrid → filter heavily"
To:
- ✅ "Get Plaza de España feed → minimal processing"

**Key Principle:** Let Madrid's API do the work. We just format and present.

**Next Step:** Test radius search from production (Task 1.1-1.3)
