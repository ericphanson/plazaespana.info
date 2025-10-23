# PR: Add CSS-only distance slider filter for events

**Title**: `feat: Add CSS-only distance slider filter for events`

**Branch**: `claude/plaza-distance-slider-011CUPqk63SQfJdnrFRgfGm7`

---

## 🎯 Summary

Adds a user-configurable distance filter with 5 preset distances (En Plaza, 250m, 500m, 750m, 1km). Implemented entirely with CSS (no JavaScript), compatible with strict Content Security Policy.

## ✨ Features

### User Interface
- **5 distance options**: En Plaza (0m), 250m, 500m, 750m, 1km (all)
- **Visual feedback**: Selected button highlighted with accent color
- **Responsive design**: Buttons wrap on mobile devices
- **Accessible**: Keyboard navigable, screen reader friendly

### Technical Implementation
- **Pure CSS filtering**: Uses `:checked` pseudo-selectors and sibling combinators
- **No JavaScript**: Works with `script-src 'none'` CSP
- **Smart fallbacks**: Text matching for events without GPS coordinates
- **Accent-insensitive**: Matches "Plaza de España", "Plaza de Espana", "Pl. España", etc.

## 🔧 Changes

### Backend
1. **Text Matching** (`internal/filter/geo.go`):
   - `IsAtPlazaEspana()`: Detects Plaza de España by venue name
   - Case-insensitive, accent-insensitive matching
   - Handles abbreviations: "Pl.", "Plza.", etc.

2. **Distance Bucketing** (`internal/filter/geo.go`):
   - `GetDistanceBucket()`: Categorizes distances into 5 buckets
   - Buckets: "0-250", "251-500", "501-750", "751-1000", "1000+"

3. **Event Metadata** (`internal/render/types.go`):
   - `DistanceMeters int`: Distance in meters (0-1000+)
   - `DistanceBucket string`: Bucket for CSS filtering
   - `AtPlaza bool`: Flag for "En Plaza" filter

4. **Rendering Pipeline** (`internal/render/grouping.go`):
   - Calculates distance for events with GPS coordinates
   - Falls back to text matching for events without coordinates
   - Populates all distance metadata fields

5. **Configuration** (`config.toml`):
   - Increased `radius_km` from 0.35km to 1.0km
   - Supports full range of distance slider

### Frontend
1. **HTML** (`templates/index-grouped.tmpl.html`):
   - 5 hidden radio buttons with `name="distance-filter"`
   - Styled labels as clickable buttons
   - Event cards have `data-distance-m`, `data-distance-bucket`, `data-at-plaza` attributes

2. **CSS** (`assets/site.css`):
   - Distance filter styling (button grid, hover states, active states)
   - Filtering rules using `:checked ~ main .event-card` selectors
   - Hides events outside selected threshold with `display: none`

## 🧪 Testing

### Unit Tests (50+ test cases)

**Text Matching** (`internal/filter/geo_plaza_test.go`):
- ✅ 28 test cases for `IsAtPlazaEspana()`
  - Accents: "España", "Espana", "ESPAÑA"
  - Abbreviations: "Pl. España", "Plza España"
  - Partial matches: "Evento en Plaza de España, Madrid"
  - Negative cases: "Plaza Mayor", "Jardines de España"

**Distance Bucketing** (`internal/filter/geo_plaza_test.go`):
- ✅ 15 test cases for `GetDistanceBucket()`
  - Boundary conditions: 0m, 250m, 500m, 750m, 1000m
  - All 5 buckets covered
  - Edge cases: 1001m, 10000m

**Text Normalization** (`internal/filter/geo_plaza_test.go`):
- ✅ 7 test cases for `normalizeText()`
  - Spanish accents: "España" → "espana"
  - French accents: "Café français" → "cafe francais"
  - German umlauts: "Müller über" → "muller uber"

### Manual Testing

To test locally:
```bash
git checkout claude/plaza-distance-slider-011CUPqk63SQfJdnrFRgfGm7
just dev
# Open http://localhost:8080
# Click each distance filter button
# Verify events filter correctly
```

## 📸 Screenshots

> **Note**: Screenshots require local testing with a browser. See `docs/screenshot-instructions.md` for how to capture them.
>
> To test this PR:
> 1. `git checkout claude/plaza-distance-slider-011CUPqk63SQfJdnrFRgfGm7`
> 2. `just dev`
> 3. Open http://localhost:8080
> 4. Click distance filter buttons to see filtering in action

### Expected Behavior

**Default (1km - all events)**:
- All distance buttons visible in header
- "1km (todos)" button highlighted
- All events within 1km shown

**250m filter**:
- "250m" button highlighted
- Only events ≤250m from Plaza de España shown
- Events >250m hidden

**En Plaza (0m) filter**:
- "En Plaza" button highlighted
- Only events with `data-at-plaza="true"` shown
- Events matched by venue name text (e.g., "Plaza de España")

## 📚 Documentation

- ✅ Comprehensive feature documentation: `docs/features/distance-slider.md`
- ✅ Screenshot instructions: `docs/screenshot-instructions.md`
- ✅ Implementation details, testing strategy, accessibility notes

## 🔄 Compatibility

- **Browsers**: All modern browsers (IE9+)
- **JavaScript**: Not required (works with JS disabled)
- **CSP**: Compatible with `script-src 'none'`
- **Accessibility**: Keyboard navigable, screen reader friendly
- **Mobile**: Responsive design, buttons wrap on small screens

## 🚀 Deployment

No special deployment steps required:
1. CSS hash updated: `37363243f168`
2. Config updated: `radius_km = 1.0`
3. All changes backward compatible
4. No database migrations needed

## 📋 Checklist

- [x] Backend text matching implemented
- [x] Backend distance bucketing implemented
- [x] Frontend radio buttons added
- [x] Frontend CSS filtering implemented
- [x] Unit tests written (50+ cases)
- [x] Documentation created
- [x] Code formatted (`gofmt`)
- [x] Config updated (radius_km = 1.0)
- [x] CSS hash updated

## 🔗 Related

- Issue: User requested distance slider feature
- Design: CSS-only implementation, no JavaScript
- References:
  - CSS `:checked` selector: https://developer.mozilla.org/en-US/docs/Web/CSS/:checked
  - Unicode normalization: https://pkg.go.dev/golang.org/x/text/unicode/norm

---

🤖 Generated with [Claude Code](https://claude.com/claude-code)
