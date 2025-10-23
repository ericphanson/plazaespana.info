# Distance Filter Feature

**Status**: Implemented
**Date**: 2025-10-23

## Overview

Users can now filter events by distance from Plaza de España using a CSS-only slider with 5 discrete distance options.

## User Interface

The distance filter appears in the header below the page title, as a row of clickable buttons:

- **En Plaza** (0m): Shows only events at Plaza de España itself
- **250m**: Shows events within 250 meters
- **500m**: Shows events within 500 meters
- **750m**: Shows events within 750 meters
- **1km (todos)**: Shows all events (default)

### Design

- **CSS-only**: No JavaScript required, works with strict CSP
- **Radio buttons**: Hidden inputs with styled labels
- **Visual feedback**: Selected button highlighted with accent color
- **Responsive**: Buttons wrap on mobile devices

## Implementation

### Backend

1. **Distance Calculation** (`internal/filter/geo.go`):
   - `GetDistanceBucket()`: Categorizes distances into 5 buckets for CSS filtering
   - Buckets: "0-250", "251-500", "501-750", "751-1000", "1000+"

2. **Text Matching** (`internal/filter/geo.go`):
   - `IsAtPlazaEspana()`: Identifies venues at Plaza de España by name
   - Loose matching: case-insensitive, accent-insensitive
   - Patterns: "Plaza de España", "Pl. España", "Plaza España", etc.
   - Handles events without GPS coordinates

3. **Event Metadata** (`internal/render/types.go`):
   - `DistanceMeters`: Distance in meters (0-1000+)
   - `DistanceBucket`: Bucket string for CSS filtering
   - `AtPlaza`: Boolean flag for "En Plaza" filter

4. **Rendering** (`internal/render/grouping.go`):
   - Calculates distance for events with GPS coordinates
   - Falls back to text matching for events without coordinates
   - Populates all distance metadata fields

### Frontend

1. **HTML** (`templates/index-grouped.tmpl.html`):
   - 5 hidden radio button inputs (`name="distance-filter"`)
   - Styled labels as clickable buttons
   - Event cards have `data-distance-m` and `data-distance-bucket` attributes
   - Special `data-at-plaza="true"` for venue name matches

2. **CSS** (`assets/site.css`):
   - Distance filter styling (button grid, hover states)
   - `:checked` pseudo-class triggers filtering rules
   - Hides events outside selected threshold using `display: none`
   - Active button styling (background, border, font weight)

### CSS Filtering Logic

```css
/* En Plaza (0m): Show only events with data-at-plaza="true" */
#distance-0:checked ~ main .event-card:not([data-at-plaza="true"]) {
  display: none;
}

/* 250m: Show only 0-250m bucket */
#distance-250:checked ~ main .event-card:not([data-distance-bucket="0-250"]) {
  display: none;
}

/* 500m: Hide 501-750m, 751-1000m, and 1000+ buckets */
#distance-500:checked ~ main .event-card[data-distance-bucket="501-750"],
#distance-500:checked ~ main .event-card[data-distance-bucket="751-1000"],
#distance-500:checked ~ main .event-card[data-distance-bucket="1000+"] {
  display: none;
}

/* 750m: Hide 751-1000m and 1000+ buckets */
#distance-750:checked ~ main .event-card[data-distance-bucket="751-1000"],
#distance-750:checked ~ main .event-card[data-distance-bucket="1000+"] {
  display: none;
}

/* 1km: Show all (no rules needed, default state) */
```

## Configuration

Updated `config.toml`:
```toml
[filter]
radius_km = 1.0  # Increased from 0.35km to support distance slider
```

The backend now fetches events up to 1km radius. The frontend filters provide user control within that range.

## Testing

### Unit Tests

**Text Matching** (`internal/filter/geo_plaza_test.go`):
- 28 test cases for `IsAtPlazaEspana()`
- Covers: accents, case variations, abbreviations, partial matches
- Negative cases: empty strings, different plazas, unrelated venues

**Distance Bucketing** (`internal/filter/geo_plaza_test.go`):
- 15 test cases for `GetDistanceBucket()`
- Covers: boundary conditions, all 5 buckets
- Edge cases: 0m, 250m, 500m, 750m, 1000m, 1001m, 10000m

**Text Normalization** (`internal/filter/geo_plaza_test.go`):
- 7 test cases for `normalizeText()`
- Covers: Spanish, French, German accents
- Case conversion, mixed input

### Visual Testing

Use `shot-scraper` to capture screenshots of each distance filter state:

```bash
# Start dev server
just dev

# Capture baseline
cd screenshots
./capture.sh baseline

# Test each distance filter manually:
# - Select "En Plaza" -> verify only Plaza events shown
# - Select "250m" -> verify events within 250m
# - Select "500m" -> verify events within 500m
# - Select "750m" -> verify events within 750m
# - Select "1km (todos)" -> verify all events shown

# Capture screenshots at each state
./capture.sh distance-0m
./capture.sh distance-250m
./capture.sh distance-500m
./capture.sh distance-750m
./capture.sh distance-1000m
```

## Dependencies

- `golang.org/x/text`: Unicode normalization (accent removal)
  - Already in project (`go.mod`)
  - Used by `normalizeText()` function

## Accessibility

- Keyboard navigable: Radio buttons accessible via Tab and Arrow keys
- Screen readers: Labels announce distance options
- Visual feedback: Selected state clearly indicated
- No motion/animation: Static button states
- Color contrast: Meets WCAG AA standards

## Browser Compatibility

- **CSS :checked selector**: Supported in all modern browsers (IE9+)
- **CSS attribute selectors**: Supported universally
- **CSS sibling combinators (~)**: Supported universally
- **No JavaScript**: Works even with JS disabled

## Future Enhancements

1. **Dynamic event counts**: Show number of events per distance range
2. **Smooth slider**: Replace discrete buttons with HTML5 range input (requires JS)
3. **Persistent selection**: Store preference in URL query parameter or localStorage
4. **Animation**: Fade events in/out when filter changes (CSS transitions)
5. **Accessibility audit**: Test with screen readers and keyboard-only navigation

## Known Limitations

1. **Discrete steps**: Not a smooth slider (0-1000m), only 5 preset distances
2. **No event count preview**: User doesn't know how many events per distance until selecting
3. **Text matching heuristics**: May miss unusual Plaza de España name variations
4. **No distance for unlocated events**: Events without coordinates and no Plaza match are hidden (except at 1km)

## Files Changed

- `internal/filter/geo.go`: Text matching + bucketing functions
- `internal/filter/geo_plaza_test.go`: Comprehensive tests
- `internal/render/types.go`: TemplateEvent struct (3 new fields)
- `internal/render/grouping.go`: Distance metadata population
- `templates/index-grouped.tmpl.html`: Radio buttons + data attributes
- `assets/site.css`: Filter UI styling + filtering rules
- `config.toml`: Radius increased to 1.0km

## References

- CSS :checked selector: https://developer.mozilla.org/en-US/docs/Web/CSS/:checked
- CSS attribute selectors: https://developer.mozilla.org/en-US/docs/Web/CSS/Attribute_selectors
- Unicode normalization: https://pkg.go.dev/golang.org/x/text/unicode/norm
