# Site Quality Scanning

Local scanning tools to catch broken links, performance issues, and HTML errors before deployment.

## Quick Start

```bash
# Terminal 1: Start local server
just serve

# Terminal 2: Run all scans
just scan
```

Results are saved to `scan-results/` directory.

---

## Installation

### Recommended Setup (Full Tools)

```bash
# muffet - Fast link checker (Go)
go install github.com/raviqqe/muffet/v2@latest

# Lighthouse - Performance auditor (npm)
npm install -g lighthouse

# HTML validator (npm)
npm install -g html-validator-cli
```

### Minimum Setup (Fallback Tools)

If you have `npm`/`npx` installed, all tools can run without installation via `npx` (downloads on first use).

If you have neither, basic checks run using `curl` (limited functionality).

---

## Commands

| Command | Purpose | Priority |
|---------|---------|----------|
| `just scan` | Run all scans | 🌟 Main command |
| `just scan-links` | Check broken links, 404s, missing assets | 🔴 P1 |
| `just scan-performance` | Lighthouse audit (Core Web Vitals) | 🟡 P2 |
| `just scan-html` | HTML validation, structure checks | 🟢 P3 |

**Note:** Site must be running on `http://localhost:8080` first (`just serve`).

---

## Interpreting Results

### 1. Link Checking (`scan-links`)

**File:** `scan-results/links.txt`

#### What to Look For:

```
✅ GOOD: No errors
200 http://localhost:8080/
200 http://localhost:8080/events.json
200 http://localhost:8080/assets/style-abc123.css
```

```
❌ BAD: Missing assets or broken links
404 http://localhost:8080/assets/missing-icon.png
404 http://localhost:8080/old-page.html
ERR http://localhost:8080/broken
```

#### Common Issues:

- **404 errors:** Missing files, typos in paths
- **Broken images:** Check `<img src="...">` paths
- **CSS/asset 404s:** Run `just hash-css` to regenerate hashed files
- **External link failures:** Normal (upstream sites change), focus on internal links

#### Action Items:

1. Fix 404s for internal resources (assets, pages)
2. Update or remove broken image references
3. External 404s: Consider removing stale links or updating URLs

---

### 2. Performance (`scan-performance`)

**Files:**
- `scan-results/lighthouse.report.html` - Open in browser for visual report
- `scan-results/lighthouse.report.json` - Machine-readable data

#### Key Metrics (Lighthouse Scores):

| Metric | Good | Fair | Poor | What It Measures |
|--------|------|------|------|------------------|
| **Performance** | 90+ | 50-89 | <50 | Load speed, responsiveness |
| **Accessibility** | 90+ | 50-89 | <50 | Screen readers, contrast, semantics |
| **Best Practices** | 90+ | 50-89 | <50 | Modern web standards |
| **SEO** | 90+ | 50-89 | <50 | Search engine optimization |

#### Core Web Vitals:

- **LCP (Largest Contentful Paint):** <2.5s good, >4s poor
  - *Measures:* When main content loads
  - *Fix:* Optimize images, reduce CSS size, use content hashing

- **CLS (Cumulative Layout Shift):** <0.1 good, >0.25 poor
  - *Measures:* Visual stability (no jumping content)
  - *Fix:* Set image dimensions, avoid injecting content above fold

- **FID/INP (First Input Delay / Interaction to Next Paint):** <100ms good, >300ms poor
  - *Measures:* Responsiveness to user input
  - *Fix:* N/A for our site (no JavaScript!)

#### Expected Results for plazaespana.info:

- ✅ **Performance: 95-100** (static site, minimal assets)
- ✅ **Accessibility: 90-100** (semantic HTML, alt text)
- ✅ **Best Practices: 90-100** (strong security headers)
- ✅ **SEO: 90-100** (meta tags, semantic structure)

#### Common Issues:

- **Low Performance (<90):**
  - Unoptimized images (use WebP, set dimensions)
  - CSS not minified/gzipped
  - Missing cache headers

- **Low Accessibility (<90):**
  - Missing alt text on images
  - Poor color contrast
  - Missing ARIA labels
  - Non-semantic HTML

- **Low SEO (<90):**
  - Missing meta description
  - No `<title>` tag
  - Missing `<h1>` or poor heading hierarchy

---

### 3. HTML Validation (`scan-html`)

**File:** `scan-results/html-validation.txt`

#### What to Look For:

```
✅ GOOD:
✅ DOCTYPE present
✅ Charset declared
✅ HTML properly closed
✅ Viewport meta tag present
📸 Images: 15 total, 15 with alt text
```

```
❌ BAD:
❌ DOCTYPE missing
⚠️  Charset not found
❌ Missing </html>
📸 Images: 20 total, 5 with alt text
```

#### Common Issues:

- **Missing DOCTYPE:** Add `<!DOCTYPE html>` at top of template
- **Missing charset:** Add `<meta charset="utf-8">` in `<head>`
- **Unclosed tags:** Check template for missing `</div>`, `</section>`, etc.
- **Images without alt text:** Add `alt=""` for decorative, `alt="description"` for content
- **Missing viewport:** Add `<meta name="viewport" content="width=device-width, initial-scale=1">`

#### W3C Validation:

For detailed standards compliance, use W3C validator:
- Web: https://validator.w3.org/
- Upload: `scan-results/index.html`
- Or paste URL when site is publicly accessible

---

## Workflow Integration

### Before Committing Changes:

```bash
# 1. Rebuild site
just generate

# 2. Start server (terminal 1)
just serve

# 3. Run scans (terminal 2)
just scan

# 4. Review results, fix issues
open scan-results/lighthouse.report.html
cat scan-results/links.txt

# 5. Commit when all scans pass
git add . && git commit -m "fix: resolve broken links and performance issues"
```

### CI/CD Integration (Future):

Add to `.github/workflows/`:

```yaml
- name: Quality Scan
  run: |
    just generate
    just serve &
    sleep 5
    just scan
    # Fail if critical issues found
    grep -q "404" scan-results/links.txt && exit 1 || true
```

---

## Scanning Remote Site

To scan production (https://plazaespana.info):

```bash
# Links (muffet)
muffet https://plazaespana.info

# Performance (Lighthouse)
lighthouse https://plazaespana.info --view

# HTML validation
html-validator https://plazaespana.info

# Or use web tools (see docs/security-scanning-guide.md)
```

---

## Troubleshooting

**"Site not running on http://localhost:8080"**
- Run `just serve` in another terminal first
- Or combine: `just dev` (builds + serves)

**"muffet: command not found"**
- Install: `go install github.com/raviqqe/muffet/v2@latest`
- Ensure `$GOPATH/bin` is in your `$PATH`

**"lighthouse: command not found"**
- Install: `npm install -g lighthouse`
- Or let `npx` download it automatically (slower first run)

**"npx broken-link-checker hangs"**
- Use muffet instead (much faster)
- Or add `--max-connections=1` flag

**Lighthouse fails with Chrome errors:**
- Ensure Chrome/Chromium is installed
- Linux: `apt-get install chromium-browser`
- macOS: Lighthouse uses installed Chrome
- Add `--chrome-flags="--no-sandbox"` if running as root (not recommended)

---

## Tool Comparison

| Tool | Speed | Accuracy | Installation | Best For |
|------|-------|----------|--------------|----------|
| **muffet** | ⚡️ Fast | ✅ High | Go | Link checking |
| **broken-link-checker** | 🐌 Slow | ✅ High | npm | Detailed reports |
| **Lighthouse** | 🏃 Medium | ✅ High | npm | Performance + accessibility |
| **html-validator-cli** | 🏃 Medium | ✅ High | npm | Standards compliance |
| **curl fallback** | ⚡️ Fast | ⚠️ Basic | Built-in | Quick sanity checks |

---

## Related Documentation

- **Security scans:** `docs/security-scanning-guide.md` (Mozilla Observatory, SSL Labs, etc.)
- **Build reports:** `public/build-report.html` (data pipeline metrics)
- **Deployment:** `docs/deployment.md`

---

**Last Updated:** 2025-10-24
