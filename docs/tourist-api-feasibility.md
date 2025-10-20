# Tourist API (300028) Feasibility Assessment

**Date:** 2025-10-20
**Status:** BLOCKED - Unable to download sample data due to network issues

## Investigation Summary

### Download Attempts

Attempted to download tourist API data from multiple endpoints:
- `https://datos.madrid.es/egob/catalogo/300028-0-agenda-turismo.json` - Empty response
- `https://datos.madrid.es/egob/catalogo/300028-0-agenda-turismo.xml` - Empty response
- `https://datos.madrid.es/egob/catalogo/300028-10037314-agenda-turismo.xml` - Timeout

**Issue:** All download attempts resulted in either:
- Empty files (0 bytes)
- Network timeouts (>30 seconds)
- No response from server

**Hypothesis:**
1. Tourist API endpoint may not be publicly accessible from our network
2. API may require authentication or different access pattern
3. Dataset ID format may have changed
4. Dataset may no longer be actively maintained

### What We Know from Documentation

From previous web search results (datos.gob.es and datos.madrid.es):

**Tourist Agenda Dataset (300028):**
- **Content:** Exhibitions, musicals, theater, dance, children's activities, fairs, congresses, concerts, sports events
- **Source:** esmadrid.com tourism portal
- **Data Included:**
  - Basic event data
  - Detailed descriptions
  - Geographic position and postal address
  - **Photographs for each event** ✅
  - Opening hours
  - Access costs (where applicable)

- **Languages:** Spanish, English, French, German, Italian, Portuguese, Russian
- **Formats:** XML (multiple language variants: 10037314, 10037315, 10037316, etc.)

**LICENSE RESTRICTIONS:**
- ⚠️ **Texts are free to use**
- ⚠️ **Photographs have usage restrictions** (not fully free)

### Comparison with Municipal API (300107)

| Feature | Municipal API (300107) | Tourist API (300028) | Status |
|---------|----------------------|---------------------|--------|
| **Data Access** | ✅ Working | ❌ Blocked | ISSUE |
| **Event Types** | Municipal activities | Tourist attractions | Different coverage |
| **Geographic Scope** | All municipal centers | Tourist venues | Unknown overlap |
| **Images** | ❌ No | ✅ Yes (restricted) | Major feature |
| **Descriptions** | ✅ Basic | ✅ Detailed | Improvement |
| **Coordinates** | ✅ Yes | ✅ Yes | Same |
| **Multilingual** | ❌ Spanish only | ✅ 7 languages | Nice-to-have |
| **License** | ✅ Open data | ⚠️ Photos restricted | Legal concern |

## Critical Questions (UNANSWERED)

Due to inability to download sample data, these questions remain:

### 1. Data Structure
- ❓ What is the XML/JSON schema?
- ❓ Does it use same ID-EVENTO field?
- ❓ How are images referenced (URL field)?
- ❓ What are category/type fields called?
- ❓ Is geographic data in same format (lat/lon)?

### 2. Geographic Coverage
- ❓ Do tourist events include Plaza de España area?
- ❓ What is the geographic distribution?
- ❓ How many events would pass our 0.35km radius filter?

### 3. Event Overlap
- ❓ Is there overlap with municipal API (300107)?
- ❓ Can we merge on ID-EVENTO or are IDs different?
- ❓ Would we get duplicates or complementary events?

### 4. Data Quality
- ❓ How current is the data?
- ❓ Update frequency?
- ❓ Are GPS coordinates reliable?
- ❓ What percentage of events have photos?

### 5. Photo Usage Rights
- ❓ Exact license terms for photographs?
- ❓ Attribution requirements?
- ❓ Commercial use restrictions?
- ❓ Can we use for open-source personal project?

## Recommendations

### Option A: Alternative Investigation Methods

Since direct download is blocked, try:

1. **Manual browser download**
   - Visit https://datos.madrid.es/portal/site/egob
   - Search for dataset 300028
   - Download XML/JSON file manually via browser
   - Upload to repository for analysis

2. **Try from different network**
   - VPN or different ISP
   - Direct connection (not container)
   - May be geo-restricted or rate-limited

3. **Contact datos.madrid.es support**
   - Ask for current API endpoint
   - Clarify photo usage restrictions
   - Verify dataset is still maintained

### Option B: Proceed Without Tourist API

**Given:**
- Current municipal API (300107) is working
- We successfully fetch 1055 unique events
- After filtering: 13 events in Plaza de España area
- We already added descriptions to event cards

**Argument:**
- Tourist API adds **photos** but has **legal restrictions**
- Unknown if it would add events in our target area (Plaza de España)
- Network access issues suggest maintenance/reliability concerns
- Current implementation is already production-ready

**Recommendation:**
**DEFER tourist API integration** until we can:
1. Successfully download and analyze sample data
2. Verify photo license compatibility
3. Confirm geographic coverage overlap
4. Assess actual value-add for our specific use case (Plaza de España area)

### Option C: Focus on Enhancing Current Data

Instead of adding tourist API, enhance current events with:

**Immediate wins (no new API):**
1. ✅ **Descriptions** - DONE (Task 2)
2. ✅ **Build report** - DONE (Task 3)
3. **Category badges** - Extract from existing TIPO field (if available)
4. **Date range display** - Show FECHA-FIN if multi-day event
5. **Better mobile styling** - Improve responsive design
6. **Event sorting** - By date, distance, venue

**Enrichment without photos:**
- Pull venue details from separate Madrid POI datasets
- Add map link (Google Maps) using GPS coordinates
- Show distance from Plaza de España
- Add "Add to Calendar" functionality

## Next Steps

### Immediate (Required before proceeding)

1. **Obtain tourist API sample data**
   - Try manual browser download
   - Or use curl from host machine (not container)
   - Need at least 1 complete sample file to analyze

2. **Review license terms**
   - Read full legal text at datos.madrid.es
   - Determine if photos can be used for:
     - Personal website
     - Open-source project
     - GitHub Pages deployment

### If Sample Data Obtained

1. Parse XML/JSON structure
2. Map fields to CanonicalEvent
3. Check ID overlap with 300107
4. Count events in Plaza de España radius
5. Assess if 4th source vs replacement

### If Sample Data Still Blocked

1. Document as "investigated but blocked"
2. Update CLAUDE.md to note tourist API exists but unavailable
3. Close investigation
4. Focus on enhancing current data

## Conclusion

**Cannot recommend tourist API integration at this time** due to:
- ❌ Unable to access sample data for analysis
- ❌ Unknown photo license restrictions
- ❌ Unknown geographic coverage
- ❌ Unknown data structure/schema
- ⚠️ Possible API endpoint changes or deprecation

**Current status is production-ready without tourist API.**

**Action required:** Obtain sample tourist API data file before any further evaluation.
