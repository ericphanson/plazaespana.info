# Screenshot Instructions for Distance Filter PR

Since `shot-scraper` requires a browser and can't run in the CI sandbox, you'll need to capture screenshots locally to demonstrate the distance filter UI.

## Setup

1. Ensure you're on the feature branch:
   ```bash
   git checkout claude/plaza-distance-slider-011CUPqk63SQfJdnrFRgfGm7
   ```

2. Start the development server:
   ```bash
   just dev
   ```

   This will build the site and start a local server at `http://localhost:8080`

## Screenshot Workflow

### Option 1: Using shot-scraper (from devcontainer)

The devcontainer has `shot-scraper` pre-installed with Playwright/Chromium.

```bash
# Create screenshots directory
mkdir -p screenshots/distance-filter

# Capture default state (1km filter, all events)
shot-scraper http://localhost:8080 \
  --width 1400 \
  --output screenshots/distance-filter/01-default-1km.png

# For each distance filter, manually:
# 1. Open http://localhost:8080 in a browser
# 2. Click the desired distance button
# 3. Use shot-scraper with JavaScript to simulate the click

# Or use browser dev tools to add :checked attribute, then screenshot
```

### Option 2: Manual Screenshots (easier!)

1. Open `http://localhost:8080` in your browser
2. For each distance filter setting, take a screenshot:
   - **Default (1km)**: All events visible
   - **Click "750m"**: Fewer events (only up to 750m)
   - **Click "500m"**: Even fewer events (only up to 500m)
   - **Click "250m"**: Closest events only (0-250m)
   - **Click "En Plaza"**: Only events at Plaza de España itself

3. Save screenshots with descriptive names:
   - `distance-filter-1km-default.png`
   - `distance-filter-750m.png`
   - `distance-filter-500m.png`
   - `distance-filter-250m.png`
   - `distance-filter-0m-en-plaza.png`

4. Focus on capturing:
   - The distance filter buttons (show active state)
   - The event cards below (show which events are visible)
   - Desktop view is sufficient (mobile optional)

### Option 3: shot-scraper with Manual Clicks

```bash
# Terminal 1: Start dev server
just dev

# Terminal 2: Take screenshots
cd screenshots
mkdir -p distance-filter

# Default state (1km selected)
shot-scraper http://localhost:8080 \
  --width 1400 --height 900 \
  --output distance-filter/01-default-1km.png

# For other states, you'll need to:
# 1. Use browser dev tools to inspect the radio buttons
# 2. Add 'checked' attribute to desired radio button
# 3. Remove 'checked' from currently selected button
# 4. Take screenshot
# (This is tedious, manual screenshots are easier)
```

## What to Include in PR

1. **Before/After comparison**:
   - Screenshot showing the new distance filter UI (buttons in header)
   - Screenshot showing events filtered to 250m vs 1km

2. **Key screenshots**:
   - Distance filter UI (close-up of buttons)
   - Full page with different filter selections
   - Mobile view (optional, but nice to have)

3. **GIF/Video** (optional but impressive):
   - Record clicking through different distance options
   - Show events appearing/disappearing as filter changes
   - Tools: QuickTime (Mac), OBS Studio, SimpleScreenRecorder (Linux)

## Tips

- Use browser zoom at 100% for consistent screenshots
- Clear any browser extensions that might affect the page
- Use private/incognito mode to avoid cookie banners
- Screenshot during daytime for light mode (or test dark mode!)
- Capture both desktop (1400px) and mobile (375px) widths

## PR Description Template

```markdown
## 📸 Screenshots

### Distance Filter UI
![Distance filter buttons](screenshots/distance-filter/filter-ui.png)

### Filtering in Action

**All events (1km - default)**
![1km filter](screenshots/distance-filter/01-default-1km.png)

**Medium range (500m)**
![500m filter](screenshots/distance-filter/03-filter-500m.png)

**Close events only (250m)**
![250m filter](screenshots/distance-filter/04-filter-250m.png)

**At Plaza de España (0m)**
![En Plaza filter](screenshots/distance-filter/05-filter-0m.png)

## 🎯 How to Test

1. Check out this branch
2. Run `just dev`
3. Open http://localhost:8080
4. Click each distance filter button
5. Verify events filter correctly
```

## Next Steps

After capturing screenshots:

1. Add them to the PR description
2. Commit screenshots to a `screenshots/` directory (optional)
3. Link to screenshots in PR body using relative paths or upload directly to GitHub PR
