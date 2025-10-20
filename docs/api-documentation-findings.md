# Madrid Open Data API Documentation Findings

**Date:** 2025-10-20
**Source:** scrape.html (Swagger/OpenAPI documentation from datos.madrid.es)
**Discovery:** **API supports query parameters for filtering!**

## Major Discovery: Query Parameters

The Madrid Open Data API supports **filtering parameters** that we're not currently using!

### Available Query Parameters for Events APIs

#### Geographic Filters
```
?distrito_nombre=MONCLOA-ARAVACA    # Filter by district
?barrio_nombre=ARGUELLES            # Filter by neighborhood
?latitud=40.42338                   # Filter by latitude
?longitud=-3.71217                  # Filter by longitude
```

#### District Values (All Madrid Districts)
- **MONCLOA-ARAVACA** ← Plaza de España is here!
- CENTRO
- CHAMBERI
- ARGANZUELA
- RETIRO
- SALAMANCA
- CHAMARTIN
- TETUAN
- FUENCARRAL-EL PARDO
- LATINA
- CARABANCHEL
- PUENTE DE VALLECAS
- MORATALAZ
- CIUDAD LINEAL
- HORTALEZA
- BARAJAS
- SAN BLAS-CANILLEJAS

### API Endpoints with Filter Support

#### 1. Agenda de actividades y eventos (300107) - **OUR CURRENT API**
**Endpoint:** `/catalogo/300107-0-agenda-actividades-eventos.json`

**Supported Filters:**
- `distrito_nombre` - District filter
- `barrio_nombre` - Neighborhood filter
- `latitud` - Latitude filter
- `longitud` - Longitude filter

**Example Query:**
```
https://datos.madrid.es/egob/catalogo/300107-0-agenda-actividades-eventos.json?distrito_nombre=MONCLOA-ARAVACA
```

#### 2. Actividades Culturales 100 días (206974)
**Endpoint:** `/catalogo/206974-0-agenda-eventos-culturales-100.json`

**Supported Filters:**
- `distrito_nombre`
- `barrio_nombre`
- `latitud`
- `longitud`

#### 3. Agenda de actividades deportivas (212504)
**Endpoint:** `/catalogo/212504-0-agenda-actividades-deportes.json`

**Supported Filters:**
- `distrito_nombre`
- `barrio_nombre`
- `latitud`
- `longitud`

#### 4. Actividades en Bibliotecas (206717)
**Endpoint:** `/catalogo/206717-0-agenda-eventos-bibliotecas.json`

**Supported Filters:**
- `distrito_nombre`
- `barrio_nombre`
- `latitud`
- `longitud`

### Event Detail Endpoint

**Individual Event:**
```
/catalogo/tipo/evento/{id}.json
```

**Filters:**
- `DISTRITO_DESC` - District description
- `DISTRITO_COD` - District code

---

## Impact on Current Implementation

### What We're Doing Now (Inefficient)
```go
// Fetch ALL Madrid events (1000+ events)
fetch.Client.FetchJSON("https://datos.madrid.es/.../300107-0-agenda-actividades-eventos.json")

// Then filter client-side by GPS radius
for _, evt := range allEvents {
    if filter.WithinRadius(lat, lon, evt.Latitude, evt.Longitude, radiusKm) {
        // Keep event
    }
}

// Result: 13 events from 1055 unique events (98.8% discarded)
```

### What We COULD Do (Efficient)
```go
// Fetch ONLY Moncloa-Aravaca district events
fetch.Client.FetchJSON("https://datos.madrid.es/.../300107-0-agenda-actividades-eventos.json?distrito_nombre=MONCLOA-ARAVACA")

// Then filter by precise GPS radius
for _, evt := range districtEvents {
    if filter.WithinRadius(lat, lon, evt.Latitude, evt.Longitude, radiusKm) {
        // Keep event
    }
}

// Result: Fetches ~50-100 events instead of 1000+
//         Faster parsing, less memory, quicker builds
```

---

## Benefits of Using District Filter

### 1. **Performance Improvement**
- ✅ **90%+ reduction** in data transfer (1055 events → ~50-100 events)
- ✅ **Faster parsing** (less JSON to decode)
- ✅ **Lower memory** usage during processing
- ✅ **Quicker builds** (especially on NFSN FreeBSD server)

### 2. **Server-Side Filtering**
- ✅ **Less network bandwidth**
- ✅ **Reduced CPU** on build process
- ✅ **More reliable** (let the API do the heavy lifting)

### 3. **Geographic Relevance**
- ✅ Plaza de España is in **MONCLOA-ARAVACA** district
- ✅ District filter = perfect match for our use case
- ✅ Still apply 0.35km radius for precision

### 4. **Simpler Logic**
```diff
- Fetch 3 formats × 1000+ events = 3000+ events
+ Fetch 3 formats × ~100 events = ~300 events
  Merge & deduplicate
  Filter by radius
```

---

## Neighborhood Information

### Plaza de España Neighborhoods

**District:** MONCLOA-ARAVACA

**Nearby Neighborhoods (barrios):**
- Argüelles (contains Plaza de España)
- Universidad
- Casa de Campo
- Aravaca

**Could refine further:**
```
?distrito_nombre=MONCLOA-ARAVACA&barrio_nombre=ARGUELLES
```

But district-level filtering is probably sufficient given our 0.35km radius.

---

## Additional API Features Discovered

### 1. Multiple Format Support
```
/catalogo/300107-0-agenda-actividades-eventos.{formato}

Where formato:
- json
- xml
- csv
- rdf
- geo
```

### 2. Date Parameters (Some Endpoints)
```
?date=...
?fecha_dato=...
```

**Note:** Not all endpoints support date filtering. Events API seems to return all future events.

### 3. Type/Category Filters (Limited)
```
?tipo=...
?categoria=...
```

**Note:** Not prominently featured in events APIs. Categories likely embedded in event data itself.

---

## Implementation Recommendations

### Priority 1: Add District Filter to Current Fetchers ✅ IMPLEMENT

**Change:** Minimal code change, big performance win

**Before:**
```go
const (
    jsonURL = "https://datos.madrid.es/egob/catalogo/300107-0-agenda-actividades-eventos.json"
    xmlURL  = "https://datos.madrid.es/egob/catalogo/300107-0-agenda-actividades-eventos.xml"
    csvURL  = "https://datos.madrid.es/egob/catalogo/300107-0-agenda-actividades-eventos.csv"
)
```

**After:**
```go
const (
    districtFilter = "?distrito_nombre=MONCLOA-ARAVACA"
    jsonURL = "https://datos.madrid.es/egob/catalogo/300107-0-agenda-actividades-eventos.json" + districtFilter
    xmlURL  = "https://datos.madrid.es/egob/catalogo/300107-0-agenda-actividades-eventos.xml" + districtFilter
    csvURL  = "https://datos.madrid.es/egob/catalogo/300107-0-agenda-actividades-eventos.csv" + districtFilter
)
```

**Impact:**
- ✅ 90% less data fetched
- ✅ Faster builds
- ✅ Same results (still filter by radius)
- ✅ Zero breaking changes

**Estimated Effort:** 10 minutes

### Priority 2: Make District Configurable (Optional)

Add flag:
```go
flag.String("distrito", "MONCLOA-ARAVACA", "Madrid district filter")
```

**Use Case:** Could deploy same code for other Madrid districts
- "Eventos en Retiro"
- "Eventos en Centro"
- etc.

**Estimated Effort:** 20 minutes

### Priority 3: Add Neighborhood Filter (Optional)

```go
flag.String("barrio", "", "Madrid neighborhood filter (optional)")
```

Refine to specific neighborhood within district.

**Estimated Effort:** 15 minutes

---

## Testing Strategy

### Test 1: Validate Filter Works
```bash
curl "https://datos.madrid.es/egob/catalogo/300107-0-agenda-actividades-eventos.json?distrito_nombre=MONCLOA-ARAVACA"
```

**Expected:** JSON response with only Moncloa-Aravaca events
**From Container:** ❌ Network issues (but likely works from production)

### Test 2: Compare Event Counts
```bash
# All Madrid
curl ".../300107-0-agenda-actividades-eventos.json" | jq '.["@graph"] | length'

# Moncloa-Aravaca only
curl ".../300107-0-agenda-actividades-eventos.json?distrito_nombre=MONCLOA-ARAVACA" | jq '.["@graph"] | length'
```

**Expected:** District filter returns 10-20% of total events

### Test 3: Verify Plaza de España Events Included
After filtering, verify our current 13 events are still present.

---

## Potential Issues & Mitigations

### Issue 1: Filter May Miss Events
**Problem:** Event venue in Moncloa-Aravaca, but system tagged wrong district

**Mitigation:** Keep geographic radius filter as backup
```go
// Server-side filter (fast)
url := baseURL + "?distrito_nombre=MONCLOA-ARAVACA"

// Client-side filter (precise)
if !filter.WithinRadius(...) {
    continue // Still verify GPS distance
}
```

### Issue 2: Network Issues Persist
**Problem:** Container can't download from datos.madrid.es

**Mitigation:**
- Test from production NFSN server
- If still blocked, report issue to datos.madrid.es
- Fallback: remove filter, continue with full fetch

### Issue 3: URL Encoding
**Problem:** Spaces in "MONCLOA-ARAVACA"

**Solution:** URL encode properly
```go
import "net/url"

params := url.Values{}
params.Add("distrito_nombre", "MONCLOA-ARAVACA")
fullURL := baseURL + "?" + params.Encode()
```

---

## Alternative Uses for API Filters

### Use Case 1: Multi-District Site
Deploy same code for different districts:
```
eventos-retiro.example.com   → distrito=RETIRO
eventos-centro.example.com   → distrito=CENTRO
eventos-chamberi.example.com → distrito=CHAMBERI
```

### Use Case 2: Lat/Lon API Filter
Instead of distrito, use precise coordinates:
```
?latitud=40.42338&longitud=-3.71217
```

**Note:** Unclear if this does radius search or just filters by exact match.

### Use Case 3: Combine Filters
```
?distrito_nombre=MONCLOA-ARAVACA&barrio_nombre=ARGUELLES
```

Very precise neighborhood focus.

---

## Documentation Quality

**API Docs (scrape.html) Quality:**
- ✅ Swagger/OpenAPI format
- ✅ Clear parameter descriptions
- ✅ Example values
- ✅ All endpoints documented

**Missing:**
- ⚠️ No response schema examples
- ⚠️ No rate limit documentation
- ⚠️ No authentication requirements
- ⚠️ No error code documentation

---

## Summary

### Key Discovery
**Madrid Open Data API supports query parameters!**

**distrito_nombre=MONCLOA-ARAVACA** filters events to our target area.

### Immediate Action
**Add district filter to current implementation:**
```diff
- jsonURL = ".../300107-0-agenda-actividades-eventos.json"
+ jsonURL = ".../300107-0-agenda-actividades-eventos.json?distrito_nombre=MONCLOA-ARAVACA"
```

**Benefits:**
- 90% less data
- Faster builds
- Same functionality
- 10-minute change

### Long-term Potential
- Configurable districts
- Multi-city deployment
- Neighborhood refinement
- Lat/lon filtering

---

## Next Steps

1. ✅ **Document findings** (this file)
2. **Test from production** (when deployed to NFSN)
3. **Implement district filter** (10 minutes)
4. **Measure improvement** (build time, data size)
5. **Consider neighborhood refinement**

**This is a significant discovery that makes the site more efficient!**
