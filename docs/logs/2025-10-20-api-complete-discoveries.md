# Complete API Discoveries from scrape.html Analysis

**Date:** 2025-10-20
**Source:** Comprehensive analysis of 51,253 lines of Swagger/OpenAPI documentation
**Total API Operations:** 196 endpoints
**Total Parameters:** 478 unique parameters

## 🎯 MAJOR DISCOVERIES

### Discovery #1: Server-Side Geographic Filtering (DISTRITO)
**Already documented** in `api-documentation-findings.md`

```
?distrito_nombre=MONCLOA-ARAVACA
```

### Discovery #2: 🔥 **RADIUS SEARCH** (NEW!)

The API supports **server-side radius filtering**!

**Parameters:**
```
?latitud=40.42338              # Latitude (decimal, use . as separator)
?longitud=-3.71217             # Longitude (decimal, use . as separator)
?distancia=350                 # Distance in METERS (positive integer)
```

**Requirements (from API docs):**
- `latitud` requires `longitud` and `distancia`
- `longitud` requires `latitud` and `distancia`
- `distancia` must be positive integer in meters, requires lat/lon

**Example Query:**
```
https://datos.madrid.es/egob/catalogo/300107-0-agenda-actividades-eventos.json?latitud=40.42338&longitud=-3.71217&distancia=350
```

**This is HUGE!** We're currently doing client-side radius filtering. The API can do it for us!

### Discovery #3: Pagination Support

**Parameters:**
```
?_page=1          # Page number
?_pageSize=100    # Results per page
```

**Use Case:** If distrito filter returns 100+ events, can paginate through results.

### Discovery #4: Multiple Query Strategies

The API supports **three different geographic filtering approaches**:

#### Option A: District Filter (Coarse)
```
?distrito_nombre=MONCLOA-ARAVACA
```
- Fastest (fewest events)
- Administrative boundaries
- ~50-100 events for Moncloa-Aravaca

#### Option B: Radius Search (Precise)
```
?latitud=40.42338&longitud=-3.71217&distancia=350
```
- Server-side GPS radius
- Exact 350m circle
- Unknown performance vs distrito

#### Option C: Distrito + Barrio (Very Precise)
```
?distrito_nombre=MONCLOA-ARAVACA&barrio_nombre=ARGUELLES
```
- Neighborhood-level filtering
- Most precise administrative filter
- Smallest result set

#### Option D: Hybrid (Distrito + Client Radius)
```
?distrito_nombre=MONCLOA-ARAVACA
# Then filter client-side by 350m radius
```
- **RECOMMENDED**: Balances server and client filtering
- Reduces data transfer (distrito)
- Guarantees precision (client radius)

---

## Complete Events API Parameters

### Agenda de actividades y eventos (300107)

**Endpoint:**
```
/catalogo/300107-0-agenda-actividades-eventos.{formato}
```

**All Supported Parameters:**

#### Geographic Filters
| Parameter | Type | Description | Example |
|-----------|------|-------------|---------|
| `distrito_nombre` | select | District name | `MONCLOA-ARAVACA` |
| `barrio_nombre` | string | Neighborhood name | `ARGUELLES` |
| `latitud` | decimal | Latitude (requires lon + dist) | `40.42338` |
| `longitud` | decimal | Longitude (requires lat + dist) | `-3.71217` |
| `distancia` | integer | Radius in meters (requires lat + lon) | `350` |

#### Pagination
| Parameter | Type | Description | Example |
|-----------|------|-------------|---------|
| `_page` | integer | Page number | `1` |
| `_pageSize` | integer | Results per page | `100` |

#### Format
| Parameter | Type | Description | Example |
|-----------|------|-------------|---------|
| `formato` | select | Response format | `json`, `xml`, `csv`, `rdf`, `geo` |

---

## Performance Comparison

### Current Implementation
```
Fetch: ALL Madrid events (no filters)
Size: ~1055 unique events
Filter: Client-side by 0.35km radius
Result: 13 events
Efficiency: 1.2% (98.8% discarded)
```

### Option 1: Distrito Filter Only
```
Fetch: ?distrito_nombre=MONCLOA-ARAVACA
Size: ~50-100 events (estimated)
Filter: Client-side by 0.35km radius
Result: ~13 events
Efficiency: ~15-25% (75-85% discarded)
Data Reduction: 90%
```

### Option 2: Server-Side Radius (NEW!)
```
Fetch: ?latitud=40.42338&longitud=-3.71217&distancia=350
Size: Unknown (depends on event density)
Filter: None needed (API does it!)
Result: Exactly events within 350m
Efficiency: 100% (0% discarded)
Data Reduction: 95%+ (potentially)
```

### Option 3: Hybrid (RECOMMENDED)
```
Fetch: ?distrito_nombre=MONCLOA-ARAVACA
Size: ~50-100 events
Filter: Client-side by 0.35km radius (backup precision)
Result: ~13 events
Efficiency: ~15-25%
Data Reduction: 90%
Reliability: High (distrito is stable, radius ensures precision)
```

---

## Testing Needed

### Test 1: Validate Radius Search Works
```bash
curl "https://datos.madrid.es/egob/catalogo/300107-0-agenda-actividades-eventos.json?latitud=40.42338&longitud=-3.71217&distancia=350"
```

**Expected:** JSON with only events within 350m of Plaza de España

**Question:** Does `distancia` use:
- Euclidean distance (straight line)
- Haversine distance (great-circle, accurate for Earth)
- Manhattan distance (city blocks)

### Test 2: Compare Distrito vs Radius
```bash
# Distrito filter
curl ".../300107-0-agenda-actividades-eventos.json?distrito_nombre=MONCLOA-ARAVACA" | jq '.["@graph"] | length'

# Radius filter
curl ".../300107-0-agenda-actividades-eventos.json?latitud=40.42338&longitud=-3.71217&distancia=350" | jq '.["@graph"] | length'
```

**Compare:**
- Event count
- Event IDs (are they the same?)
- Performance (which is faster?)

### Test 3: Pagination
```bash
# First page
curl ".../300107-0-agenda-actividades-eventos.json?distrito_nombre=MONCLOA-ARAVACA&_page=1&_pageSize=10"

# Second page
curl ".../300107-0-agenda-actividades-eventos.json?distrito_nombre=MONCLOA-ARAVACA&_page=2&_pageSize=10"
```

**Verify:** Different events returned

---

## Other Endpoint Discoveries

### Monuments with Distrito Filter
```
/catalogo/300356-0-monumentos-ciudad-madrid.json?distrito_nombre=MONCLOA-ARAVACA
```

**Use Case:** Show monuments near Plaza de España

### Museums with Distrito Filter
```
/catalogo/201132-0-museos.json?distrito_nombre=MONCLOA-ARAVACA
```

**Use Case:** Show museums near events

### Parks with Distrito Filter
```
/catalogo/200761-0-parques-jardines.json?distrito_nombre=MONCLOA-ARAVACA
```

**Use Case:** Show "Event near Parque del Oeste"

### Cultural Centers
```
/catalogo/200304-0-centros-culturales.json?distrito_nombre=MONCLOA-ARAVACA
```

**Use Case:** Enrich venue information

### WiFi Locations
```
/catalogo/216619-0-wifi-municipal.json?distrito_nombre=MONCLOA-ARAVACA
```

**Use Case:** "Free WiFi available at this venue"

### Temples/Churches
```
/catalogo/209426-0-templos-catolicas.json?distrito_nombre=MONCLOA-ARAVACA
```

**Use Case:** Context for events at religious venues

---

## Implementation Recommendations (Updated)

### Priority 1: Test Radius Search ✅ **MUST DO**

**Action:** Test if server-side radius search works and is accurate

**If YES:**
```go
// Use radius search (simplest, most efficient)
url := fmt.Sprintf("%s?latitud=%f&longitud=%f&distancia=%d",
    baseURL, lat, lon, int(radiusKm*1000))
```

**If NO or Inaccurate:**
```go
// Use distrito + client radius (our original plan)
url := baseURL + "?distrito_nombre=MONCLOA-ARAVACA"
// Then filter by radius client-side
```

**Estimated Effort:** 30 minutes testing

### Priority 2: Implement Best Filtering Strategy

**Decision Tree:**
```
Does radius search work accurately?
├─ YES → Use server radius (?latitud&longitud&distancia)
└─ NO  → Use distrito + client radius (?distrito_nombre + filter.WithinRadius)
```

**Estimated Effort:** 15 minutes implementation

### Priority 3: Add Pagination (Optional)

**Only if:**
- Result sets become too large
- Want to limit memory usage

```go
if eventCount > 100 {
    // Implement pagination
    url += "&_pageSize=100"
}
```

**Estimated Effort:** 30 minutes

---

## Query Parameter Patterns Discovered

### Geographic Filtering (Consistent Across APIs)
- `distrito_nombre` - Available on: Events, Museums, Parks, Monuments, Cultural Centers
- `barrio_nombre` - Available on: Events, Cultural Centers
- `latitud`/`longitud`/`distancia` - Available on: Events (unknown on others)

### Coordinate Systems
Multiple coordinate parameters found:
- `COORDENADA_OFICIAL_X/Y` - Official coordinates
- `COORDENADA_REAL_X/Y` - Real coordinates
- `COORD_GIS_X/Y` - GIS coordinates
- `coordenada_x/y_local` - Local coordinates

**Implication:** Different datasets may use different coordinate systems. Check metadata.

### Date Filtering (Limited)
- `date` - Some endpoints
- `beginDate` - Some endpoints
- No `endDate` parameter found
- Events API doesn't seem to support date filtering (returns all future events)

---

## Additional Features Found

### 1. Multiple Response Formats
All `/catalogo/*` endpoints support:
- `json` (default)
- `xml`
- `csv`
- `rdf` (RDF/XML)
- `geo` (GeoJSON likely)

### 2. Detail Endpoints
Pattern: `/catalogo/tipo/{tipo}/{id}.json`

Example:
```
/catalogo/tipo/evento/50016637.json
```

Get individual event details by ID.

### 3. Category/Type Filtering (Some Endpoints)
- `TIPO` parameter on some endpoints
- Event types embedded in data (not filterable)

### 4. Access Info
- `desc_tipo_acceso_local` - Access type description
- Could filter for accessible venues

---

## Network Issue Update

**All download attempts from container still fail:**
- distrito filter: 0 bytes
- radius filter: Not tested (will timeout)
- monuments: 0 bytes

**Hypothesis:** datos.madrid.es may:
- Block container IP range
- Require specific User-Agent
- Rate limit aggressively
- Only work from Spanish IPs

**Test Plan:**
1. Deploy to NFSN production (FreeBSD)
2. Test all filters from production server
3. If still blocked, contact datos.madrid.es support

---

## Summary of All Discoveries

### Geographic Filtering (3 Methods)
1. ✅ **Distrito filter** - `?distrito_nombre=MONCLOA-ARAVACA` (90% reduction)
2. 🔥 **Radius search** - `?latitud&longitud&distancia=350` (95%+ reduction, **NEW!**)
3. ✅ **Neighborhood filter** - `?barrio_nombre=ARGUELLES` (combines with distrito)

### Pagination
4. ✅ **Page/PageSize** - `?_page=1&_pageSize=100` (handle large results)

### Multiple Formats
5. ✅ **Format selection** - `json`, `xml`, `csv`, `rdf`, `geo`

### Detail Endpoints
6. ✅ **Individual resources** - `/catalogo/tipo/evento/{id}.json`

### Related Datasets
7. ✅ **Monuments** - 300356-0-monumentos-ciudad-madrid.json
8. ✅ **Museums** - 201132-0-museos.json
9. ✅ **Parks** - 200761-0-parques-jardines.json
10. ✅ **Cultural Centers** - 200304-0-centros-culturales.json
11. ✅ **WiFi** - 216619-0-wifi-municipal.json
12. ✅ **Temples** - 209426-0-templos-catolicas.json

### Total API Scope
- **196 API operations** across all Madrid services
- **478 unique parameters** across all endpoints
- **Consistent patterns** for geographic filtering

---

## Immediate Next Steps

1. ✅ **Document all discoveries** (this file)
2. **Test radius search from production**
3. **Implement best filtering strategy**
4. **Measure performance improvement**
5. **Consider POI enrichment** (monuments, parks)

**The radius search discovery could eliminate ALL client-side filtering!**

This is a game-changer for performance.
