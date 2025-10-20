# Madrid Open Data Catalog Analysis

**Date:** 2025-10-20
**Source:** catalogo.csv (653 datasets)
**Goal:** Identify relevant datasets to enrich Plaza de España events site

## Summary

Found **8 highly relevant datasets** for event enrichment, including:
- ✅ **3 currently using** (Agenda de actividades y eventos - our 300107 API)
- 🎯 **5 new candidates** for enrichment

**Top Recommendation:** **"Agenda turística"** dataset - same tourist API we tried earlier, but now we have confirmation it exists and is actively maintained (19,565 downloads, daily updates).

## Currently Used Dataset

### 1. **Agenda de actividades y eventos** ✅ CURRENT
- **ID:** 300107 (from earlier investigation)
- **Sector:** cultura-ocio
- **Keywords:** eventos, fiestas, actividades, www.madrid.es
- **Update Frequency:** Continua (continuous)
- **Formats:** CSV, JSON, API, RDF, GEO, XML
- **Downloads:** 306,487 ⭐ (most popular)
- **Responsibility:** Dirección General de Atención a la Ciudadanía
- **URL:** https://datos.madrid.es/portal/site/egob/?vgnextoid=57be24206a91b510VgnVCM2000001f4a900aRCRD

**Status:** Currently fetching JSON, XML, CSV in parallel. Working well (1055 unique events merged).

---

## NEW CANDIDATE DATASETS

### 2. **Agenda turística de la ciudad de Madrid** 🎯 TOP PRIORITY
- **Sector:** turismo
- **Keywords:** turismo, agenda turística, actividades turísticas, teatro cine conciertos ballet
- **Update Frequency:** Diaria (daily)
- **Formats:** XML
- **Downloads:** 19,565
- **Responsibility:** Madrid Destino, Cultura, Turismo y Negocio S.A.
- **URL:** https://datos.madrid.es/portal/site/egob/?vgnextoid=30c1a0d4c16f3510VgnVCM1000001d4a900aRCRD

**This is the 300028 dataset we investigated earlier!**

**Known Features (from earlier investigation):**
- ✅ Event photographs
- ✅ Detailed descriptions
- ✅ 7 languages (ES, EN, FR, DE, IT, PT, RU)
- ✅ GPS coordinates
- ⚠️ Photo usage restrictions

**Next Step:** Try downloading again with specific language code XMLs found in HTML.

### 3. **Actividades Culturales y de Ocio Municipal (100 días)**
- **Sector:** cultura-ocio
- **Keywords:** cultura ocio eventos fiestas actividades, www.madrid.es
- **Update Frequency:** Continua
- **Formats:** CSV, JSON, API, RDF, GEO, XML
- **Downloads:** 325,005 ⭐ (very popular)
- **URL:** https://datos.madrid.es/portal/site/egob/?vgnextoid=6c0b6d01df986410VgnVCM2000000c205a0aRCRD

**Potential Value:**
- Similar to 300107 but filtered to next 100 days
- May have same data as our current source
- **Recommendation:** Low priority - likely duplicate of what we have

### 4. **Agenda de actividades deportivas**
- **Sector:** deporte
- **Keywords:** deporte, agenda deportiva, eventos deportivas
- **Update Frequency:** Continua
- **Formats:** CSV, JSON, API, RDF, GEO, XML
- **Downloads:** 134,788
- **URL:** https://datos.madrid.es/portal/site/egob/?vgnextoid=b802209e501cb410VgnVCM1000000b205a0aRCRD

**Potential Value:**
- Sports events (might overlap with current data)
- Good geographic data likely
- **Recommendation:** Medium priority - would add sports events near Plaza de España

### 5. **Monumentos de la ciudad de Madrid**
- **Sector:** cultura-ocio
- **Keywords:** monumentos, www.madrid.es
- **Update Frequency:** Continua
- **Formats:** CSV, JSON, API, RDF, GEO, XML
- **Downloads:** 24,891
- **URL:** https://datos.madrid.es/portal/site/egob/?vgnextoid=eb8e993ae322b610VgnVCM1000001d4a900aRCRD

**Potential Value:**
- Static POIs (not events)
- Could enrich venue information
- **Recommendation:** Low priority for events site, but could show nearby monuments

### 6. **Museos de la ciudad de Madrid**
- **Sector:** cultura-ocio
- **Keywords:** museos, cultura, www.madrid.es
- **Update Frequency:** Continua
- **Formats:** CSV, JSON, API, RDF, GEO, XML
- **Downloads:** 66,798
- **URL:** https://datos.madrid.es/portal/site/egob/?vgnextoid=118f2fdbecc63410VgnVCM1000000b205a0aRCRD

**Potential Value:**
- Museum catalog (not events, but could show which museums host events)
- Could cross-reference with event venues
- **Recommendation:** Medium priority - enrich venue data

### 7. **Puntos de Interés turístico** (www.esmadrid.com)
- **Sector:** turismo
- **Keywords:** museos monumentos exposiciones, arte en madrid
- **Update Frequency:** Diaria
- **Formats:** XML
- **Downloads:** 10,602
- **URL:** https://datos.madrid.es/portal/site/egob/?vgnextoid=3b70a73970504510VgnVCM2000001f4a900aRCRD

**Potential Value:**
- Tourist POIs (not events)
- Could show nearby attractions
- **Recommendation:** Low priority - complementary to events

### 8. **Eventos de carácter comercial en CentroCentro**
- **Sector:** cultura-ocio
- **Keywords:** Madrid Destino, Palacio de Telecomunicaciones, Galería de cristal
- **Update Frequency:** Anual
- **Formats:** CSV, XLS
- **Downloads:** 2,041
- **URL:** https://datos.madrid.es/portal/site/egob/?vgnextoid=1716dca940144610VgnVCM2000001f4a900aRCRD

**Potential Value:**
- Very specific to CentroCentro venue
- Likely already covered by 300107
- **Recommendation:** Skip - too narrow

---

## Prioritized Recommendations

### Priority 1: Investigate Tourist Agenda (Retry)

**Dataset:** Agenda turística de la ciudad de Madrid
**Action:** Retry download with specific XML URLs from catalog page

**Why:**
- 19,565 downloads confirms active use
- Daily updates
- Photos and rich data
- This IS the 300028 dataset we wanted

**Download URLs to try:**
```
https://datos.madrid.es/egob/catalogo/300028-10037314-agenda-turismo.xml (Spanish - most downloads)
https://datos.madrid.es/egob/catalogo/300028-10037315-agenda-turismo.xml (English)
...etc
```

**Next Steps:**
1. Download Spanish XML sample
2. Parse structure
3. Check for images, categories
4. Verify GPS coverage for Plaza de España
5. Check license terms for photos
6. Decide: 4th source vs replacement

### Priority 2: Test Sports Events API

**Dataset:** Agenda de actividades deportivas
**Action:** Download sample, check Plaza de España coverage

**Why:**
- Might have different events than cultural agenda
- Same data structure likely (same portal)
- 134K downloads = popular

**API Endpoints (predicted based on pattern):**
```
https://datos.madrid.es/egob/catalogo/[ID]-0-agenda-deportes.json
https://datos.madrid.es/egob/catalogo/[ID]-0-agenda-deportes.xml
```

### Priority 3: Cross-Reference Museums & Monuments

**Datasets:** Museos + Monumentos
**Action:** Download and use as venue enrichment (not events)

**Why:**
- Can enrich event venue information
- Show "this event is at [Museum Name], a historic site..."
- Add context without needing photos

---

## Feasibility Assessment

### Dataset #2: Tourist Agenda (300028) - RETRY

**Pros:**
- ✅ Actively maintained (daily updates)
- ✅ Popular (19,565 downloads)
- ✅ Photos + rich data confirmed
- ✅ We know the exact XML URLs now
- ✅ Multilingual support

**Cons:**
- ⚠️ Previous download attempts failed (network/container issue?)
- ⚠️ Photo license restrictions
- ❓ Unknown Plaza de España coverage
- ❓ Unknown data structure

**Recommendation:** **RETRY with specific URLs**

### Dataset #4: Sports Events

**Pros:**
- ✅ Same data portal (likely same structure)
- ✅ Different event type (sports vs cultural)
- ✅ Continuous updates

**Cons:**
- ❓ Unknown Plaza de España relevance
- ❓ May overlap with 300107

**Recommendation:** **Test if tourist API fails**

### Datasets #5, #6, #7: POI Data

**Pros:**
- ✅ Can enrich venues without needing new events
- ✅ Well-established datasets

**Cons:**
- ❌ Not events (static POIs)
- ⚠️ Adds complexity for marginal value

**Recommendation:** **DEFER - focus on events first**

---

## Immediate Action Plan

### Step 1: Download Tourist Agenda Sample (PRIORITY)

```bash
# Try the Spanish XML with explicit URL from catalog
curl -o data/investigation/tourist-spanish.xml \
  "https://datos.madrid.es/egob/catalogo/300028-10037314-agenda-turismo.xml"

# If that fails, try from portal URL
curl -L -o data/investigation/tourist-portal.xml \
  "https://datos.madrid.es/portal/site/egob/menuitem.c05c1f754a33a9fbe4b2e4b284f1a5a0/?vgnextoid=30c1a0d4c16f3510VgnVCM1000001d4a900aRCRD&vgnextchannel=374512b9ace9f310VgnVCM100000171f5a0aRCRD&vgnextfmt=default"
```

**Success Criteria:**
- File size > 0 bytes
- Valid XML structure
- Contains event elements with photos

### Step 2: Parse and Analyze Structure

If download succeeds:
1. Identify XML schema
2. Map fields to CanonicalEvent
3. Check for image URLs
4. Count events in Plaza de España radius
5. Check ID format for merging

### Step 3: Decide Integration Strategy

Based on findings:

**Option A: 4th Parallel Source** (if data is complementary)
- Tourist events have different IDs than municipal
- Minimal overlap
- Adds photos and international coverage
→ Merge with existing 3 sources

**Option B: Replace Current Sources** (if data is superset)
- Tourist API contains all municipal events
- Same IDs, just more data
→ Replace JSON/XML/CSV with tourist API

**Option C: Keep Current** (if blocked or low value)
- Can't download reliably
- License issues
- No Plaza de España events
→ Enhance current data instead

---

## Alternative Enhancement Paths

If tourist API still fails:

### Path 1: Enhance Current Data Without New API

- Extract categories from existing TIPO/TIPO-EQUIPAMIENTO fields
- Better date formatting (show end dates)
- Distance from Plaza de España
- "Add to Calendar" links
- Better mobile UI

### Path 2: Use POI Datasets for Context

- Download Museums + Monuments data
- Cross-reference event venues
- Show "Event at [Museum Name], built in 1850..."
- Add map links to nearby attractions

### Path 3: Sports Events as Alternative

- If cultural tourist API is blocked
- Try sports agenda instead
- Different event type, might work better

---

## Technical Notes

### URL Pattern Discovery

Found that catalog has specific download links:
- Not `/egob/catalogo/[ID]-0-[name].xml`
- But `/egob/catalogo/[ID]-[LANG_CODE]-[name].xml`

**Language codes found:**
- 10037314 (Spanish - 5,562 downloads)
- 10037315 (English - 2,615 downloads)
- 10037316, 10037318, 10037319, 10037320 (other languages)

### Data Quality Indicators

**High confidence datasets:**
- Download count > 10,000 = widely used
- Update frequency "continua" or "diaria" = actively maintained
- Multiple formats available = well-supported

**Red flags:**
- 0 downloads in recent period
- Annual updates (stale data)
- Single format only

---

## Conclusion

**Primary Goal:** Get tourist agenda (300028) working
**Backup Plan:** Sports events or enhance current data
**Long-term:** POI enrichment for venue context

**Immediate action:** Retry tourist API download with specific URL: `300028-10037314-agenda-turismo.xml`
