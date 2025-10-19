# Debugging Session: No Events Appearing on Site

**Date:** 2025-10-20
**Issue:** Site generates successfully but shows 0 events after filtering (1001 events fetched from CSV)

## Phase 1: Root Cause Investigation

### 1. Error Messages and Observations

From `just dev` output:
```
2025/10/20 01:18:11 JSON fetch failed: decoding JSON: invalid character '\n' in string literal
2025/10/20 01:18:11 XML fetch failed: decoding XML: expected element type <response> but have <Contenidos>
2025/10/20 01:18:12 Fetched 1001 events from CSV
2025/10/20 01:18:12 After deduplication: 1001 events
2025/10/20 01:18:12 After filtering: 0 events
```

**Key observations:**
- CSV fetch succeeds (1001 events)
- Deduplication works (1001 remains)
- **Geographic filtering removes ALL events (0 remaining)**

### 2. Data Flow Analysis

**Pipeline:** Fetch CSV → Parse → Deduplicate → Filter (geo + time) → Render

**Critical code path (main.go:116-120):**
```go
for _, event := range rawEvents {
    // Skip if missing coordinates
    if event.Lat == 0 || event.Lon == 0 {
        continue
    }
    // ... rest of filtering
}
```

**Hypothesis forming:** All events have lat=0, lon=0, causing them to be filtered out.

### 3. Coordinate Parsing Investigation

**Current implementation (client.go:174-179):**
```go
if latStr := getField(row, headerMap, "COORDENADA-LATITUD"); latStr != "" {
    fmt.Sscanf(latStr, "%f", &event.Lat)
}
```

**Problem:** `fmt.Sscanf` may be failing silently if coordinate format doesn't match `%f` expectations.

### 4. Evidence Gathering - COMPLETED

**Debug output:**
```
DEBUG[1]: lat='' lon='' title='100 Libros juntas'
DEBUG[2]: lat='' lon='' title='1984'
```

**ROOT CAUSE IDENTIFIED:** Coordinate fields are empty strings in the CSV data.

**Implications:**
- `getField(row, headerMap, "COORDENADA-LATITUD")` returns `""`
- The `if latStr != ""` check fails, so `event.Lat` stays at default value `0`
- Same for longitude
- Filter at main.go:118 removes all events where `lat == 0 || lon == 0`

---

## Phase 2: Pattern Analysis - COMPLETED

### CSV Column Analysis

**Actual CSV columns (29 total):**
- `LATITUD` ✓
- `LONGITUD` ✓
- `COORDENADA-X` (projection coordinates)
- `COORDENADA-Y` (projection coordinates)
- ... other fields ...

**Code expectations:**
- Looking for: `COORDENADA-LATITUD` ✗
- Looking for: `COORDENADA-LONGITUD` ✗

**THE REAL ROOT CAUSE:**
Column name mismatch! CSV uses `LATITUD`/`LONGITUD` but code looks for `COORDENADA-LATITUD`/`COORDENADA-LONGITUD`.

This explains why coordinate fields are empty - `getField()` returns `""` when column doesn't exist.

---

## Phase 3: Hypothesis and Testing

### Hypothesis

**Statement:** The CSV parsing uses incorrect column names. Changing from `COORDENADA-LATITUD`/`COORDENADA-LONGITUD` to `LATITUD`/`LONGITUD` will fix coordinate parsing and allow events to pass geographic filtering.

### Minimal Test - COMPLETED

**Change made:** Column names in client.go:180-181 changed to `LATITUD`/`LONGITUD`

**Result:**
```
DEBUG[1]: lat='40.37691447788016' lon='-3.7416256113806847' fecha='2025-10-25 00:00:00.0'
DEBUG[2]: lat='40.4249721516942' lon='-3.689948085098791' fecha='2026-03-03 00:00:00.0'
```

✅ **Hypothesis CONFIRMED** - Coordinates now parse correctly!

**However,** discovered SECOND root cause:
```
DEBUG: Filtered out - no coords: 94, geo: 366, parse fail: 541, past: 0
```

541 events (54%) failing date parsing!

---

## SECOND ROOT CAUSE: Date Format Mismatch

### Investigation

**CSV date format:** `YYYY-MM-DD HH:MM:SS.S` (e.g., `2025-10-25 00:00:00.0`)
**Parser expects:** `DD/MM/YYYY` (time.go:12)

**Code location:** `internal/filter/time.go:12`
```go
layout := "02/01/2006"  // Wrong for CSV!
```

### Fix Strategy

CSV format needs parser to use layout: `2006-01-02 15:04:05.0` or handle multiple formats

---

## Phase 4: Implementation - COMPLETED

### Changes Made

1. **Coordinate Column Names** (client.go:180-181)
   - Changed from `COORDENADA-LATITUD`/`COORDENADA-LONGITUD` to `LATITUD`/`LONGITUD`

2. **Date Parser** (filter/time.go:13-38)
   - Updated to handle multiple formats:
     - `YYYY-MM-DD HH:MM:SS.S` (CSV format)
     - `DD/MM/YYYY HH:MM` (JSON/XML format with time)
     - `DD/MM/YYYY` (JSON/XML format without time)

3. **Tests Updated**
   - Added CSV date format test cases to time_test.go
   - Updated CSV test data to use correct column names
   - All 22 tests passing ✅

### Results

**With fixes applied:**
- 0.35km radius: 0 events (by design - too restrictive for this location)
- 1km radius: 12 events ✅
- 2km radius: 42 events ✅
- 10km radius: 181 events ✅

**Filter breakdown (2km radius):**
```
Filtered: noCoords=94 geo=688 parseFail=174 past=3 kept=42
```

### Root Causes Summary

1. **Bug #1 (FIXED):** CSV coordinate column names incorrect
   - Expected: `LATITUD`, `LONGITUD`
   - Code used: `COORDENADA-LATITUD`, `COORDENADA-LONGITUD`

2. **Bug #2 (FIXED):** Date parser didn't handle CSV timestamp format
   - CSV provides: `YYYY-MM-DD HH:MM:SS.S`
   - Parser only handled: `DD/MM/YYYY`

3. **Bug #3 (FIXED):** Date parser concatenated CSV date+time with HORA field
   - CSV dates include full timestamp: `2025-11-16 00:00:00.0`
   - HORA field (e.g., `19:00`) was being appended, creating invalid format
   - Fixed: Only append HORA for DD/MM/YYYY format, not CSV format

4. **Configuration (NOT A BUG):** Default 0.35km radius too small
   - No events exist within 350m of Plaza de España that meet all criteria
   - Recommend increasing to 1-2km for this location

## Final Results

**All bugs fixed:**
✅ Coordinate column names corrected
✅ CSV date format parsing implemented
✅ HORA field handling fixed for CSV dates

**Test results:**
- All 22 tests passing
- 0 parse failures with 2km radius
- 204 events successfully rendered

**With 2km radius:**
```
Filtered: noCoords=94 geo=688 parseFail=0 past=15 kept=204
```

**Site verification:**
- `events.json`: 204 events ✅
- `index.html`: 204 event articles ✅

## Recommendations

For production deployment:
1. Increase radius from 0.35km to 1-2km depending on desired coverage
2. Consider implementing secondary text-based filtering for venues near Plaza de España
3. Monitor parse failures in logs to catch future API format changes
