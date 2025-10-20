# Handoff: esmadrid.com Tourism Feed Integration

**Date:** 2025-10-20
**Status:** ⏸️ **PAUSED - Container rebuild required**
**Next Action:** Test esmadrid.com feed after container rebuild

---

## Summary

Discovered that current data source (datos.madrid.es dataset 300107) contains **cultural center programming**, not **Plaza de España outdoor events** (movies, concerts, gaming events). User identified esmadrid.com tourism feed as correct source, but it's currently blocked by firewall.

**Firewall updated** - ready for container rebuild to test new feed.

---

## What We Accomplished Today

### 1. Distrito-Based Filtering (Commit: 96792aa)

**Problem:** Only 13 events showing, 95% of events missing GPS coordinates

**Solution:** Implemented 3-tier filtering system:
1. **Priority 1**: Filter by distrito (CENTRO, MONCLOA-ARAVACA) - handles 95% of events
2. **Priority 2**: GPS radius (0.35km from Plaza de España) - for events with coordinates
3. **Priority 3**: Text matching (venue/address keywords) - fallback for missing data

**Key Fixes:**
- Fixed XML parsing to extract nested DISTRITO field from LOCALIZACION nodes
- Fixed pipeline merge to preserve distrito data from XML/CSV when JSON lacks it
- Added CSV distrito parsing from DISTRITO-INSTALACION column

**Results:** 13 → 161 events (12x improvement with distrito data)

**Files Changed:**
- `internal/event/event.go` - Added Distrito field to CanonicalEvent
- `internal/fetch/types.go` - Added Distrito to XMLEvent/CSVEvent, fixed extractAttributes
- `internal/fetch/client.go` - Parse DISTRITO-INSTALACION from CSV
- `internal/pipeline/pipeline.go` - Merge logic preserves distrito and other fields
- `cmd/buildsite/main.go` - 3-tier filtering implementation

---

### 2. Chronological Sorting (Commit: 5ccf10f)

**Problem:** Events displayed in arbitrary order (by ID or fetch order)

**Solution:** Sort filtered events by StartTime before rendering

```go
// Sort events by start time (upcoming events first)
sort.Slice(filteredEvents, func(i, j int) bool {
    return filteredEvents[i].StartTime.Before(filteredEvents[j].StartTime)
})
```

**Files Changed:**
- `cmd/buildsite/main.go` - Added sorting, imported "sort" package

---

### 3. Time-Based Filtering (Commit: 5ccf10f)

**Problem:** Old exhibitions from 2023-2024 appearing in results (long-running events with future end dates)

**Solution:** Filter by start time instead of end time - exclude events that started >2 weeks ago

```go
// Filter out events that started more than 2 weeks ago
twoWeeksAgo := now.AddDate(0, 0, -14)
if evt.StartTime.Before(twoWeeksAgo) {
    pastEvents++
    continue
}
```

**Results:** 161 → 142 events (removed 19 stale exhibitions)

**Files Changed:**
- `cmd/buildsite/main.go` - Changed from EndTime to StartTime filtering

---

### 4. Data Source Investigation

**Critical Discovery:** Dataset 300107 (Agenda de actividades) is **wrong type of events**

**Current events (142):**
- ✅ Teatro Español productions
- ✅ Conde Duque cultural center exhibitions
- ✅ Library workshops, museum programming
- ✅ Indoor venues IN CENTRO/MONCLOA-ARAVACA districts

**Missing events (what user wants):**
- ❌ Outdoor movies at Plaza de España
- ❌ Concerts and live music at the plaza
- ❌ Gaming events (Riot Games/League of Legends)
- ❌ Weekend public square programming

**Datasets Tested:**
- ❌ 300028 (Agenda turística) - HTTP 404
- ❌ 300378 (Fiestas populares) - HTTP 404
- ❌ 300401 (Actividades deportivas) - HTTP 404
- ✅ 300107 (Agenda actividades) - Works, but wrong content type

**User Solution:** Use esmadrid.com tourism feed instead
- URL: `https://www.esmadrid.com/opendata/agenda_v1_es.xml`
- Likely contains Plaza outdoor events from tourism perspective
- Currently blocked by firewall

---

## Firewall Update

**Problem:** Container firewall blocks www.esmadrid.com (connection refused on port 443)

**Solution:** Updated `.devcontainer/init-firewall.sh` to allow esmadrid.com

**Changes Made:**
```bash
# Added to allowed domains list (lines 77-78):
"www.esmadrid.com" \
"esmadrid.com" \
```

**File Changed:** `.devcontainer/init-firewall.sh`

**Action Required:** Rebuild container for firewall changes to take effect

---

## Next Steps (After Container Rebuild)

### Step 1: Download and Examine esmadrid.com Feed

```bash
# Download feed to fixture
mkdir -p /workspace/test/fixtures
curl -sL "https://www.esmadrid.com/opendata/agenda_v1_es.xml" \
  -o /workspace/test/fixtures/esmadrid-agenda.xml

# Check file size and structure
ls -lh /workspace/test/fixtures/esmadrid-agenda.xml
head -100 /workspace/test/fixtures/esmadrid-agenda.xml

# Search for Plaza de España events
grep -i "plaza.*españa" /workspace/test/fixtures/esmadrid-agenda.xml | head -20

# Count total events
grep -c "<item>" /workspace/test/fixtures/esmadrid-agenda.xml
```

### Step 2: Analyze XML Schema

**Questions to answer:**
1. What's the XML structure? (RSS? Custom format? ATOM?)
2. Field mapping:
   - Event ID → What field?
   - Title → What field?
   - Dates/times → What format?
   - Location → Address? Coordinates? Distrito?
   - Description → Available?
   - Event type/category → Available?
3. Does it have Plaza de España outdoor events?
4. Event count and date range?

### Step 3: Integration Decision

**If feed has Plaza events:**
1. Create new parser in `internal/fetch/types.go`:
   - `EsmadridEvent` struct
   - `ToCanonical()` method
2. Add esmadrid fetch to pipeline (`internal/pipeline/pipeline.go`)
3. Update main to include 4th source (JSON/XML/CSV/Esmadrid)
4. Test filtering with new events

**If feed still doesn't have Plaza events:**
1. Document findings in investigation log
2. Present options to user:
   - Option A: Pivot site to "Cultural Events Near Plaza de España"
   - Option B: Manual curation for Plaza events
   - Option C: Web scraping madrid.es website
   - Option D: Contact Madrid tourism directly

---

## Current Implementation Status

**Working Features:**
- ✅ 3-source pipeline (JSON/XML/CSV from datos.madrid.es)
- ✅ Distrito-based filtering (95% coverage)
- ✅ GPS radius fallback (for events with coordinates)
- ✅ Text matching fallback (for missing data)
- ✅ Chronological sorting (upcoming events first)
- ✅ Time filtering (exclude events started >2 weeks ago)
- ✅ Multi-source merge with field preservation
- ✅ Build report with detailed stats

**Current Results:**
- 142 events from CENTRO/MONCLOA-ARAVACA districts
- All at cultural venues (museums, theaters, libraries, cultural centers)
- Sorted chronologically, recent/upcoming only
- Build time: ~3-5 seconds

**Test Status:**
- 22 tests passing (100% success rate)
- All packages green (fetch, filter, render, snapshot, pipeline)

---

## Files Modified This Session

1. `.devcontainer/init-firewall.sh` - Added www.esmadrid.com to firewall allowlist
2. `cmd/buildsite/main.go` - Distrito filtering, sorting, time filter fixes
3. `internal/event/event.go` - Added Distrito field
4. `internal/fetch/types.go` - XMLEvent/CSVEvent Distrito field, extractAttributes fix
5. `internal/fetch/client.go` - CSV distrito parsing
6. `internal/pipeline/pipeline.go` - Field-preserving merge logic
7. `docs/logs/2025-10-20-distrito-filtering.md` - Implementation log
8. `docs/logs/2025-10-20-plaza-events-investigation.md` - Data source investigation
9. `docs/logs/2025-10-20-handoff-esmadrid-integration.md` - This handoff doc

---

## Recent Commits

**96792aa** - `feat: implement distrito-based filtering for Plaza de España events`
- Add Distrito field to CanonicalEvent, XMLEvent, and CSVEvent
- Parse DISTRITO field from XML/CSV sources (906+ events have distrito data)
- Fix pipeline merge to preserve distrito from XML/CSV when JSON lacks it
- Implement 3-tier filtering: distrito → GPS radius → text matching
- Results: 152 events (was 13) - 12x improvement!

**5ccf10f** - `fix: sort events chronologically and filter old events by start time`
- Sort filtered events by StartTime for better UX
- Change time filtering to use StartTime instead of EndTime
- Exclude events that started >2 weeks ago (not just ended)
- Results: 142 events (removed 19 stale exhibitions from 2023-2024)

---

## Documentation Created

**Comprehensive logs in `/workspace/docs/logs/`:**
1. `2025-10-20-distrito-filtering.md` - Distrito implementation details
2. `2025-10-20-plaza-events-investigation.md` - Why current API doesn't have Plaza events
3. `2025-10-20-handoff-esmadrid-integration.md` - This handoff document

**Key findings:**
- datos.madrid.es has cultural center events, NOT public square events
- esmadrid.com tourism feed is likely correct source for Plaza outdoor events
- Alternative datasets (300028, 300378, 300401) all return HTTP 404

---

## Testing After Rebuild

**Verification checklist:**

1. **Firewall works:**
   ```bash
   curl -I "https://www.esmadrid.com"
   # Should get HTTP 200 or 301, NOT connection refused
   ```

2. **Feed downloads:**
   ```bash
   curl -sL "https://www.esmadrid.com/opendata/agenda_v1_es.xml" \
     -o /workspace/test/fixtures/esmadrid-agenda.xml
   ls -lh /workspace/test/fixtures/esmadrid-agenda.xml
   # Should show file size > 0
   ```

3. **Current build still works:**
   ```bash
   just build
   just test
   # 22 tests should pass
   ```

4. **Examine new feed:**
   ```bash
   # Structure
   head -100 /workspace/test/fixtures/esmadrid-agenda.xml

   # Plaza events
   grep -i "plaza.*españa" /workspace/test/fixtures/esmadrid-agenda.xml

   # Event count
   grep -c "<item>" /workspace/test/fixtures/esmadrid-agenda.xml || \
   grep -c "<event>" /workspace/test/fixtures/esmadrid-agenda.xml
   ```

---

## Open Questions (To Answer After Rebuild)

1. **Does esmadrid.com feed have Plaza de España outdoor events?**
   - Movies, concerts, gaming events that happen at the plaza itself?

2. **What's the XML schema?**
   - RSS 2.0? ATOM? Custom Madrid format?
   - Field names for ID, title, date, location, description?

3. **Event coverage:**
   - How many events total?
   - Date range (historical? future only?)
   - Geographic coverage (all Madrid? tourist areas only?)

4. **Data quality:**
   - Structured location data (coordinates, distrito, address)?
   - Rich descriptions?
   - Event categories/types?

5. **Integration strategy:**
   - Replace datos.madrid.es entirely?
   - Add as 4th source and merge?
   - Use esmadrid for Plaza events, datos.madrid for cultural centers?

---

## Context for Next Session

**User Intent:**
> "I want to find events at plaza espana. They have some like every weekend. a few weeks ago there was a riot LOL thing there. there's movies and music. We must be on the wrong feeds."

**Current Problem:**
- We're filtering events at venues IN districts (museums, theaters, libraries)
- User wants events AT the public square itself (outdoor programming)
- datos.madrid.es doesn't publish Plaza outdoor events

**User's Solution:**
> "use this feed: https://www.esmadrid.com/opendata/agenda_v1_es.xml"

**Blocker:** Firewall blocking esmadrid.com

**Status:** Firewall updated, container rebuild required

---

## Commands Reference

```bash
# After rebuild - test firewall
curl -I "https://www.esmadrid.com"

# Download feed
curl -sL "https://www.esmadrid.com/opendata/agenda_v1_es.xml" \
  -o /workspace/test/fixtures/esmadrid-agenda.xml

# Examine structure
head -150 /workspace/test/fixtures/esmadrid-agenda.xml

# Search for Plaza events
grep -i "plaza" /workspace/test/fixtures/esmadrid-agenda.xml | head -20

# Current build (should still work)
just build
just test

# View current events
cat public/events.json | jq '.[0:5]'
```

---

## Build Stats (Before esmadrid Integration)

**Last successful build:**
```
JSON: 1055 events, 0 errors
XML: 1002 events, 0 errors
CSV: 1002 events, 0 errors
After merge: 1055 unique events from 3059 total (65.5% deduplication)

Filtering:
- Filtered by distrito: 161
- Filtered by radius: 0
- Filtered by text: 2
- Past events excluded: 21
- Final events: 142

Build time: ~3.5 seconds
Output: index.html (34 KB), events.json (8 KB)
```

**Source coverage (of 142 events):**
- In all 3 sources: 85%
- In 2 sources: 12%
- In 1 source only: 3%

---

## Reminder: Git State

**Current branch:** main

**Uncommitted changes:**
- `.devcontainer/init-firewall.sh` - Added esmadrid.com to allowlist

**After testing esmadrid feed, commit as:**
```bash
git add .devcontainer/init-firewall.sh
git commit -m "feat: add esmadrid.com to firewall for tourism events feed

Allow access to www.esmadrid.com and esmadrid.com to fetch Madrid
tourism events feed (agenda_v1_es.xml). This feed should contain
Plaza de España outdoor events (movies, concerts, gaming events)
that are missing from datos.madrid.es cultural events feed.

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## End of Handoff

**Resume Point:** Container rebuild → test esmadrid.com feed → analyze schema → integration decision

**Contact:** All documentation in `/workspace/docs/logs/2025-10-20-*.md`
