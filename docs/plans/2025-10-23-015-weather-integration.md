# Weather Integration (AEMET) Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task when executing.

**Goal:** Add AEMET weather forecasts (temperature, rain probability, sky conditions) to event cards using official Spanish meteorological data.

**Architecture:** New `internal/weather` package fetches daily forecasts from AEMET OpenData API (two-step process), matches forecast days to event dates, and enriches Event structs with weather data. Template renders weather info (AEMET official icons, temp, precip %) on cards. Graceful degradation if API fails.

**Tech Stack:** AEMET OpenData API, Go stdlib (net/http, encoding/json, time), existing fetch.HTTPCache, AEMET official PNG icons

**Date:** 2025-10-23
**Priority:** MEDIUM
**Status:** 📋 Planning

---

## Problem Statement

Events displayed on the site lack weather context. Users planning to attend outdoor events need to know:
- Will it rain?
- What's the temperature?
- What are the general sky conditions (sunny/cloudy/overcast)?

This information helps users make informed decisions about which events to attend.

---

## Solution Overview

Integrate weather forecasts from **AEMET** (Agencia Estatal de Meteorología - Spanish State Meteorological Agency) into event cards.

**Data source:** AEMET OpenData API
- **Endpoint:** `https://opendata.aemet.es/opendata/api/prediccion/especifica/municipio/diaria/28079`
- **Municipality:** Madrid (ID: 28079)
- **Authentication:** Free API key (indefinite validity since 2017 policy change)
- **Forecast range:** Up to 7 days daily forecast
- **Rate limits:** Respectful usage (hourly updates acceptable)
- **Weather icons:** Publicly accessible PNGs at `https://www.aemet.es/imagenes/png/estado_cielo/{code}.png`
- **License:** Open data per Law 18/2015 (Spain), requires attribution to AEMET

**Display approach:**
- Add weather icons/indicators to event cards
- Show high temperature for the day
- Show precipitation probability
- Show sky state (sun/clouds/rain icons)
- CSS-only presentation (no JavaScript, consistent with site architecture)

---

## AEMET API Details

### Registration Process

1. Visit https://opendata.aemet.es/centrodedescargas/altaUsuario
2. Register with email address
3. Receive API key by email
4. **Key validity: Indefinite** (since September 2017, AEMET no longer uses time-limited keys)
5. Store in config or environment variable

**Note:** Prior to September 13, 2017, AEMET used time-limited API keys, but they changed their policy to promote mass reuse of meteorological data. Modern API keys do not expire, though you should still handle potential key revocation or account issues gracefully.

### API Request Flow (Two-Step Process)

AEMET uses an indirect data access pattern:

**Step 1:** Request forecast endpoint with API key
```bash
GET https://opendata.aemet.es/opendata/api/prediccion/especifica/municipio/diaria/28079
Header: api_key: YOUR_API_KEY
```

**Response 1:** Metadata with data URL
```json
{
  "descripcion": "exito",
  "estado": 200,
  "datos": "https://opendata.aemet.es/opendata/sh/abc123...",
  "metadatos": "https://opendata.aemet.es/opendata/sh/meta456..."
}
```

**Step 2:** Fetch actual forecast data from `datos` URL
```bash
GET https://opendata.aemet.es/opendata/sh/abc123...
```

**Response 2:** Actual forecast JSON (see data structure below)

### Forecast Data Structure

```json
[
  {
    "origen": {
      "productor": "Agencia Estatal de Meteorología - AEMET",
      "web": "https://www.aemet.es",
      "enlace": "...",
      "language": "es",
      "copyright": "© AEMET. Autorizado el uso de la información...",
      "notaLegal": "..."
    },
    "elaborado": "2025-10-23T12:34:56",
    "nombre": "Madrid",
    "provincia": "Madrid",
    "prediccion": {
      "dia": [
        {
          "fecha": "2025-10-23T00:00:00",
          "orto": "08:15",
          "ocaso": "19:30",
          "temperatura": {
            "maxima": 24,
            "minima": 12,
            "dato": [
              {"value": 14, "hora": 6},
              {"value": 18, "hora": 12},
              {"value": 22, "hora": 15},
              ...
            ]
          },
          "estadoCielo": [
            {"value": "11", "periodo": "00-06", "descripcion": "Despejado"},
            {"value": "14", "periodo": "06-12", "descripcion": "Nuboso"},
            {"value": "15", "periodo": "12-18", "descripcion": "Muy nuboso"},
            {"value": "16", "periodo": "18-24", "descripcion": "Cubierto"}
          ],
          "precipitacion": [
            {"value": "0", "periodo": "00-06"},
            {"value": "2", "periodo": "06-12"},
            {"value": "5", "periodo": "12-18"},
            {"value": "1", "periodo": "18-24"}
          ],
          "probPrecipitacion": [
            {"value": 10, "periodo": "00-12"},
            {"value": 65, "periodo": "12-24"}
          ],
          "viento": [...],
          "rachaMax": [...],
          "humedadRelativa": {...},
          "sensTermica": {...},
          "uvMax": 6
        },
        // ... more days (up to 7 days)
      ]
    }
  }
]
```

### Sky State Codes (estadoCielo.value)

Based on AEMET documentation:
- **11-13**: Clear/Despejado (day/night variations)
- **14**: Few clouds/Poco nuboso
- **15**: Partly cloudy/Intervalos nubosos
- **16**: Very cloudy/Muy nuboso
- **17**: Overcast/Cubierto
- **23-27**: Rain variations (light/moderate/heavy)
- **43-46**: Snow variations
- **51-53**: Storm variations

Codes ending in 'n' (e.g., "11n") indicate nighttime conditions.

### Key Fields to Extract

For each event date:
1. **Temperature max** (`temperatura.maxima`): High temp for the day
2. **Precipitation probability** (`probPrecipitacion`): % chance of rain (use highest period covering event time)
3. **Sky state** (`estadoCielo`): Icon representation (use period covering event time)
4. **Precipitation amount** (`precipitacion`): Total mm (use period covering event time)

---

## Licensing and Attribution

### AEMET Open Data License

AEMET data and weather icons are available under Spain's **Law 18/2015** on reuse of public sector information.

**Permitted uses:**
- ✅ Commercial and non-commercial use
- ✅ Redistribution and modification
- ✅ Integration into value-added services

**Attribution requirements:**

1. **Mandatory citation:** Must cite AEMET as author of the data
   - Use: `© AEMET` or
   - Use: "Información elaborada por la Agencia Estatal de Meteorología" or
   - Use: "Fuente: AEMET" (for value-added services)

2. **Logo retention:** If AEMET logo appears in original data, it must be retained (not applicable to weather icons)

3. **No endorsement:** Cannot declare, insinuate, or suggest that AEMET participates in, sponsors, or supports your reuse

**Weather icons specifically:**
- Icons at `https://www.aemet.es/imagenes/png/estado_cielo/*.png` are covered under the same open data license
- File size: ~1.3KB per PNG
- Cacheable: max-age=3600 (1 hour)
- No authentication required for icon access

**Implementation:**
- Add AEMET attribution to site footer (already includes data sources)
- Update ATTRIBUTION.md with AEMET copyright and license details
- Include AEMET logo/link in build report weather section
- Ensure weather icons load from AEMET's CDN (no local copies initially)

---

## Implementation Architecture

### New Package: `internal/weather`

```
internal/weather/
  client.go          # AEMET API client (two-step fetch)
  types.go           # Forecast, DayForecast, Weather structs
  matcher.go         # Match event dates to forecast days
  icons.go           # Map sky codes to CSS classes/display info
  cache.go           # Cache AEMET responses (reuse existing fetch.HTTPCache)
```

### Data Flow

```
Pipeline (existing)
    ↓
Fetch Events (existing)
    ↓
Filter Events (existing)
    ↓
[NEW] Fetch Weather ← AEMET API
    ↓
[NEW] Match Weather to Events ← Join on date
    ↓
Render with Weather (modified)
    ↓
Static HTML/CSS
```

### Configuration Changes

**config.toml additions:**
```toml
[weather]
enabled = true
api_key_env = "AEMET_API_KEY"  # Read from env var for security
municipality_code = "28079"    # Madrid
cache_ttl_hours = 6           # Cache forecast for 6 hours
```

**Command-line flag additions:**
```bash
-weather-api-key string     # AEMET API key (or use AEMET_API_KEY env var)
-weather-enabled            # Enable weather integration (default: false initially)
```

### Event Struct Extension

**internal/event/event.go modifications:**
```go
type Event struct {
    // ... existing fields ...

    // Weather info (populated during pipeline)
    Weather *Weather `json:"weather,omitempty"`
}

type Weather struct {
    Date              string  `json:"date"`               // Forecast date (YYYY-MM-DD)
    TempMax           int     `json:"temp_max"`           // Max temp (°C)
    TempMin           int     `json:"temp_min"`           // Min temp (°C)
    PrecipProb        int     `json:"precip_prob"`        // Precipitation probability (%)
    PrecipAmount      float64 `json:"precip_amount"`      // Total precipitation (mm)
    SkyCode           string  `json:"sky_code"`           // AEMET sky state code (e.g., "12", "15n")
    SkyDescription    string  `json:"sky_description"`    // Human-readable sky state (Spanish)
    SkyIconURL        string  `json:"sky_icon_url"`       // AEMET official icon URL
    WeatherCategory   string  `json:"weather_category"`   // Simplified category for CSS (clear/cloudy/rain/etc)
    IsNight           bool    `json:"is_night"`           // True if code ends with 'n'
}
```

### Weather Client Implementation

**internal/weather/client.go:**
```go
type Client struct {
    apiKey       string
    httpCache    *fetch.HTTPCache
    municipalityCode string
}

func (c *Client) FetchForecast() (*Forecast, error) {
    // Step 1: Request forecast endpoint
    metaURL := fmt.Sprintf("https://opendata.aemet.es/opendata/api/prediccion/especifica/municipio/diaria/%s", c.municipalityCode)
    metaResp := c.fetchWithAPIKey(metaURL)

    // Step 2: Extract datos URL
    datosURL := parseMetaResponse(metaResp)

    // Step 3: Fetch actual forecast data
    forecastData := c.httpCache.Fetch(datosURL)

    return parseForecast(forecastData)
}
```

### Weather Matching Logic

**internal/weather/matcher.go:**
```go
func MatchEventsToWeather(events []event.Event, forecast *Forecast) []event.Event {
    // Build date -> forecast map
    forecastMap := buildDateMap(forecast)

    // For each event:
    for i := range events {
        eventDate := extractDate(events[i].Fecha)
        if dayForecast, ok := forecastMap[eventDate]; ok {
            events[i].Weather = buildWeatherFromForecast(dayForecast, events[i])
        }
    }

    return events
}

func buildWeatherFromForecast(day *DayForecast, evt event.Event) *event.Weather {
    // Determine event time period (morning/afternoon/evening)
    period := determineEventPeriod(evt)

    // Extract sky state for that period
    skyState := extractSkyForPeriod(day.EstadoCielo, period)

    // Extract precipitation probability for that period
    precipProb := extractPrecipProbForPeriod(day.ProbPrecipitacion, period)

    return &event.Weather{
        Date:            day.Fecha,
        TempMax:         day.Temperatura.Maxima,
        TempMin:         day.Temperatura.Minima,
        PrecipProb:      precipProb,
        SkyCode:         skyState.Value,
        SkyDescription:  skyState.Descripcion,
        SkyIconURL:      GetAEMETIconURL(skyState.Value),
        WeatherCategory: GetWeatherCategory(skyState.Value),
        IsNight:         IsNightCondition(skyState.Value),
    }
}
```

### Icon Mapping

**internal/weather/icons.go:**

AEMET provides official weather icons as PNGs at `https://www.aemet.es/imagenes/png/estado_cielo/{code}.png`. These icons are publicly accessible and cacheable (Cache-Control: max-age=3600).

```go
// GetAEMETIconURL returns the official AEMET icon URL for a sky state code
func GetAEMETIconURL(code string) string {
    // AEMET icons use numeric codes: 11, 12, 13, 14, etc.
    // Some codes have 'n' suffix for night (e.g., "11n")
    // The icon files use just the base code (11.png works for both 11 and 11n)
    baseCode := strings.TrimSuffix(code, "n")
    return fmt.Sprintf("https://www.aemet.es/imagenes/png/estado_cielo/%s.png", baseCode)
}

// IsNightCondition checks if the code represents a night condition
func IsNightCondition(code string) bool {
    return strings.HasSuffix(code, "n")
}

// GetWeatherCategory returns a simplified category for CSS styling
func GetWeatherCategory(code string) string {
    baseCode := strings.TrimSuffix(code, "n")

    switch {
    case baseCode >= "11" && baseCode <= "13":
        return "clear"      // Clear/Despejado
    case baseCode == "14" || baseCode == "15":
        return "partial"    // Few clouds/Partly cloudy
    case baseCode == "16" || baseCode == "17":
        return "cloudy"     // Very cloudy/Overcast
    case baseCode >= "23" && baseCode <= "27":
        return "rain"       // Rain
    case baseCode >= "43" && baseCode <= "46":
        return "snow"       // Snow
    case baseCode >= "51" && baseCode <= "53":
        return "storm"      // Storm
    default:
        return "unknown"
    }
}
```

**Benefits of using AEMET's official icons:**
- ✅ Authoritative source (official Spanish meteorology agency)
- ✅ Publicly accessible (no authentication needed)
- ✅ Cacheable (1-hour cache control headers)
- ✅ Comprehensive coverage (all sky state codes have corresponding icons)
- ✅ Consistent with AEMET's own weather displays
- ✅ No licensing concerns (public government data)
- ✅ Small file size (~1.3KB per PNG)

---

## Template Changes

### Event Card Weather Display

**generator/templates/index.tmpl.html modifications:**

```html
<article class="event-card {{.EventType}}" id="ev-{{.IDEvento}}"
         data-distance-m="{{.DistanceMeters}}"
         {{if .AtPlaza}}data-at-plaza="true"{{end}}>
  {{if eq .EventType "city"}}
  <span class="event-badge city-badge">Evento Ciudad</span>
  {{else}}
  <span class="event-badge cultural-badge">Cultural</span>
  {{end}}

  <!-- NEW: Weather indicator -->
  {{if .Weather}}
  <div class="weather-info weather-{{.Weather.WeatherCategory}}" title="Pronóstico: {{.Weather.SkyDescription}}">
    <img src="{{.Weather.SkyIconURL}}"
         alt="{{.Weather.SkyDescription}}"
         class="weather-icon"
         width="20"
         height="20"
         loading="lazy">
    <span class="weather-temp">{{.Weather.TempMax}}°</span>
    {{if gt .Weather.PrecipProb 30}}
    <span class="weather-precip">💧{{.Weather.PrecipProb}}%</span>
    {{end}}
  </div>
  {{end}}

  <h3>{{.Titulo}}</h3>
  <p class="when">{{.StartHuman}}</p>
  {{if .NombreInstalacion}}<p class="where">{{.NombreInstalacion}}</p>{{end}}
  {{if .DistanceHuman}}<p class="distance"><!-- SVG icon -->{{.DistanceHuman}} de Plaza de España</p>{{end}}
  {{if .Description}}<p class="description">{{.Description}}</p>{{end}}
  {{if .ContentURL}}<p><a href="{{.ContentURL}}">Más información</a></p>{{end}}
</article>
```

### CSS Styling

**generator/assets/site.css additions:**

```css
/* Weather info container */
.weather-info {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  margin-bottom: 0.5rem;
  font-size: 0.9rem;
  color: #666;
}

/* Weather icon from AEMET */
.weather-icon {
  width: 20px;
  height: 20px;
  flex-shrink: 0;
  /* AEMET icons are transparent PNGs, work on light/dark backgrounds */
}

/* Temperature display */
.weather-temp {
  font-weight: 600;
  color: #ea580c;  /* Match site accent color */
}

/* Precipitation probability (only shown if >30%) */
.weather-precip {
  color: #3b82f6;  /* Blue for rain */
  font-size: 0.85rem;
}

/* Optional: Category-based styling for weather container */
.weather-info.weather-rain {
  /* Could add subtle rain-colored border or background */
}

.weather-info.weather-clear {
  /* Could add subtle sunny styling */
}

/* Dark mode support (if site adds dark mode later) */
@media (prefers-color-scheme: dark) {
  .weather-icon {
    /* AEMET icons may need slight opacity adjustment for dark backgrounds */
    opacity: 0.9;
  }
}
```

**Notes on AEMET icon integration:**
- Icons load from `https://www.aemet.es/imagenes/png/estado_cielo/{code}.png`
- Browser caches automatically per AEMET's Cache-Control headers (1 hour)
- `loading="lazy"` defers loading until icon is near viewport
- `width` and `height` prevent layout shift during load
- Icons are ~1.3KB each, minimal bandwidth impact

---

## Build Report Integration

Add weather fetch stats to build report:

**internal/report/types.go additions:**
```go
type BuildReport struct {
    // ... existing fields ...

    WeatherReport *WeatherReport `json:"weather_report,omitempty"`
}

type WeatherReport struct {
    Enabled         bool      `json:"enabled"`
    FetchTimestamp  time.Time `json:"fetch_timestamp"`
    Municipality    string    `json:"municipality"`
    DaysCovered     int       `json:"days_covered"`
    EventsMatched   int       `json:"events_matched"`
    EventsUnmatched int       `json:"events_unmatched"`
    CacheHit        bool      `json:"cache_hit"`
    Errors          []string  `json:"errors,omitempty"`
}
```

**Build report display:**
```html
<section class="report-section weather-section">
  <h2>Weather Integration</h2>
  {{if .WeatherReport.Enabled}}
    <div class="metric">
      <span class="label">Municipality:</span>
      <span class="value">{{.WeatherReport.Municipality}}</span>
    </div>
    <div class="metric">
      <span class="label">Forecast days:</span>
      <span class="value">{{.WeatherReport.DaysCovered}}</span>
    </div>
    <div class="metric">
      <span class="label">Events matched:</span>
      <span class="value">{{.WeatherReport.EventsMatched}} / {{add .WeatherReport.EventsMatched .WeatherReport.EventsUnmatched}}</span>
    </div>
    <div class="metric">
      <span class="label">Cache hit:</span>
      <span class="value">{{if .WeatherReport.CacheHit}}✓{{else}}✗{{end}}</span>
    </div>
  {{else}}
    <p>Weather integration disabled</p>
  {{end}}
</section>
```

---

## Respectful Fetching Strategy

### Caching Approach

Reuse existing `internal/fetch` infrastructure:

1. **HTTP Cache:** Store AEMET responses in `data/http-cache/`
2. **TTL:** 6 hours (weather doesn't change frequently)
3. **Request Audit:** Track AEMET API calls in `data/request-audit.json`
4. **Development Mode:** Aggressive caching (avoid hitting API during testing)
5. **Production Mode:** Normal caching (fresh data every 6 hours)

### Request Throttling

- **No burst requests:** Single AEMET fetch per build (2-step process counted as 1 logical request)
- **Delay after event fetch:** Add 2-second delay before weather fetch
- **Cache-first:** Check cache before making any API call
- **Graceful degradation:** If AEMET fails, events still render (without weather)

### API Key Management

**Security considerations:**
- Never commit API key to repo
- Read from environment variable: `AEMET_API_KEY`
- Document registration process in README
- Include fallback: if no key provided, disable weather silently

**Deployment:**
- Add env var to NFSN scheduled task
- Document in `docs/deployment.md`
- CI/CD: Use GitHub Secrets for preview deployments

---

## Error Handling

### Failure Modes

1. **No API key provided:** Disable weather silently, log warning
2. **API key expired:** Log error, disable weather, send notification (via build report)
3. **AEMET API down:** Use cached forecast if available, otherwise disable weather
4. **Invalid response:** Log error, disable weather, continue rendering events
5. **Network timeout:** Use cached forecast, log warning
6. **Rate limited (429):** Use cached forecast, log warning, respect Retry-After header

### Graceful Degradation

**Principle:** Weather is enhancement, not requirement. Site must work without it.

```go
func (p *Pipeline) enrichWithWeather(events []event.Event) []event.Event {
    if !p.weatherEnabled {
        return events
    }

    forecast, err := p.weatherClient.FetchForecast()
    if err != nil {
        p.logger.Warnf("Weather fetch failed: %v (continuing without weather)", err)
        return events  // Return events unchanged
    }

    return weather.MatchEventsToWeather(events, forecast)
}
```

---

## Testing Strategy

**CRITICAL PRINCIPLE:** Never hit real APIs during automated testing. Always use fixtures.

### Test Fixtures

All tests use fixtures from `generator/testdata/fixtures/`:
- `aemet-madrid-metadata.json` - AEMET two-step metadata response
- `aemet-madrid-forecast.json` - AEMET forecast data (real structure, frozen in time)

**Fixture philosophy:**
- Committed to repo (versioned, reproducible)
- Refreshed manually when needed (not during CI)
- Represents real API responses (structure validation)
- Enables offline development

### Unit Tests (Use Fixtures)

1. **weather/client_test.go:**
   - Use `httptest.NewServer()` serving fixture data
   - Mock two-step AEMET flow with fixture files
   - Test API key header injection (verify in mock server)
   - Test error handling (404, 500, timeout) with mock responses
   - Test cache hit behavior
   - **NO REAL API CALLS** - mock server only

2. **weather/matcher_test.go:**
   - Load `aemet-madrid-forecast.json` fixture
   - Test date matching (exact match, no match, multi-day events)
   - Test period extraction (morning/afternoon/evening)
   - Test sky code mapping (use codes from fixture)
   - Test precipitation probability extraction
   - **NO API DEPENDENCY** - pure data transformation tests

3. **weather/icons_test.go:**
   - Test sky code to icon URL mapping
   - Test night code handling ('n' suffix)
   - Test unknown code fallback
   - Test GetWeatherCategory() function
   - **NO API CALLS** - pure function tests

### Integration Tests (Use Fixtures)

1. **pipeline_test.go modifications:**
   - Mock weather client returning fixture data
   - Test pipeline with weather enabled (fixture-based)
   - Test pipeline with weather disabled
   - Test pipeline with weather fetch failure (mock error)
   - **NO REAL API CALLS** - dependency injection with mocks

### Manual Testing

1. **Development mode:**
   - Register for AEMET API key
   - Run `just dev` with weather enabled
   - Verify weather appears on event cards
   - Verify caching works (second run instant)

2. **Production simulation:**
   - Run with production fetch mode
   - Verify respectful delays
   - Check request audit trail

---

## Implementation Tasks

### Phase 0: Test Fixtures (30 min) - **DO THIS FIRST**

**Critical:** Fetch real AEMET data early to use during development/testing. Never hit real APIs during automated testing.

- [ ] **Task 0.1:** Update `scripts/fetch-fixtures.sh` to fetch AEMET weather data
  - Add AEMET two-step fetch (metadata → datos URL → forecast JSON)
  - Save both metadata and forecast data to `generator/testdata/fixtures/`
  - Handle missing `AEMET_API_KEY` gracefully (skip with warning)
  - Files created:
    - `generator/testdata/fixtures/aemet-madrid-metadata.json` (two-step metadata response)
    - `generator/testdata/fixtures/aemet-madrid-forecast.json` (actual forecast data)

- [ ] **Task 0.2:** Fetch fixture with real API key
  ```bash
  export AEMET_API_KEY="your-key-here"
  ./scripts/fetch-fixtures.sh
  ```

- [ ] **Task 0.3:** Commit fixtures to repo
  - Weather data is date-specific but structure is stable
  - Refresh fixtures periodically (monthly or when AEMET API changes)
  - Include fixture date in filename or metadata
  - Add comment in fixture JSON noting it's test data

- [ ] **Task 0.4:** Update `.envrc.local.example` to include AEMET_API_KEY
  ```bash
  # AEMET OpenData API key (for weather forecasts)
  # Register at: https://opendata.aemet.es/centrodedescargas/altaUsuario
  # Required for: weather integration, fixture fetching
  export AEMET_API_KEY=your_aemet_api_key_here
  ```

- [ ] **Task 0.5:** Add justfile command for fetching fixtures
  ```just
  # Fetch test fixtures from upstream APIs
  fetch-fixtures:
      @echo "📥 Fetching test fixtures..."
      @./scripts/fetch-fixtures.sh
      @echo "✅ Fixtures updated in generator/testdata/fixtures/"
  ```

- [ ] **Task 0.6:** Update deployment documentation (docs/deployment.md)
  - Add section for AEMET API key setup on NFSN
  - Add env var to cron command example: `AEMET_API_KEY=...`
  - Document how to update key if changed

- [ ] **Task 0.7:** Update CI configuration (.github/workflows/ci.yml)
  - **Do NOT add API key to CI** (fixtures only)
  - Add comment explaining why: "Weather tests use fixtures, no API key needed"
  - Consider adding workflow to refresh fixtures monthly (manual trigger)

- [ ] **Task 0.8:** Document fixture usage in CLAUDE.md
  - How to refresh fixtures: `just fetch-fixtures`
  - When to refresh (AEMET API changes, or monthly)
  - How tests use fixtures (no real API calls)
  - Note that fixtures are committed to repo

**Why do this first:**
- Tests need real AEMET response structure
- Avoids hitting API during development iterations
- Provides concrete examples of AEMET data format
- Enables TDD without API dependency
- Sets up all configuration early (API keys, env vars, docs)

---

### Phase 1: Infrastructure (2-3 hours)

- [ ] **Task 1.1:** Create `internal/weather` package structure
- [ ] **Task 1.2:** Implement AEMET client with two-step fetch
- [ ] **Task 1.3:** Define weather types (Forecast, DayForecast, Weather)
- [ ] **Task 1.4:** Integrate with existing HTTPCache system
- [ ] **Task 1.5:** Add configuration (config.toml + flags)
- [ ] **Task 1.6:** Add API key environment variable support

### Phase 2: Data Processing (2 hours)

- [ ] **Task 2.1:** Implement date matcher (events → forecast days)
- [ ] **Task 2.2:** Implement period extractor (event time → sky state period)
- [ ] **Task 2.3:** Implement sky code to icon mapper
- [ ] **Task 2.4:** Implement precipitation probability extractor
- [ ] **Task 2.5:** Extend Event struct with Weather field

### Phase 3: Pipeline Integration (1 hour)

- [ ] **Task 3.1:** Add weather fetch step to pipeline
- [ ] **Task 3.2:** Add weather matching step
- [ ] **Task 3.3:** Add error handling with graceful degradation
- [ ] **Task 3.4:** Add 2-second delay before weather fetch

### Phase 4: Presentation (1.5 hours)

- [ ] **Task 4.1:** Update HTML template with weather display (img tags for AEMET icons)
- [ ] **Task 4.2:** Add CSS styles for weather info container
- [ ] **Task 4.3:** Test AEMET icon loading and browser caching
- [ ] **Task 4.4:** Test responsive layout with weather info

### Phase 5: Reporting (1 hour)

- [ ] **Task 5.1:** Add WeatherReport to BuildReport
- [ ] **Task 5.2:** Populate weather stats in pipeline
- [ ] **Task 5.3:** Add weather section to build report HTML
- [ ] **Task 5.4:** Include AEMET attribution in footer

### Phase 6: Testing (2 hours)

- [ ] **Task 6.1:** Write unit tests for weather client
- [ ] **Task 6.2:** Write unit tests for matcher
- [ ] **Task 6.3:** Write unit tests for icon mapper
- [ ] **Task 6.4:** Update integration tests
- [ ] **Task 6.5:** Manual testing with real API

### Phase 7: Documentation (1 hour)

- [ ] **Task 7.1:** Update CLAUDE.md with weather integration details
- [ ] **Task 7.2:** Update README with weather feature
- [ ] **Task 7.3:** Update docs/deployment.md with API key setup
- [ ] **Task 7.4:** Add AEMET attribution to ATTRIBUTION.md

### Phase 8: Deployment (30 min)

- [ ] **Task 8.1:** Add AEMET_API_KEY to NFSN environment
- [ ] **Task 8.2:** Update cron command with `-weather-enabled` flag
- [ ] **Task 8.3:** Test on production
- [ ] **Task 8.4:** Monitor for errors in first 24 hours

**Total estimated time:** 12-13 hours (including 1 hour for Phase 0 setup)

---

## Success Criteria

✅ Weather info displays on event cards (icon + temp + precip %)
✅ Weather data fetched from AEMET API with two-step process
✅ API key managed via environment variable (not in repo)
✅ Caching works (6-hour TTL, cache hits logged)
✅ Respectful fetching (2s delay, request audit trail)
✅ Graceful degradation (site works if AEMET fails)
✅ Build report shows weather fetch stats
✅ All tests pass (22 existing + 8 new weather tests)
✅ No JavaScript added (CSS-only presentation)
✅ Mobile responsive (weather info fits on small screens)
✅ AEMET attribution included
✅ Documentation complete (README, deployment docs)

---

## Alternative Approaches Considered

### Alternative 1: Use Open-Meteo (Third-party)

**Pros:**
- No API key required
- Better JSON API design
- More developer-friendly

**Cons:**
- Not the official Spanish source
- Adds external dependency outside Spain
- Less accurate for Madrid specifically

**Decision:** Use AEMET (official Spanish source, more appropriate)

### Alternative 2: Fetch Hourly Forecast

**Pros:**
- More precise matching to event times
- Better accuracy for morning vs evening events

**Cons:**
- More complex data structure
- More data to cache
- Overkill for most events (all-day or evening)

**Decision:** Use daily forecast (simpler, sufficient for most events)

### Alternative 3: Use Custom Icons Instead of AEMET PNGs

**Pros:**
- Full control over icon design
- Could match site branding better
- Could use SVG for infinite scalability
- No external dependency on AEMET CDN

**Cons:**
- Licensing complexity (need to find/create licensed icons)
- Not authoritative (AEMET icons are "official" Spanish weather icons)
- Maintenance burden (need to update if AEMET adds new codes)
- Larger payload if embedding SVGs in HTML
- Loss of consistency with AEMET's own weather displays

**Decision:** Use AEMET's official PNG icons
- Authoritative and consistent with source
- Publicly accessible with clear licensing
- Small file size (~1.3KB each)
- Browser-cacheable (1-hour max-age)
- No licensing concerns
- Could revisit later if AEMET CDN proves unreliable

---

## Security & Privacy Considerations

### API Key Security

- ✅ Never commit API key to repo
- ✅ Use environment variable
- ✅ Document rotation process (3-month expiry)
- ✅ Log API errors (but not API key)

### Data Privacy

- ✅ Weather data is public (no user data involved)
- ✅ No tracking added
- ✅ No cookies required
- ✅ AEMET attribution included

### Rate Limiting

- ✅ Respect AEMET rate limits
- ✅ Cache aggressively (6-hour TTL)
- ✅ Handle 429 gracefully
- ✅ Monitor request audit trail

---

## Monitoring & Maintenance

### What to Monitor

1. **API key expiry:** Set calendar reminder for 2.5 months
2. **AEMET API errors:** Check build report daily for first week
3. **Cache hit rate:** Should be >90% after first fetch
4. **Weather match rate:** Should be >80% (events within 7-day forecast window)

### Maintenance Tasks

- **Every 3 months:** Renew AEMET API key
- **Weekly:** Check build report for weather errors
- **Monthly:** Verify AEMET API still uses same endpoints
- **Yearly:** Review sky code mappings for accuracy

---

## Future Enhancements

### Phase 2 Features (Not in Initial Implementation)

1. **Weather-based filtering:**
   - CSS-only filter to hide rainy events (checkbox + CSS)
   - Show only sunny outdoor events

2. **Multi-day event weather:**
   - Show weather for entire event range
   - Aggregate forecast (e.g., "Mostly sunny, 18-24°")

3. **Weather alerts:**
   - Highlight severe weather warnings
   - Pull from AEMET alerts API

4. **Historical weather:**
   - For past events, show actual weather (if cached)
   - Compare forecast vs actual

5. **Hourly forecast precision:**
   - Match specific event times to hourly forecast
   - Show "Rain expected at 7pm" for evening events

---

## Questions & Decisions

### Q1: What if event is >7 days in future?

**Decision:** No weather shown (AEMET only provides 7-day forecast)

### Q2: What if AEMET changes API?

**Decision:**
- Monitor API version in response
- Log warning if version changes
- Graceful degradation (disable weather)
- Add integration test with real API call (manual)

### Q3: Multi-day events?

**Decision:**
- Initial: Show weather for start date only
- Future: Aggregate weather across all days

### Q4: Indoor vs outdoor events?

**Decision:**
- No differentiation initially (data doesn't indicate indoor/outdoor)
- Future: Add manual tagging or ML classification

### Q5: Night events (after sunset)?

**Decision:**
- Use night sky codes ('n' suffix)
- Show evening period forecast (18-24)
- Display moon icon for clear nights

---

## Dependencies

### External Services

- **AEMET OpenData API:** https://opendata.aemet.es/
- **API key registration:** Free, renewable every 3 months

### Go Packages (Standard Library Only)

- `encoding/json`: Parse AEMET responses
- `net/http`: HTTP requests
- `time`: Date matching
- Reuse existing `internal/fetch` infrastructure

### Configuration

- Environment variable: `AEMET_API_KEY`
- Config file: `config.toml` (weather section)
- Command flags: `-weather-enabled`, `-weather-api-key`

---

## Rollout Plan

### Development Phase (Week 1)

1. Implement infrastructure (Phase 1-2)
2. Register for AEMET API key
3. Local testing with real API
4. Iterate on icon design

### Testing Phase (Week 2)

1. Implement presentation + reporting (Phase 3-5)
2. Write tests (Phase 6)
3. Manual testing with edge cases
4. Documentation (Phase 7)

### Deployment Phase (Week 3)

1. Deploy to preview environment
2. Monitor for 48 hours
3. Deploy to production
4. Monitor for 1 week

### Stabilization (Week 4)

1. Fix any issues discovered
2. Optimize caching strategy
3. Refine icon mappings based on user feedback
4. Document lessons learned

---

## Rollback Plan

If weather integration causes issues:

1. **Quick rollback:** Set `-weather-enabled=false` in cron
2. **Code rollback:** Revert commits, redeploy
3. **Partial rollback:** Keep code, disable in config
4. **Data cleanup:** Clear weather cache if corrupted

**No data loss risk:** Weather is additive feature, doesn't modify existing event data.

---

## API Key Management

✅ **GOOD NEWS:** AEMET API keys have **indefinite validity** (since September 2017 policy change).

**No automatic expiration**, but you should still handle potential issues:

**Possible key issues:**
- Account closure (if email bounces or user deactivates)
- Policy changes (unlikely, but monitor AEMET announcements)
- Rate limit violations (if we exceed limits, key could be revoked)

**If key stops working:**
1. Check AEMET OpenData announcements for policy changes
2. Try registering new API key at https://opendata.aemet.es/centrodedescargas/altaUsuario
3. Update `AEMET_API_KEY` environment variable on server
4. Test with `just dev` locally first
5. Deploy to production

**Monitoring:**
- Build report will show AEMET API errors
- Set up alert if weather fetch fails >24 hours
- No routine renewal needed (unlike old 3-month policy)
