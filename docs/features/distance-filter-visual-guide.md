# Distance Filter - Visual Guide

**Feature**: CSS-only distance filtering with 5 preset distances
**Status**: ✅ Implemented and Tested
**Date**: 2025-10-23

## Overview

The distance filter allows users to show only events within a specific radius from Plaza de España. It's implemented entirely with CSS (no JavaScript), making it fast, accessible, and compatible with strict Content Security Policy.

## UI Components

### Distance Filter Buttons

Located in the header below the event counts, the filter presents 5 options:

- **En Plaza** (0m) - Events at Plaza de España itself
- **250m** - Events within 250 meters
- **500m** - Events within 500 meters
- **750m** - Events within 750 meters
- **1km (todos)** - All events (default)

The selected button is highlighted in blue with bold text.

### Default View (1km - All Events)

![Default view showing all events](../../screenshots/distance-filter/04-header-closeup.png)

**Default state:**
- "1km (todos)" button is selected (blue background)
- All events within 1km radius are visible
- Event counts: 45 city events, 65 cultural events

### Filtered View - 250m

![250m filter active](../../screenshots/distance-filter/05-filter-250m.png)

**When 250m is selected:**
- "250m" button highlighted in blue
- Only events ≤250m from Plaza de España are shown
- Event counts reduced to closest events only
- Visible sections: Past Weekend (1), This Weekend (3), This Week (1), Later This Month (1), Eventos en Curso (39)

### Full Desktop View

![Full page desktop view](../../screenshots/distance-filter/02-full-page-desktop.png)

**Desktop experience (1400px wide):**
- Events displayed in responsive grid (up to 3 columns)
- Distance filter buttons wrap if needed
- Cultural events toggle below distance filter
- Smooth filtering with no page reload

### Mobile View

![Mobile view (375px)](../../screenshots/distance-filter/03-mobile.png)

**Mobile experience:**
- Distance filter buttons stack/wrap on narrow screens
- Single column event layout
- All filtering functionality preserved
- Touch-friendly button sizing

## How It Works

### CSS-Only Implementation

The filter uses hidden radio buttons with styled labels:

```html
<!-- Hidden radio inputs -->
<input type="radio" name="distance-filter" id="distance-1000" value="1000" checked>
<input type="radio" name="distance-filter" id="distance-250" value="250">

<!-- Styled labels (the visible buttons) -->
<label for="distance-1000">1km (todos)</label>
<label for="distance-250">250m</label>
```

### Filtering Logic

Each event card has distance metadata:

```html
<article class="event-card"
  data-distance-m="320"
  data-distance-bucket="251-500"
  data-at-plaza="false">
```

CSS hides events based on selected filter:

```css
/* When 250m is selected, hide events not in 0-250m bucket */
#distance-250:checked ~ main .event-card:not([data-distance-bucket="0-250"]) {
  display: none;
}
```

### Special Case: "En Plaza" (0m)

The "En Plaza" filter shows only events specifically at Plaza de España:

```css
#distance-0:checked ~ main .event-card:not([data-at-plaza="true"]) {
  display: none;
}
```

Events are marked with `data-at-plaza="true"` when:
1. GPS coordinates place them at the plaza, OR
2. Text matching finds "Plaza de España" in title, venue, address, or description

Text matching is accent-insensitive and handles abbreviations:
- "Plaza de España" ✓
- "Pl. España" ✓
- "Pza. Espana" (no accent) ✓
- "PLAZA DE ESPAÑA" ✓

## User Experience

### Visual Feedback

- **Selected state**: Blue background, white text, bold font
- **Hover state**: Border color changes to accent blue
- **Default state**: White background, dark border

### Keyboard Navigation

- **Tab**: Navigate between buttons
- **Space/Enter**: Select button
- **Arrow keys**: Move between radio buttons in group

### Accessibility

- ✅ Screen reader compatible (radio button group)
- ✅ Keyboard navigable
- ✅ Clear visual feedback for selected state
- ✅ Works without JavaScript
- ✅ Semantic HTML (form controls)

## Technical Details

### Distance Bucketing

Events are categorized into 5 buckets for efficient CSS filtering:

| Bucket | Distance Range | CSS Attribute Value |
|--------|---------------|---------------------|
| 0-250m | 0-250 meters | `data-distance-bucket="0-250"` |
| 251-500m | 251-500 meters | `data-distance-bucket="251-500"` |
| 501-750m | 501-750 meters | `data-distance-bucket="501-750"` |
| 751-1000m | 751-1000 meters | `data-distance-bucket="751-1000"` |
| 1000m+ | >1000 meters | `data-distance-bucket="1000+"` |

### Backend Processing

1. **Calculate distance**: Haversine formula for GPS coordinates
2. **Bucket assignment**: `GetDistanceBucket(distanceMeters)` assigns bucket
3. **Text fallback**: Events without coordinates checked for Plaza de España mentions
4. **Template rendering**: Distance metadata added to HTML attributes

### Browser Compatibility

- ✅ All modern browsers (Chrome, Firefox, Safari, Edge)
- ✅ IE9+ (CSS :checked selector support)
- ✅ No JavaScript required
- ✅ Works with strict CSP (`script-src 'none'`)

## Testing

### Unit Tests

**Distance bucketing** (`internal/filter/geo_plaza_test.go`):
- ✅ 15 test cases covering all 5 buckets
- ✅ Boundary conditions (0m, 250m, 500m, 750m, 1000m)
- ✅ Edge cases (1001m, 10000m)

**Text matching** (`internal/filter/text_test.go`):
- ✅ 13 test cases for `MatchesPlazaEspana()`
- ✅ Accent variations, abbreviations, case sensitivity
- ✅ Negative cases (other plazas, partial matches)

### Visual Testing

Screenshots captured with shot-scraper:
- Desktop default view (1400x900)
- Full page desktop (1400px wide)
- Mobile view (375x812)
- Header close-up (distance filter UI)
- 250m filter active
- 500m filter active
- En Plaza filter active

### Manual Testing

To test locally:
```bash
git checkout claude/plaza-distance-slider-011CUPqk63SQfJdnrFRgfGm7
just dev
# Open http://localhost:8080
# Click each distance filter button
# Verify event counts change correctly
```

## Performance

- **No JavaScript**: Filter is instant (CSS-only)
- **No HTTP requests**: All filtering happens client-side
- **Minimal CSS**: ~50 lines for complete implementation
- **Small payload**: Distance metadata adds ~20 bytes per event

## Future Enhancements

1. **Dynamic event counts**: Show "(15 events)" next to each distance button
2. **URL parameters**: Persist selection in URL query string
3. **Local storage**: Remember user's last selection
4. **Animation**: Fade events in/out when filter changes
5. **Range slider**: Continuous slider from 0-1000m (requires JavaScript)

## Related Documentation

- Implementation details: [distance-slider.md](distance-slider.md)
- Screenshot workflow: [../../docs/screenshot-instructions.md](../screenshot-instructions.md)
- Test coverage: [geo_plaza_test.go](../../internal/filter/geo_plaza_test.go)

## Screenshots

All screenshots available in `screenshots/distance-filter/`:
- `01-default-desktop.png` - Default view (1km, desktop)
- `02-full-page-desktop.png` - Full page scroll (desktop)
- `03-mobile.png` - Mobile view (375px)
- `04-header-closeup.png` - Header with distance filter
- `05-filter-250m.png` - 250m filter active
- `06-filter-500m.png` - 500m filter active
- `07-filter-0m-plaza.png` - En Plaza filter active

---

**Implementation Date**: 2025-10-23
**Branch**: `claude/plaza-distance-slider-011CUPqk63SQfJdnrFRgfGm7`
**PR**: [Link to PR]
**Status**: ✅ Ready for production
