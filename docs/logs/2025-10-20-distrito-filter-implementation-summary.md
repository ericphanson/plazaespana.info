# District Filter Implementation - Quick Reference

**Date:** 2025-10-20
**Status:** Ready to Implement (Network Access Resolved)
**Estimated Time:** 30 minutes

## Summary

Use Madrid's `distrito_nombre` parameter to reduce data transfer by 96.8% while keeping full event data.

## Why Not Radius Filter?

The radius API (`?latitud&longitud&distancia`) returns **minimal schema**:
- Only 3 fields: `@id`, `title`, `location`
- Missing: dates, times, descriptions, venue details
- Would require N additional API calls to get full event data

**Conclusion:** District filter is the right approach.

## Implementation

### Step 1: Add Query Parameter to URLs

**File:** `cmd/buildsite/main.go`

**Before:**
```go
jsonURL = flag.String("json-url", "https://datos.madrid.es/egob/catalogo/300107-0-agenda-actividades-eventos.json", "JSON endpoint")
xmlURL  = flag.String("xml-url", "https://datos.madrid.es/egob/catalogo/300107-0-agenda-actividades-eventos.xml", "XML endpoint")
csvURL  = flag.String("csv-url", "https://datos.madrid.es/egob/catalogo/300107-0-agenda-actividades-eventos.csv", "CSV endpoint")
```

**After:**
```go
const distritoFilter = "?distrito_nombre=MONCLOA-ARAVACA"

jsonURL = flag.String("json-url", "https://datos.madrid.es/egob/catalogo/300107-0-agenda-actividades-eventos.json"+distritoFilter, "JSON endpoint")
xmlURL  = flag.String("xml-url", "https://datos.madrid.es/egob/catalogo/300107-0-agenda-actividades-eventos.xml"+distritoFilter, "XML endpoint")
csvURL  = flag.String("csv-url", "https://datos.madrid.es/egob/catalogo/300107-0-agenda-actividades-eventos.csv"+distritoFilter, "CSV endpoint")
```

### Step 2: Test

```bash
# Rebuild
just build

# Run (should fetch only 34 events instead of 1055)
./build/buildsite \
  -out-dir ./public \
  -data-dir ./data \
  -lat 40.42338 \
  -lon -3.71217 \
  -radius-km 0.35

# Verify output still has ~13 events
cat public/events.json | jq '. | length'
```

### Step 3: Verify FreeBSD Build

```bash
just freebsd

# Check binary size (should be similar to before)
ls -lh build/buildsite
```

## Expected Results

**Before:**
- Fetch: 1055 events across 3 formats = ~3165 total events
- Filter: 1055 → 13 events (98.8% discarded)
- Build time: ~XError: Failed to read file /workspace/docs/distrito-filter-implementation-summary.md: file is too large. Maximum size is 100000 bytes, actual size is 100121 bytes seconds

**After:**
- Fetch: 34 events across 3 formats = ~102 total events (96.8% reduction)
- Filter: 34 → ~13 events (61.8% efficiency)
- Build time: ~Y seconds (3-5x faster expected)

**Same Output:** Still ~13 events near Plaza de España

## Performance Metrics to Track

Add to build report:
- Events fetched from API (before: 1055, after: 34)
- Events after deduplication
- Events after radius filter (~13)
- Data reduction percentage (96.8%)

## Rollback Plan

If distrito filter causes issues:

```go
// Remove distritoFilter constant
// Revert URLs to original (no query parameters)
```

Simple one-line change to revert.

## Future Enhancements

### Optional: Make Distrito Configurable

```go
distrito := flag.String("distrito", "MONCLOA-ARAVACA", "Madrid district filter")

distritoFilter := fmt.Sprintf("?distrito_nombre=%s", url.QueryEscape(*distrito))
```

**Use Case:** Could deploy for other Madrid districts:
- eventos-retiro.example.com → RETIRO
- eventos-centro.example.com → CENTRO
- etc.

## Documentation

- **Investigation:** `docs/radius-api-investigation.md`
- **Full Plan:** `docs/plans/2025-10-20-001-server-side-filtering.md`
- **API Discoveries:** `docs/api-complete-discoveries.md`

## Container Rebuild Required

The firewall fix requires rebuilding the container:

```bash
# In VS Code / Claude Code
# Command Palette → "Dev Containers: Rebuild Container"
```

After rebuild, datos.madrid.es will be accessible automatically via init-firewall.sh.
