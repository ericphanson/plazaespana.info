# UI Iteration Workflow with shot-scraper

**Date**: 2025-10-20
**Status**: Active

## Overview

Establish a systematic workflow for iterating on UI design using shot-scraper to capture screenshots, analyze visual design, identify improvements, and verify changes. This workflow enables rapid UI iteration with visual feedback.

## Context

**New capability**: `shot-scraper` is now available in the devcontainer for automated screenshot capture.

## Goals

1. **Systematic visual review**: Capture consistent screenshots of all key pages
2. **Multi-viewport testing**: Desktop, mobile, and full-page views
3. **Iterative improvement**: Screenshot → Review → Change → Verify cycle
4. **Visual documentation**: Track UI evolution over time

## Workflow

### Phase 1: Capture Baseline Screenshots

**Pages to screenshot**:

- `localhost:8080` - Main page
- `localhost:8080/build-report.html` - Build report


**Viewport configurations**:

- **Desktop full**: `--width 1400` (no height = full page scroll)
- **Desktop viewport**: `--width 1400 --height 900` (above-the-fold only)
- **Tablet**: `--width 768 --height 1024`
- **Mobile**: `--width 375 --height 812` (iPhone size)

**Screenshot script** (`/workspace/screenshots/capture.sh`):

```bash
#!/bin/bash
# UI Screenshot Capture Script
# Usage: ./capture.sh [timestamp]

set -e

TIMESTAMP=${1:-$(date +%Y%m%d-%H%M%S)}
OUTPUT_DIR="/workspace/screenshots/${TIMESTAMP}"
mkdir -p "${OUTPUT_DIR}"

BASE_URL="http://localhost:8000"

echo "📸 Capturing screenshots to ${OUTPUT_DIR}"

# Feed views
echo "  → Feed views..."
shot-scraper "${BASE_URL}/feed" -o "${OUTPUT_DIR}/feed-desktop.png" --width 1400 --height 900
shot-scraper "${BASE_URL}/feed" -o "${OUTPUT_DIR}/feed-full.png" --width 1400
shot-scraper "${BASE_URL}/feed" -o "${OUTPUT_DIR}/feed-mobile.png" --width 375 --height 812

# Smart filters
echo "  → Smart filters..."
for filter in reply needs-review my-prs-need-attention waiting-on-others unread quiet; do
  shot-scraper "${BASE_URL}/smart/${filter}" -o "${OUTPUT_DIR}/smart-${filter}.png" --width 1400 --height 900
done

# Repo filters (URL-encode the slashes)
echo "  → Repository filters..."
shot-scraper "${BASE_URL}/repo/JuliaLang%2Fjulia" -o "${OUTPUT_DIR}/repo-julia.png" --width 1400 --height 900
shot-scraper "${BASE_URL}/repo/JuliaLang%2FPkg.jl" -o "${OUTPUT_DIR}/repo-pkg.png" --width 1400 --height 900

# Reason filters
echo "  → Reason filters..."
shot-scraper "${BASE_URL}/filter/review_requested" -o "${OUTPUT_DIR}/filter-review.png" --width 1400 --height 900
shot-scraper "${BASE_URL}/filter/mention" -o "${OUTPUT_DIR}/filter-mention.png" --width 1400 --height 900

# System pages
echo "  → System pages..."
shot-scraper "${BASE_URL}/metrics/ui" -o "${OUTPUT_DIR}/metrics-ui.png" --width 1400 --height 900
shot-scraper "${BASE_URL}/settings" -o "${OUTPUT_DIR}/settings.png" --width 1400 --height 900

echo "✅ Screenshots captured: $(ls -1 ${OUTPUT_DIR}/*.png | wc -l) files"
echo "📁 Location: ${OUTPUT_DIR}"
ls -lh "${OUTPUT_DIR}" | grep "\.png$"
```

### Phase 2: Visual Review & Analysis

**Review process**:

1. **Open screenshots** using `Read` tool to view images
2. **Analyze each page** for:
   - Visual hierarchy (does the eye flow naturally?)
   - Color usage (appropriate contrast, not too harsh?)
   - Spacing & density (enough breathing room?)
   - Typography (readable, consistent sizing?)
   - Badge/indicator clarity (too many? too few?)
   - Responsive behavior (mobile/tablet views work?)
   - Accessibility concerns (contrast, focus states)

3. **Document findings** in a review checklist:
   ```markdown
   ## UI Review - [TIMESTAMP]

   ### Feed Page
   - ✅ Visual hierarchy clear
   - ⚠️  Badge density high on some items
   - ❌ Mobile: timestamp wrapping awkwardly

   ### Smart Filters
   - ✅ Empty state clear ("All caught up!")
   - ⚠️  Navigation could be more prominent

   [... continue for each page ...]
   ```

### Phase 3: Identify & Prioritize Improvements

**Categorize findings**:

- **Critical**: Breaks usability or accessibility
- **Important**: Significantly impacts user experience
- **Nice-to-have**: Polish and refinement

**Common improvement patterns**:

1. **Spacing issues**: Padding, margins, gaps
2. **Color problems**: Contrast, saturation, transparency
3. **Typography**: Size, weight, line-height, letter-spacing
4. **Layout**: Alignment, wrapping, overflow
5. **States**: Hover, focus, active, disabled
6. **Responsive**: Breakpoint behavior, mobile layout

### Phase 4: Implement Changes

**Iteration cycle**:

1. **Make CSS changes** to `/workspace/public/css/app.css`
2. **Rebuild site** `just kill && just dev`
3. **Capture new screenshots** (same script, new timestamp)
4. **Compare before/after** using `Read` tool on both sets
5. **Verify improvement** - did it fix the issue without creating new ones?
6. **Commit if satisfied** or iterate further

**Comparison technique**:

```bash
# Capture before making changes
./capture.sh baseline

# Make CSS changes
vim /workspace/public/css/app.css

# Capture after changes (server hot-reloads)
./capture.sh iteration-1

# View side-by-side in Claude Code
# Read screenshots/baseline/feed-desktop.png
# Read screenshots/iteration-1/feed-desktop.png
```

### Phase 5: Document & Commit

**When changes are ready**:

1. **Document in retro**: Create or update retro with:
   - What was changed and why
   - Before/after observations
   - Design decisions made
   - Any trade-offs or compromises

2. **Commit changes**:
   ```bash
   git add public/css/app.css
   git commit -m "refactor(ui): [description of changes]"
   ```

3. **Archive screenshots** (optional):
   - Keep baseline + final iteration
   - Delete intermediate iterations to save disk space
   - Or move to separate archive directory

## Directory Structure

```
/workspace/screenshots/
├── capture.sh              # Screenshot capture script
├── baseline/               # Initial screenshots before changes
│   ├── feed-desktop.png
│   ├── feed-mobile.png
│   └── ...
├── iteration-1/            # After first round of changes
├── iteration-2/            # After second round
└── final/                  # Final approved version (symlink or copy)
```

## Usage Examples

### Example 1: Badge Density Review

```bash
# Capture current state
cd /workspace/screenshots
./capture.sh baseline

# Review feed page
# Notice: too many badges create visual noise

# Change: Hide "subscribed" badge (already done in retro 032)
# Change: Reduce badge font-size from 11px to 10px
vim /workspace/public/css/app.css

# Capture after change
./capture.sh less-badges

# Compare
# Read screenshots/baseline/feed-desktop.png
# Read screenshots/less-badges/feed-desktop.png

# If good, commit; if not, revert and try different approach
```

### Example 2: Mobile Layout Check

```bash
# Capture mobile views
./capture.sh mobile-check

# Review all *-mobile.png files
# Notice: activity sentence wrapping awkwardly

# Change: Adjust flex layout and max-width for mobile
vim /workspace/public/css/app.css

# Capture again
./capture.sh mobile-fixed

# Compare mobile screenshots
```

### Example 3: Metrics Dashboard Redesign

```bash
# Current metrics page
./capture.sh metrics-before

# Redesign: Change grid layout, add section dividers, improve card styling

# Capture after each major change
./capture.sh metrics-grid
./capture.sh metrics-dividers
./capture.sh metrics-final

# Pick best iteration, commit
```

## Best Practices

1. **Always capture before changing**: Baseline is critical for comparison
2. **Use timestamps**: Avoids overwriting previous iterations
3. **Focus on one area at a time**: Don't try to fix everything at once
4. **Test multiple viewports**: Desktop changes can break mobile
5. **Review empty states**: Capture pages with no data (smart filters when empty)
6. **Check loaded states**: Capture pages with data (feed with many notifications)
7. **Watch disk space**: Delete intermediate iterations, keep baseline + final only
8. **Document decisions**: Not just what changed, but why

## Disk Space Management

**shot-scraper produces large PNGs**. To avoid running out of disk:

1. **Delete intermediate iterations** once final version is committed
2. **Keep only**: baseline + final for documentation
3. **Check disk usage**: `check-disk` before large screenshot sessions
4. **Compress if needed**: `pngcrush` or similar (not installed by default)

## Success Criteria

This workflow is successful when:

✅ UI changes are tested visually before committing
✅ Mobile/tablet layouts verified alongside desktop
✅ Before/after comparisons demonstrate clear improvement
✅ Design iterations documented for future reference
✅ No "oops, that broke the mobile view" surprises after commit

## Next Steps

1. Create `/workspace/screenshots/capture.sh` script
2. Capture initial baseline of current UI (post-retro-032)
3. Identify first improvement area (spacing? colors? mobile?)
4. Run first iteration cycle
5. Document learnings in new retro
