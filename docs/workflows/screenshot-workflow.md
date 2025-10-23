# UI Iteration Workflow with shot-scraper

**Date**: 2025-10-20
**Status**: Active

## Overview

Establish a systematic workflow for iterating on UI design using shot-scraper to capture screenshots, analyze visual design, identify improvements, and verify changes. This workflow enables rapid UI iteration with visual feedback for the Madrid events listing page.

## Context

**New capability**: `shot-scraper` is now available in the devcontainer for automated screenshot capture.

**Target**: The static events site displays filtered events near Plaza de España with hand-rolled CSS, dark mode support, and responsive design.

## Goals

1. **Systematic visual review**: Capture consistent screenshots of the events listing page
2. **Multi-viewport testing**: Desktop, tablet, and mobile views (responsive design verification)
3. **Iterative improvement**: Screenshot → Review → Change → Verify cycle
4. **Visual documentation**: Track UI evolution over time
5. **State testing**: Empty state (no events), populated state (multiple events), and various event types

## Workflow

### Phase 1: Capture Baseline Screenshots

**Pages to screenshot**:

- `localhost:8080` - Events listing page (main/only page)

**Viewport configurations**:

- **Desktop full**: `--width 1400` (no height = full page scroll)
- **Desktop viewport**: `--width 1400 --height 900` (above-the-fold only)
- **Tablet**: `--width 768 --height 1024`
- **Mobile**: `--width 375 --height 812` (iPhone size)

**Screenshot script** (`scripts/capture.sh`):

```bash
#!/bin/bash
# Madrid Events UI Screenshot Capture Script
# Usage: ./scripts/capture.sh [timestamp]

set -e

TIMESTAMP=${1:-$(date +%Y%m%d-%H%M%S)}
OUTPUT_DIR="screenshots/${TIMESTAMP}"
mkdir -p "${OUTPUT_DIR}"

BASE_URL="http://localhost:8080"

echo "📸 Capturing screenshots to ${OUTPUT_DIR}"

# Main events page - multiple viewports
echo "  → Events page (desktop full)..."
shot-scraper "${BASE_URL}" -o "${OUTPUT_DIR}/events-desktop-full.png" --width 1400

echo "  → Events page (desktop viewport)..."
shot-scraper "${BASE_URL}" -o "${OUTPUT_DIR}/events-desktop.png" --width 1400 --height 900

echo "  → Events page (tablet)..."
shot-scraper "${BASE_URL}" -o "${OUTPUT_DIR}/events-tablet.png" --width 768 --height 1024

echo "  → Events page (mobile)..."
shot-scraper "${BASE_URL}" -o "${OUTPUT_DIR}/events-mobile.png" --width 375 --height 812

echo "✅ Screenshots captured: $(ls -1 ${OUTPUT_DIR}/*.png | wc -l) files"
echo "📁 Location: ${OUTPUT_DIR}"
ls -lh "${OUTPUT_DIR}" | grep "\.png$"
```

### Phase 2: Visual Review & Analysis

**Review process**:

1. **Open screenshots** using `Read` tool to view images
2. **Analyze each page** for:
   - Visual hierarchy (event title → date/time → venue → link flow naturally?)
   - Color usage (appropriate contrast, especially in dark mode?)
   - Spacing & density (cards have breathing room? not too cramped?)
   - Typography (event titles prominent, metadata readable, consistent sizing?)
   - Event card layout (information clear at a glance?)
   - Date/time formatting (concise but unambiguous?)
   - Responsive behavior (mobile/tablet views work? cards stack properly?)
   - Accessibility concerns (contrast ratios, link underlines, focus states)
   - Attribution footer (visible but not intrusive?)

3. **Document findings** in a review checklist:
   ```markdown
   ## UI Review - [TIMESTAMP]

   ### Events Listing Page - Desktop
   - ✅ Event cards have clear visual hierarchy
   - ⚠️  Date formatting could be more concise
   - ❌ Long venue names overflow container

   ### Events Listing Page - Mobile
   - ✅ Cards stack nicely
   - ⚠️  Event time wrapping awkwardly on small screens
   - ✅ Attribution footer readable

   ### Overall
   - ✅ Dark mode works well
   - ⚠️  Spacing between cards could be more consistent
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

1. **Make CSS changes** to `/workspace/assets/site.css`
2. **Rebuild site** with `just hash-css && just build` (to regenerate CSS hash)
3. **Restart dev server** if needed: `just dev`
4. **Capture new screenshots** (same script, new timestamp)
5. **Compare before/after** using `Read` tool on both sets
6. **Verify improvement** - did it fix the issue without creating new ones?
7. **Commit if satisfied** or iterate further

**Comparison technique**:

```bash
# Capture before making changes
./scripts/capture.sh baseline

# Make CSS changes
vim /workspace/assets/site.css

# Regenerate hashed CSS and rebuild
just hash-css && just build

# Capture after changes
./scripts/capture.sh iteration-1

# View side-by-side in Claude Code
# Read screenshots/baseline/events-desktop.png
# Read screenshots/iteration-1/events-desktop.png
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
   git add assets/site.css public/
   git commit -m "refactor(ui): [description of changes]"
   ```

3. **Archive screenshots** (optional):
   - Keep baseline + final iteration
   - Delete intermediate iterations to save disk space
   - Or move to separate archive directory

## Directory Structure

```
scripts/
└── capture.sh                  # Screenshot capture script

screenshots/                    # Output directory (gitignored)
├── baseline/                   # Initial screenshots before changes
│   ├── events-desktop-full.png
│   ├── events-desktop.png
│   ├── events-tablet.png
│   └── events-mobile.png
├── iteration-1/                # After first round of changes
├── iteration-2/                # After second round
└── final/                      # Final approved version (symlink or copy)
```

## Usage Examples

### Example 1: Event Card Spacing Review

```bash
# Capture current state

./scripts/capture.sh baseline

# Review events page
# Notice: Event cards feel cramped, need more breathing room

# Change: Increase margin between cards
# Change: Add more padding inside each card
vim /workspace/assets/site.css

# Rebuild with hashed CSS
just hash-css && just build

# Capture after change
./scripts/capture.sh more-spacing

# Compare
# Read screenshots/baseline/events-desktop.png
# Read screenshots/more-spacing/events-desktop.png

# If good, commit; if not, revert and try different approach
```

### Example 2: Mobile Layout Check

```bash
# Capture mobile views
./scripts/capture.sh mobile-check

# Review mobile screenshots
# Notice: Event dates wrapping awkwardly on small screens
# Notice: Venue names too long, pushing times off screen

# Change: Use more compact date format for mobile
# Change: Truncate venue names with ellipsis
# Change: Stack time below venue on narrow screens
vim /workspace/assets/site.css

# Rebuild
just hash-css && just build

# Capture again
./scripts/capture.sh mobile-fixed

# Compare mobile screenshots
# Read screenshots/mobile-check/events-mobile.png
# Read screenshots/mobile-fixed/events-mobile.png
```

### Example 3: Dark Mode Color Refinement

```bash
# Current dark mode
./scripts/capture.sh dark-mode-before

# Redesign: Adjust background colors for better contrast
# Change: Soften link colors to reduce eye strain
# Change: Improve event card borders for dark theme

vim /workspace/assets/site.css

# Capture after each major change
just hash-css && just build
./scripts/capture.sh dark-mode-contrast

# More adjustments to link colors
vim /workspace/assets/site.css
just hash-css && just build
./scripts/capture.sh dark-mode-links

# Final adjustments
vim /workspace/assets/site.css
just hash-css && just build
./scripts/capture.sh dark-mode-final

# Review all iterations, pick best version, commit
```

## Best Practices

1. **Always capture before changing**: Baseline is critical for comparison
2. **Use timestamps**: Avoids overwriting previous iterations
3. **Focus on one area at a time**: Don't try to fix everything at once
4. **Test multiple viewports**: Desktop changes can break mobile
5. **Review empty states**: Capture page when no events match filters (rare, but possible)
6. **Check loaded states**: Capture page with multiple events (typical state)
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

1. Ensure `scripts/capture.sh` is executable
2. Start development server with `just dev`
3. Capture initial baseline of current events UI
4. Identify first improvement area (event card layout? date formatting? mobile responsiveness?)
5. Run first iteration cycle
6. Document any significant UI changes in project log
