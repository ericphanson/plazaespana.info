# Security Scanning Guide: How to Run Tools Locally

This guide shows you how to run security and quality scanning tools against plazaespana.info (or any website) on your local machine.

---

## Quick Reference

| Tool | Type | Installation | Command |
|------|------|--------------|---------|
| Mozilla Observatory | Web | None | Visit URL |
| SecurityHeaders.com | Web | None | Visit URL |
| SSL Labs | Web | None | Visit URL |
| testssl.sh | CLI | Git clone | `./testssl.sh plazaespana.info` |
| webhint | CLI | npm | `npx hint https://plazaespana.info` |

---

## 1. Mozilla HTTP Observatory

**What it tests:** HTTP security headers, CSP, HSTS, cookies, redirects, and more.

### Web Interface (Easiest)

Visit: https://developer.mozilla.org/en-US/observatory/

1. Enter `plazaespana.info` in the search box
2. Click "Scan"
3. Wait 30-60 seconds for results
4. Review detailed test breakdown

### API (For Automation)

```bash
# Initiate scan
curl -X POST "https://observatory-api.mdn.mozilla.net/api/v2/scan?host=plazaespana.info"

# Response includes scan ID and initial results:
# {
#   "id": 76055521,
#   "grade": "B+",
#   "score": 80,
#   "tests_passed": 9,
#   "tests_failed": 1
# }

# For full details, visit the web interface using the scan ID
```

**Note:** The v2 API doesn't provide detailed per-test results via JSON. Use the web interface for detailed analysis.

---

## 2. SecurityHeaders.com

**What it tests:** HTTP security headers (CSP, HSTS, X-Frame-Options, etc.)

### Web Interface (Easiest)

Visit: https://securityheaders.com/

1. Enter `plazaespana.info`
2. Check "Follow redirects" if needed
3. Click "Scan"
4. Review grade and missing headers

### API (Requires API Key)

```bash
# Purchase API key at https://securityheaders.com/api/
# Then:
curl -H "x-api-key: YOUR_KEY_HERE" \
  "https://api.securityheaders.com/?q=plazaespana.info&hide=on&followRedirects=on"
```

---

## 3. SSL Labs (Qualys)

**What it tests:** SSL/TLS configuration, certificate validity, cipher suites, vulnerabilities.

### Web Interface (Easiest)

Visit: https://www.ssllabs.com/ssltest/

1. Enter `plazaespana.info`
2. Click "Submit"
3. Wait 2-3 minutes (thorough scan)
4. Review grade (A+ to F)

### API (Free, No Key Required)

```bash
# Initiate scan
curl "https://api.ssllabs.com/api/v3/analyze?host=plazaespana.info"

# Check status (scan takes 60-120 seconds)
curl "https://api.ssllabs.com/api/v3/analyze?host=plazaespana.info"

# When status="READY", full results are available in JSON
```

**API Limitations:**
- Rate limited (1 scan per host per hour)
- Scans are intentionally slow (avoid harming servers)
- Use `startNew=on` parameter to force new scan

---

## 4. testssl.sh (CLI Tool)

**What it tests:** Comprehensive SSL/TLS testing - protocols, ciphers, vulnerabilities, headers.

### Installation

```bash
# Clone repository
git clone --depth 1 https://github.com/drwetter/testssl.sh.git
cd testssl.sh

# Run scan
./testssl.sh plazaespana.info
```

### Basic Usage

```bash
# Quick scan (fast, less thorough)
./testssl.sh --fast plazaespana.info

# Comprehensive scan (recommended)
./testssl.sh plazaespana.info

# Wide output (better for terminals)
./testssl.sh --wide plazaespana.info

# HTML report
./testssl.sh --html plazaespana.info > report.html

# JSON output (for automation)
./testssl.sh --jsonfile results.json plazaespana.info

# Check specific vulnerabilities
./testssl.sh --vulnerable plazaespana.info

# Check HTTP headers
./testssl.sh --headers plazaespana.info
```

### Advanced Options

```bash
# Full scan with HTML report and color output
./testssl.sh --color 3 --html plazaespana.info | tee report.html

# Scan specific port
./testssl.sh plazaespana.info:443

# Use specific protocols only
./testssl.sh --protocols plazaespana.info

# Check cipher suites
./testssl.sh --cipher-per-proto plazaespana.info
```

### Requirements

- Bash 3.2 or higher
- OpenSSL (usually pre-installed)
- Standard Unix tools (dig/host/drill/nslookup, hexdump)

**Install missing tools on Debian/Ubuntu:**
```bash
sudo apt-get install bsdmainutils bind9-dnsutils
```

**On macOS:**
```bash
brew install bash openssl
```

---

## 5. webhint (CLI Tool)

**What it tests:** Accessibility, performance, PWA features, security, best practices, HTML/CSS validation.

### Installation

**Option 1: Use npx (No Installation)**
```bash
# Run directly with npx
npx hint https://plazaespana.info
```

**Option 2: Global Install**
```bash
# Install globally
npm install -g hint

# Run scan
hint https://plazaespana.info
```

**Option 3: Local Project**
```bash
# Create package.json if needed
npm init -y

# Install locally
npm install hint --save-dev

# Run scan
npx hint https://plazaespana.info
```

### Basic Usage

```bash
# Scan website
npx hint https://plazaespana.info

# Scan local file
npx hint ./public/index.html

# Use specific configuration
npx hint --config .hintrc https://plazaespana.info

# Output formats
npx hint --formatter html https://plazaespana.info > report.html
npx hint --formatter json https://plazaespana.info > results.json
```

### Configuration

Create `.hintrc` file to customize:

```json
{
  "connector": {
    "name": "puppeteer"
  },
  "formatters": ["html", "summary"],
  "hints": {
    "axe": "error",
    "content-type": "error",
    "highest-available-document-mode": "error",
    "meta-charset-utf-8": "error",
    "meta-viewport": "error",
    "no-friendly-error-pages": "error",
    "no-html-only-headers": "error"
  }
}
```

### Requirements

- Node.js 14.x or higher
- Chromium (automatically downloaded by Puppeteer)
- For Linux servers, you may need:
  ```bash
  # Debian/Ubuntu
  sudo apt-get install -y \
    libnss3 libatk1.0-0 libatk-bridge2.0-0 \
    libcups2 libdrm2 libxkbcommon0 libxcomposite1 \
    libxdamage1 libxrandr2 libgbm1 libasound2
  ```

---

## 6. Additional Useful Tools

### securityheaders.com CLI Alternative

Use `curl` to inspect headers directly:

```bash
# Get all HTTP headers
curl -I https://plazaespana.info

# Check specific security headers
curl -I https://plazaespana.info | grep -i "content-security-policy\|strict-transport-security\|x-frame-options\|x-content-type-options"

# Get headers in color (if curl supports it)
curl -sI https://plazaespana.info | grep --color=always -i "^[a-z-]*:"
```

### Check TLS version with OpenSSL

```bash
# Test TLS 1.3
openssl s_client -connect plazaespana.info:443 -tls1_3

# Test TLS 1.2
openssl s_client -connect plazaespana.info:443 -tls1_2

# Test TLS 1.1 (should fail on modern sites)
openssl s_client -connect plazaespana.info:443 -tls1_1

# Get certificate info
openssl s_client -connect plazaespana.info:443 -showcerts </dev/null 2>/dev/null | openssl x509 -inform pem -text
```

### Check HSTS Preload Eligibility

Visit: https://hstspreload.org/

Enter `plazaespana.info` to check if the site qualifies for HSTS preload list.

### Check Subresource Integrity

```bash
# Generate SRI hash for a local file
cat generator/assets/style.css | openssl dgst -sha384 -binary | openssl base64 -A

# Or use online tool: https://www.srihash.org/
```

---

## 7. Automated Scanning Script

Create a script to run all scans at once:

```bash
#!/bin/bash
# scan-security.sh

DOMAIN="${1:-plazaespana.info}"
OUTPUT_DIR="security-scans-$(date +%Y%m%d-%H%M%S)"

mkdir -p "$OUTPUT_DIR"
cd "$OUTPUT_DIR"

echo "Scanning $DOMAIN..."
echo "Results will be saved to: $OUTPUT_DIR"
echo ""

# 1. HTTP Headers
echo "[1/5] Checking HTTP headers..."
curl -I "https://$DOMAIN" > headers.txt 2>&1

# 2. testssl.sh (if available)
if command -v testssl.sh &> /dev/null; then
  echo "[2/5] Running testssl.sh (this may take 2-3 minutes)..."
  testssl.sh --html "$DOMAIN" > testssl-report.html 2>&1
else
  echo "[2/5] testssl.sh not found, skipping..."
fi

# 3. webhint (if node available)
if command -v npx &> /dev/null; then
  echo "[3/5] Running webhint..."
  npx hint "https://$DOMAIN" > webhint-results.txt 2>&1
else
  echo "[3/5] Node.js/npx not found, skipping webhint..."
fi

# 4. Mozilla Observatory API
echo "[4/5] Checking Mozilla Observatory..."
curl -X POST "https://observatory-api.mdn.mozilla.net/api/v2/scan?host=$DOMAIN" | python3 -m json.tool > observatory.json 2>&1

# 5. SSL Labs API (initiate scan)
echo "[5/5] Initiating SSL Labs scan..."
curl "https://api.ssllabs.com/api/v3/analyze?host=$DOMAIN" | python3 -m json.tool > ssllabs-status.json 2>&1

echo ""
echo "Scan complete! Results saved to: $OUTPUT_DIR"
echo ""
echo "Web-based scans (visit these URLs for full details):"
echo "  - Mozilla Observatory: https://developer.mozilla.org/en-US/observatory/analyze?host=$DOMAIN"
echo "  - SecurityHeaders: https://securityheaders.com/?q=$DOMAIN"
echo "  - SSL Labs: https://www.ssllabs.com/ssltest/analyze.html?d=$DOMAIN"
echo ""
```

Make it executable and run:

```bash
chmod +x scan-security.sh
./scan-security.sh plazaespana.info
```

---

## 8. Continuous Monitoring

### GitHub Actions

Add to `.github/workflows/security-scan.yml`:

```yaml
name: Security Scan

on:
  schedule:
    - cron: '0 0 1 * *'  # Monthly on 1st day
  workflow_dispatch:      # Manual trigger

jobs:
  scan:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@v4

      - name: Run Mozilla Observatory
        run: |
          curl -X POST "https://observatory-api.mdn.mozilla.net/api/v2/scan?host=plazaespana.info" \
            | tee observatory-results.json

      - name: Run testssl.sh
        run: |
          git clone --depth 1 https://github.com/drwetter/testssl.sh.git
          cd testssl.sh
          ./testssl.sh --jsonfile ../testssl-results.json plazaespana.info

      - name: Upload Results
        uses: actions/upload-artifact@v4
        with:
          name: security-scan-results
          path: |
            observatory-results.json
            testssl-results.json
```

### Local Cron Job

Add to crontab (`crontab -e`):

```bash
# Run security scan monthly
0 0 1 * * /path/to/scan-security.sh plazaespana.info >> /var/log/security-scans.log 2>&1
```

---

## 9. Interpreting Results

### Grade Scales

**Mozilla Observatory:**
- A+ (100+): Exceptional
- A (90-99): Excellent
- B (70-89): Good ← plazaespana.info (B+, 80)
- C (50-69): Fair
- D (25-49): Poor
- F (0-24): Very Poor

**SecurityHeaders.com:**
- A: Excellent ← plazaespana.info
- B: Good
- C: Fair
- D: Poor
- F: Very Poor

**SSL Labs:**
- A+ (>95): Exceptional
- A (80-95): Excellent
- B (65-79): Good
- C (50-64): Fair
- T: Certificate issues
- F (<50): Failed

### Common Issues & Fixes

**Missing Content-Security-Policy:**
```apache
# In .htaccess
Header set Content-Security-Policy "default-src 'self'; style-src 'self' 'unsafe-inline';"
```

**Missing HSTS:**
```apache
Header set Strict-Transport-Security "max-age=31536000; includeSubDomains; preload"
```

**Missing X-Frame-Options:**
```apache
Header set X-Frame-Options "DENY"
```

**Weak TLS Configuration:**
- Use Mozilla SSL Configuration Generator: https://ssl-config.mozilla.org/

---

## 10. References

- **Mozilla Observatory:** https://developer.mozilla.org/en-US/observatory/
- **SecurityHeaders:** https://securityheaders.com/
- **SSL Labs:** https://www.ssllabs.com/ssltest/
- **testssl.sh:** https://github.com/drwetter/testssl.sh
- **webhint:** https://webhint.io/
- **OWASP Secure Headers:** https://owasp.org/www-project-secure-headers/
- **MDN Web Security:** https://developer.mozilla.org/en-US/docs/Web/Security

---

**Last Updated:** 2025-10-24
