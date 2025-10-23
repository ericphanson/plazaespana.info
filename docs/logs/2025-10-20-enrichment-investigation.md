# Data Enrichment Investigation

**Date:** 2025-10-20
**Goal:** Determine if event cards can be enriched with additional data (images, categories, detailed information)

## Summary

Madrid provides multiple open data APIs for events. The current implementation uses the **municipal activities API** (300107) which has basic event information but **no images**. A separate **tourist agenda API** (300028) exists with richer data including **photographs**, but with usage restrictions.

## Key Findings

### 1. Current API: Municipal Activities (300107)

**URL:** https://datos.madrid.es/egob/catalogo/300107-0-agenda-actividades-eventos.json

**Available Fields:**
- ID-EVENTO (event ID)
- TITULO (title)
- DESCRIPCION (description) ✅ **Already using**
- FECHA / FECHA-FIN (dates)
- HORA (time)
- NOMBRE-INSTALACION (venue name)
- COORDENADA-LATITUD / COORDENADA-LONGITUD (GPS coordinates)
- CONTENT-URL (link to details page)
- DIRECCION (address)

**Missing:**
- ❌ No image/photo fields
- ❌ No category/tag fields
- ❌ Limited metadata

**Coverage:** Events in municipal centers (Cultural Centers, Libraries, Youth Centers, Senior Centers, Museums)

### 2. Tourist Agenda API (300028) - RICHER DATA

**URL:** https://datos.madrid.es/egob/catalogo/300028-0-agenda-turismo.{json|xml}

**Available Formats:**
- JSON, XML
- Multiple languages (ES, EN, FR, DE, IT, PT, RU)

**Additional Data:**
- ✅ **Photographs** (set of images per event)
- ✅ Detailed descriptions
- ✅ GPS coordinates and postal address
- ✅ Opening hours
- ✅ Access costs
- ✅ Multilingual support

**Coverage:** Tourist events (exhibitions, musicals, theater, dance, children's activities, fairs, congresses, concerts, sports)

**IMPORTANT RESTRICTION:**
- ⚠️ **Texts are free to use, but photographs have usage restrictions**
- Must review terms of service before implementing image display

**Overlap with 300107:**
- Unknown - needs investigation
- Likely covers different event types (tourist vs municipal activities)

### 3. CONTENT-URL Pattern

All events have CONTENT-URL field linking to details page:
```
http://www.madrid.es/sites/v/index.jsp?vgnextchannel=...&vgnextoid=...
```

**Investigation Status:**
- ❌ Unable to fetch pages (network timeout)
- ❓ Unknown if pages have structured data (JSON-LD, Schema.org, OpenGraph)
- ❓ Scraping feasibility unknown

**Recommendation:** Low priority - scraping HTML is fragile and maintenance-heavy

### 4. Data Source Comparison

| Feature | Municipal API (300107) | Tourist API (300028) |
|---------|----------------------|---------------------|
| Events | Municipal activities | Tourist attractions |
| Images | ❌ No | ✅ Yes (restricted) |
| Categories | ❌ No | ✅ Likely |
| Descriptions | ✅ Basic | ✅ Detailed |
| Multilingual | ❌ Spanish only | ✅ 7 languages |
| License | ✅ Open data | ⚠️ Photos restricted |

## Recommendations

### Option 1: Add Tourist API as 4th Source (RECOMMENDED)

**Approach:**
- Fetch from 300028 tourist API as 4th parallel source
- Parse event structure (JSON/XML)
- Merge with existing 3 sources using same deduplication logic
- Extract image URLs and categories from tourist data
- Only display images if event comes from tourist API source

**Pros:**
- Clean architecture - fits existing pipeline
- Rich data (images, categories, multilingual)
- No scraping needed

**Cons:**
- Photo usage restrictions must be reviewed
- May have limited overlap with municipal events (Plaza de España focus)
- Need to verify tourist events include our target area

**Effort:** 4-6 hours
1. Parse tourist API format (1h)
2. Add as 4th source in pipeline (1h)
3. Update CanonicalEvent to include ImageURL and Categories fields (1h)
4. Update HTML template to display images (1h)
5. Update CSS for image cards (1h)
6. Test and handle merge logic (1h)

### Option 2: Scrape CONTENT-URL Pages

**Approach:**
- Follow CONTENT-URL for each event
- Parse HTML for structured data (JSON-LD, OpenGraph, images)
- Enrich CanonicalEvent after initial fetch

**Pros:**
- Works with existing data source
- May find additional metadata

**Cons:**
- ❌ Fragile - HTML structure may change
- ❌ Performance - 13+ HTTP requests per build
- ❌ Maintenance burden
- ❌ Network timeouts already observed

**Recommendation:** **NOT RECOMMENDED** - too brittle

### Option 3: Categories from TIPO-EQUIPAMIENTO Field

**Current Status:** Need to check if 300107 API has category/type fields

**Approach:**
- Extract event type from existing API
- Create category badges on cards

**Pros:**
- No additional API calls
- Simple implementation

**Cons:**
- Limited to municipal event types
- No images

**Effort:** 2 hours if field exists

## Next Steps

### Immediate (Before Implementation)

1. **Review Tourist API License**
   - Read full terms at datos.madrid.es for 300028 dataset
   - Confirm photo usage restrictions
   - Determine if acceptable for personal/open-source project

2. **Fetch Sample Tourist Data**
   - Download https://datos.madrid.es/egob/catalogo/300028-0-agenda-turismo.json
   - Inspect structure for image URLs and categories
   - Check geographic coverage (does it include Plaza de España events?)

3. **Check Field Overlap**
   - Compare ID fields between 300107 and 300028
   - Determine merge strategy (same ID-EVENTO? different?)

### Implementation Plan (If Approved)

**Phase 1: Extend Pipeline (2h)**
- Create `internal/fetch/tourist.go` parser
- Add as 4th source in pipeline
- Update merge logic to handle tourist events

**Phase 2: Extend CanonicalEvent (1h)**
```go
type CanonicalEvent struct {
    // ... existing fields ...
    ImageURL   string   // Primary image URL
    Categories []string // Event types/tags
}
```

**Phase 3: Update Rendering (2h)**
- Update TemplateEvent to include ImageURL
- Add image display to HTML template
- Add CSS for image cards
- Ensure graceful fallback when no image

**Phase 4: Testing (1h)**
- Verify tourist events merge correctly
- Check image display
- Validate geographic filtering still works

## Constraints

**Must Maintain:**
- Post-canonicalization architecture
- All enrichment on CanonicalEvent, not in parsers
- Geographic filtering (Plaza de España focus)
- Atomic writes and fallback resilience

**Photo License Compliance:**
- If using tourist API photos: must display attribution
- May need to add terms/conditions link in footer
- Verify commercial use is not required for this project

## Open Questions

1. What are the exact photo usage restrictions for 300028 API?
2. Do tourist events (300028) overlap with municipal events (300107)?
3. Does 300028 cover events near Plaza de España?
4. What ID scheme does 300028 use - can we merge on ID-EVENTO?
5. Are there category/type fields in the existing 300107 API we're not using?
