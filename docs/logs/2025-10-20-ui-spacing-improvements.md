# UI Iteration: Event Card Spacing Improvements

**Date**: 2025-10-20
**Type**: UI Refinement
**Status**: Completed

## Context

First systematic UI iteration using the new screenshot workflow. After deploying the Madrid events site to production, we wanted to validate the visual design and identify areas for improvement through systematic screenshot review.

## Process

### Phase 1: Baseline Screenshot Capture

Used the new `screenshots/capture.sh` script to capture baseline screenshots across 4 viewports:
- Desktop full page (1400px width, full height)
- Desktop viewport (1400x900)
- Tablet (768x1024)
- Mobile (375x812)

**Location**: `screenshots/baseline/`

### Phase 2: Visual Review

Analyzed baseline screenshots systematically:

**Desktop View Assessment:**
- ✅ Clear visual hierarchy with purple accent colors
- ✅ Event cards have good contrast and left border accent
- ✅ Badge system works well for event categorization
- ⚠️  **Cards feel cramped**: 1rem gap between cards felt tight with 156 events
- ⚠️  **Internal padding could be more generous**: 1rem padding didn't provide enough breathing room

**Mobile View Assessment:**
- ✅ Cards stack perfectly, no overflow issues
- ✅ All information remains readable
- ⚠️  Same spacing issues as desktop, slightly more noticeable on small screens

**Key Finding**: The spacing density made the page feel cramped and reduced scannability when browsing many events.

### Phase 3: CSS Changes

Made targeted improvements to `assets/site.css`:

1. **Increased card spacing** (line 66):
   - Before: `gap: 1rem;` (16px)
   - After: `gap: 1.5rem;` (24px)
   - Impact: +50% vertical space between cards

2. **Increased card padding** (line 107):
   - Before: `padding: 1rem;` (16px)
   - After: `padding: 1.25rem;` (20px)
   - Impact: +25% internal breathing room

3. **Adjusted metadata spacing** (line 153):
   - Before: `margin: 0.25rem 0;` (4px)
   - After: `margin: 0.4rem 0;` (6.4px)
   - Impact: +60% space between date/venue lines

**Build Process:**
```bash
just hash-css && just build
./build/buildsite -config config.toml
```

### Phase 4: Comparison

Captured new screenshots with identifier `improved-spacing`.

**Before/After Comparison:**

Desktop view:
- Baseline: Cards visually blend together, requires effort to separate events mentally
- Improved: Clear visual separation, each card stands as distinct unit
- Result: **Significantly better scannability**

Mobile view:
- Baseline: Tight spacing acceptable but not optimal
- Improved: More comfortable reading experience, less visual density
- Result: **Noticeably more polished**

File size comparison:
- Desktop screenshot: 72KB → 67KB (cards take up more vertical space, fewer fit in viewport)
- Mobile screenshot: 46KB → 44KB
- Full page: 3.1MB (unchanged, same number of events)

## Decision

**✅ Ship these changes**

The improvements are:
1. **Visually noticeable**: Clear difference in before/after screenshots
2. **Unambiguously positive**: No downsides identified
3. **Responsive-safe**: Works well across all viewports
4. **Consistent with design system**: Still uses relative units (rem)

## Technical Details

**CSS changes** (`assets/site.css`):
- `.event-section { gap: 1.5rem; }` ← was 1rem
- `.event-card { padding: 1.25rem; }` ← was 1rem
- `.when, .where { margin: 0.4rem 0; }` ← was 0.25rem

**Build artifacts updated:**
- `public/assets/site.e68b0d68.css` (new hash from content change)
- `public/index.html` (references new CSS hash)

## Workflow Validation

This iteration validated the screenshot workflow:

**What worked well:**
- ✅ Systematic viewport coverage caught issues across devices
- ✅ Before/after comparison provided concrete evidence of improvement
- ✅ Shot-scraper integration seamless after initial Playwright install
- ✅ Capture script (`screenshots/capture.sh`) worked perfectly
- ✅ Visual review much faster than manual browser resizing
- ✅ `just kill` command simplified server management

**Process improvements identified:**
- Could add dark mode screenshots using `--color-scheme dark` flag
- Could create comparison script to show before/after side-by-side
- Consider capturing at additional breakpoints (e.g., 1024px for small laptops)

## Lessons Learned

1. **Small spacing changes have large UX impact**: +50% gap spacing dramatically improved readability
2. **Screenshot workflow reduces guesswork**: Concrete visual evidence builds confidence in changes
3. **Multi-viewport testing essential**: Mobile and desktop both benefited from same changes
4. **Systematic process prevents bikeshedding**: Clear before/after comparison makes decision obvious

## Next Iterations

Potential future UI improvements identified during review:
1. Date/time format could be more concise (consider removing seconds from "00:00")
2. Dark mode color refinement (soften link colors, adjust card borders)
3. Empty state design (if no events match filters)
4. Footer visibility check (confirm attribution is readable at bottom of long page)

## Screenshot References

**Baseline**: `screenshots/baseline/`
**After changes**: `screenshots/improved-spacing/`

Key comparisons:
- Desktop: Compare `baseline/events-desktop.png` vs `improved-spacing/events-desktop.png`
- Mobile: Compare `baseline/events-mobile.png` vs `improved-spacing/events-mobile.png`

## Conclusion

First use of screenshot workflow was successful. CSS changes improved visual breathing room across all viewports. Changes committed and ready for deployment.

**Status**: ✅ Complete
**Committed**: 2025-10-20
