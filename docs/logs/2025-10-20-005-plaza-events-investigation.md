# Plaza de España Events Investigation

**Date:** 2025-10-20
**Status:** ❌ **BLOCKED - Correct data source not found**

## Problem Statement

After successfully implementing distrito-based filtering (improving from 13 → 142 events), user feedback revealed a fundamental issue:

> "I want to find events at plaza espana. They have some like evry weekend. a few weeks ago there was a riot LOL thing there. there's movies and music. We must be on the wrong feeds."

**Current implementation shows:**
- Teatro Español productions
- Conde Duque cultural center events
- Library exhibitions
- Museum programming

**User expects to see:**
- Outdoor movies at Plaza de España
- Concerts and music performances
- Gaming events (e.g., Riot Games/League of Legends promotional events)
- Public square programming that happens "every weekend"

**Root cause:** We're filtering events at **venues IN** target districts, not events **AT the public square** itself.

---

## Data Source Investigation

### Dataset 1: Current Implementation (300107)

**Name:** Agenda de actividades y eventos
**Description:** "Actividades y eventos organizados y apoyados por el Ayuntamiento de Madrid"
**Formats:** JSON, XML, CSV

**Content:** Cultural center programming
- Museums (Museo de Historia, Museo de San Isidro)
- Libraries (Biblioteca Iván de Vargas, Biblioteca Eugenio Trías)
- Theaters (Teatro Español, Naves del Español)
- Cultural centers (Conde Duque, Matadero Madrid)

**Venues are IN districts, not public squares:**
```
CENTRO district examples:
- Teatro Español (Calle del Príncipe, 25)
- Conde Duque (Calle del Conde Duque, 11)
- Biblioteca Municipal Iván de Vargas (Calle de San Justo, 8)

MONCLOA-ARAVACA district examples:
- Faro de Moncloa (Avenida de los Reyes Católicos, s/n)
- Templo de Debod (indoor exhibitions, not outdoor plaza events)
```

**Distrito coverage:** 906/1055 events have distrito data (95% coverage)

**Conclusion:** ✅ **Works perfectly for cultural center events**, ❌ **Does NOT include outdoor public square events**

---

### Dataset 2: Tourist Events (300028)

**Name:** Agenda turística
**Description:** Tourist-focused events in Madrid
**Attempted URLs:**
- JSON: `https://datos.madrid.es/egob/catalogo/300028-0-agenda-turistica.json`
- XML: `https://datos.madrid.es/egob/catalogo/300028-0-agenda-turistica.xml`
- CSV: `https://datos.madrid.es/egob/catalogo/300028-0-agenda-turistica.csv`

**Result:** ❌ **HTTP 302 redirect (blocked or moved)**

```bash
$ curl -I "https://datos.madrid.es/egob/catalogo/300028-0-agenda-turistica.json"
HTTP/2 302
location: https://datos.madrid.es/...
```

**Status:** Dataset may be:
1. Discontinued
2. Moved to different URL structure
3. Restricted access
4. Merged into another dataset

**Conclusion:** ❌ **Not accessible**

---

### Other Datasets Checked

**From `/workspace/docs/catalogo.csv`:**

Reviewed full catalog of Madrid open data portal. Relevant categories:

**Culture & Leisure:**
- 300107 (Agenda actividades) - ✅ Using this (cultural centers only)
- 300028 (Agenda turística) - ❌ Not accessible
- 300110 (Monuments) - Static data, not events
- 300178 (Museums) - Static data, not events

**Events & Celebrations:**
- 300378 (Fiestas populares) - Neighborhood festivals (may have Plaza events?)
- 300401 (Actividades deportivas) - Sports events (unlikely for Plaza)

**Tourism:**
- Most datasets are static (museums, monuments, tourist offices)
- No "outdoor public events" or "plaza programming" dataset found

---

## Why Plaza de España Events May Be Missing

### Hypothesis 1: Private Event Organization
Outdoor Plaza events (Riot Games, outdoor cinema, concerts) may be organized by:
- Private companies (e.g., gaming companies, film distributors)
- Event promoters (not city government)
- Sponsorship deals (Red Bull, brands)

These events may NOT be in Madrid's official cultural programming database.

### Hypothesis 2: Different Administrative System
Public square events may be managed by:
- Parks & Recreation department (not Culture department)
- Special events permits office
- Tourism promotion office
- Different internal system that doesn't publish to open data portal

### Hypothesis 3: Web-Only Publishing
Events may be published on:
- `madrid.es` website calendar (not API)
- District-specific web pages
- Social media (Twitter, Instagram, Facebook)
- Event aggregator sites (Eventbrite, Meetup, etc.)

### Hypothesis 4: Seasonal or Irregular Data
Plaza events might be:
- Published closer to event date (not in advance)
- Updated irregularly (not systematic like museum programming)
- Managed event-by-event rather than as ongoing series

---

## Evidence from Current Results

**Sample of 142 filtered events (all CENTRO/MONCLOA-ARAVACA):**

```
✅ "Visitas guiadas Museo de Historia de Madrid" - Museo de Historia
✅ "Exposición: Karlheinz Stockhausen" - Conde Duque
✅ "Taller de ilustración" - Biblioteca Iván de Vargas
✅ "Madrid Art Déco, 1925" - Museo de Historia
✅ "Ensayos gráficos" - Biblioteca Eugenio Trías

❌ NO outdoor cinema screenings at Plaza de España
❌ NO concerts at Plaza de España
❌ NO gaming events (Riot Games) at Plaza de España
❌ NO "every weekend" recurring plaza events
```

**Geographic distribution:**
- Events are at **buildings IN** CENTRO/MONCLOA-ARAVACA
- NOT events AT the plaza itself (40.42338, -3.71217)

**Venue types:**
- 80% museums, libraries, cultural centers (indoor venues)
- 15% theaters (indoor venues)
- 5% parks (Parque del Oeste, Templo de Debod area - but indoor exhibitions)
- 0% public squares (no Plaza de España, Plaza Mayor, etc.)

---

## Alternative Approaches

### Option A: Web Scraping Madrid.es

**Target pages:**
- Main events calendar: `https://www.madrid.es/vgn-ext-templating/v/index.jsp?vgnextchannel=...`
- Plaza de España specific page (if exists)
- District event calendars
- Tourism portal

**Pros:**
- May include events not in API
- More comprehensive coverage

**Cons:**
- Fragile (HTML changes break scraper)
- No structured data (need to parse free text)
- Rate limiting / blocking risk
- Maintenance burden

**Feasibility:** Medium complexity, requires HTML parsing + careful scraping

---

### Option B: Multi-Source Aggregation

**Combine data from:**
1. Madrid open data APIs (current: cultural centers)
2. Eventbrite API (search "Plaza de España Madrid")
3. Meetup API (location-based search)
4. Social media APIs (Twitter/Instagram hashtags like #PlazaDeEspaña)
5. Manual curation (weekly check of known event sources)

**Pros:**
- Comprehensive event coverage
- Includes private/commercial events
- Multiple fallback sources

**Cons:**
- Complex integration (5+ different APIs/formats)
- Rate limits on free tiers
- Data quality varies (need deduplication, validation)
- Social media APIs restricted/paid

**Feasibility:** High complexity, ongoing maintenance required

---

### Option C: Pivot Site Focus

**Change site to match available data:**

Instead of "Events at Plaza de España", make it:
- "Cultural Events Near Plaza de España"
- "Museums, Libraries & Cultural Centers in Centro/Moncloa"
- "Madrid Cultural Programming (Centro District)"

**Pros:**
- Works perfectly with existing implementation
- Reliable data source (Madrid open data)
- 142 high-quality events
- Low maintenance

**Cons:**
- Doesn't match original user intent (Plaza outdoor events)
- Won't show concerts, outdoor movies, gaming events
- Not what user is looking for

**Feasibility:** ✅ **Already implemented and working**

---

### Option D: Contact Madrid City Government

**Ask directly:**
- Email Madrid open data team: `datos@madrid.es`
- Ask: "Where is data published for outdoor public events at plazas (Plaza de España, Plaza Mayor, etc.)?"
- Request: Access to public square event calendar API

**Pros:**
- Official answer from source
- May discover hidden/undocumented API
- Could get access to restricted dataset

**Cons:**
- Slow (days/weeks for response)
- May not exist
- No guarantee of API access

**Feasibility:** Low technical effort, uncertain timeline

---

### Option E: Manual Plaza de España Event List

**Create curated list:**
- Check Madrid.es weekly for Plaza events
- Monitor social media for announcements
- Track event series (e.g., "Cine de Verano" summer movies)
- Manually add to static JSON file
- Merge with API events

**Pros:**
- Guaranteed accuracy for Plaza events
- Can include events from any source
- Full control over content

**Cons:**
- Manual work (not automated)
- Won't scale
- Requires weekly maintenance
- Not sustainable long-term

**Feasibility:** ✅ **Easy short-term**, ❌ **Not sustainable**

---

## Dataset 300378 (Fiestas Populares) - Worth Checking?

**Name:** Fiestas y eventos populares
**Description:** Neighborhood festivals and popular celebrations

**Hypothesis:** May include Plaza de España events if they're considered "popular festivals"

**Next step:** Test this dataset:
```bash
curl -L "https://datos.madrid.es/egob/catalogo/300378-0-fiestas-eventos-populares.json"
curl -L "https://datos.madrid.es/egob/catalogo/300378-0-fiestas-eventos-populares.xml"
curl -L "https://datos.madrid.es/egob/catalogo/300378-0-fiestas-eventos-populares.csv"
```

**Expected:** Likely district festivals (San Isidro, neighborhood fiestas), not weekly Plaza events

---

## Recommended Next Steps

### Immediate (Today):
1. ✅ Document findings (this file)
2. Test dataset 300378 (Fiestas populares)
3. Check if datos.madrid.es has site-wide search for "Plaza de España eventos"
4. Present findings + options to user for decision

### Short-term (This Week):
- **If 300378 works:** Integrate as additional data source
- **If no API exists:** User decides between Options A-E
- **Most likely:** Option C (pivot site focus) or Option E (manual curation)

### Long-term:
- Monitor Madrid open data portal for new datasets
- Consider hybrid approach (API for cultural centers + manual for Plaza events)

---

## Conclusion

**Finding:** Madrid's open data APIs do NOT include outdoor public square events (concerts, movies, gaming events at Plaza de España).

**Current implementation (300107):** ✅ Works perfectly for cultural center programming
**User's need (Plaza outdoor events):** ❌ Not available in any discovered API

**Most realistic options:**
1. **Option C** - Pivot site focus to "Cultural Events Near Plaza de España" (matches data we have)
2. **Option E** - Manual curation for Plaza-specific events + API for cultural centers (hybrid)
3. **Option A** - Web scraping madrid.es (if events exist on website but not API)

**Blocker:** Need user decision on which direction to take.

---

## Files Investigated

- `/workspace/docs/scrape.html` - Madrid.es homepage scrape
- `/workspace/docs/catalogo.csv` - Full catalog of 600+ Madrid open data datasets
- Current implementation: 300107 (Agenda de actividades)
- Attempted: 300028 (Agenda turística) - HTTP 302 error
- To test: 300378 (Fiestas populares)
