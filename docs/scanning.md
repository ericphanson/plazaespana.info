# Site Quality Scanning

Scan local or production site to catch broken links, performance issues, and HTML errors.

## Quick Start

**Local site:**
```bash
# Terminal 1: Start local server
just serve

# Terminal 2: Run all scans
just scan
```

**Production site:**
```bash
just scan https://plazaespana.info
```

Results saved to `scan-results/` (git-ignored).

## Requirements

- **Node.js** with npm/npx (Node 18+ recommended)
- Chromium/Chrome (auto-downloaded by Lighthouse if needed)

## Commands

```bash
just scan [URL]                  # Run all scans (default: localhost:8080)
just scan-links [URL]            # Check broken links, 404s, missing assets
just scan-performance [URL]      # Lighthouse audit (Core Web Vitals)
just scan-html [URL]             # HTML validation

# Examples:
just scan                                    # Scan localhost
just scan https://plazaespana.info          # Scan production
just scan-performance https://plazaespana.info  # Just performance on prod
```

## Interpreting Results

### 1. Links (`scan-results/links.txt`)

**Good:**
```
✓ http://localhost:8080/
✓ http://localhost:8080/events.json
✓ http://localhost:8080/assets/style-abc123.css
```

**Bad:**
```
✖ http://localhost:8080/assets/missing.png (404)
✖ http://localhost:8080/broken (500)
```

**Fix:** Verify file paths, re-run `just hash-css` if CSS 404s.

---

### 2. Performance (`scan-results/lighthouse.report.html`)

Open in browser for visual report.

**Key Scores (0-100):**
- **Performance:** 90+ = good, <50 = poor
- **Accessibility:** 90+ = good, <50 = poor
- **Best Practices:** 90+ = good, <50 = poor
- **SEO:** 90+ = good, <50 = poor

**Expected for plazaespana.info:** 95-100 across the board (static site, minimal assets, no JS).

**Core Web Vitals:**
- **LCP (Largest Contentful Paint):** <2.5s good
- **CLS (Cumulative Layout Shift):** <0.1 good

**Common fixes:**
- Low performance: Optimize images, enable compression
- Low accessibility: Add alt text, improve contrast
- Low SEO: Add meta description, fix heading hierarchy

---

### 3. HTML (`scan-results/html-validation.txt`)

**Good:**
```
The document validates according to the specified schema(s).
```

**Bad:**
```
Error: Element div not allowed as child of span in this context
Warning: Section lacks heading
```

**Fix:** Correct invalid HTML structure, add missing semantic elements.

---

## Workflow

**Before committing:**
```bash
just generate       # Rebuild site
just serve          # Terminal 1
just scan           # Terminal 2 - scan localhost
# Review results, fix issues, repeat
```

**After deployment:**
```bash
just scan https://plazaespana.info  # Verify production
```

---

**Last Updated:** 2025-10-24
