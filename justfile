# Madrid Events Site Generator - Quick Commands
# Run 'just' to see all available commands

[private]
default:
    just --list

# Validate config.toml syntax and settings
config:
    @echo "🔍 Validating config.toml..."
    @./build/buildsite -config config.toml -validate || (echo "❌ Config validation failed" && exit 1)
    @echo "✅ Config is valid!"

# Build binary for local use
build:
    #!/usr/bin/env bash
    set -euo pipefail
    echo "🔨 Building binary..."
    GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
    cd generator && go build -ldflags="-X github.com/ericphanson/plazaespana.info/internal/version.GitCommit=$GIT_COMMIT" -o ../build/buildsite ./cmd/buildsite
    echo "✅ Built: build/buildsite (git: $GIT_COMMIT)"

# Run all tests
test:
    @echo "🧪 Running tests..."
    @cd generator && PLAZAESPANA_NO_API=1 go test ./...

# Run tests with coverage report
test-coverage:
    @echo "🧪 Running tests with coverage..."
    @cd generator && PLAZAESPANA_NO_API=1 go test -cover ./...

# Build for FreeBSD/amd64 (for NFSN deployment)
freebsd:
    @echo "🔨 Cross-compiling for FreeBSD..."
    @./scripts/build-freebsd.sh
    @echo "✅ Built: build/buildsite (FreeBSD binary)"
    @ls -lh build/buildsite

# Build log-analyzer FreeBSD binary (for NFSN deployment)
log-analyzer-freebsd:
    @echo "🔨 Building log-analyzer FreeBSD binary..."
    @cd log-analyzer && ./build-freebsd-zig.sh
    @echo "✅ Built: log-analyzer/build/log-analyzer-freebsd"
    @ls -lh log-analyzer/build/log-analyzer-freebsd

# Deploy files to NFSN (internal helper, assumes binary already built)
[private]
_deploy-files:
    #!/usr/bin/env bash
    set -euo pipefail

    # Check required environment variables
    if [ -z "${NFSN_HOST:-}" ]; then
        echo "❌ Error: NFSN_HOST environment variable not set"
        echo "   Example: export NFSN_HOST=ssh.phx.nearlyfreespeech.net"
        exit 1
    fi
    if [ -z "${NFSN_USER:-}" ]; then
        echo "❌ Error: NFSN_USER environment variable not set"
        echo "   Example: export NFSN_USER=username"
        exit 1
    fi

    echo "🚀 Deploying to NearlyFreeSpeech.NET..."
    echo "   Host: $NFSN_HOST"
    echo "   User: $NFSN_USER"
    echo ""

    # Create remote directories if needed
    echo "📁 Creating remote directories..."
    ssh "$NFSN_USER@$NFSN_HOST" '
        command -v python3 >/dev/null 2>&1 || { echo "python3 is required for log-analyzer publish pipeline"; exit 1; }
        command -v curl >/dev/null 2>&1 || { echo "curl is required for Bunny backup sync"; exit 1; }
        mkdir -p /home/private/bin /home/private/templates /home/private/data /home/private/log-analyzer-data /home/public/assets /home/public/analytics
    '

    # Upload new files with .new suffix (atomic swap later)
    echo "📤 Uploading binary..."
    scp build/buildsite "$NFSN_USER@$NFSN_HOST:/home/private/bin/buildsite.new"

    echo "📤 Uploading log-analyzer binary..."
    scp log-analyzer/build/log-analyzer-freebsd "$NFSN_USER@$NFSN_HOST:/home/private/bin/log-analyzer.new"

    echo "📤 Uploading config..."
    scp config.toml "$NFSN_USER@$NFSN_HOST:/home/private/config.toml.new"

    # Upload AEMET API key if present in environment
    if [ -n "${AEMET_API_KEY:-}" ]; then
        echo "📤 Uploading AEMET API key file..."
        echo -n "$AEMET_API_KEY" > build/aemet-api-key.txt
        chmod 600 build/aemet-api-key.txt
        scp build/aemet-api-key.txt "$NFSN_USER@$NFSN_HOST:/home/private/aemet-api-key.txt"
        rm build/aemet-api-key.txt
        echo "✅ API key uploaded to /home/private/aemet-api-key.txt"
    else
        echo "⚠️  AEMET_API_KEY not set - skipping API key upload (weather will be disabled)"
    fi

    echo "📤 Uploading templates..."
    scp generator/templates/index.tmpl.html "$NFSN_USER@$NFSN_HOST:/home/private/templates/index.tmpl.html.new"

    echo "📤 Uploading cron wrapper script..."
    scp ops/cron-generate.sh "$NFSN_USER@$NFSN_HOST:/home/private/bin/cron-generate.sh.new"

    echo "📤 Uploading log-analyzer cron wrapper script..."
    scp ops/log-analyzer-weekly.sh "$NFSN_USER@$NFSN_HOST:/home/private/bin/log-analyzer-weekly.sh.new"

    echo "📤 Uploading log-analyzer publish script..."
    scp ops/log-analyzer-publish.py "$NFSN_USER@$NFSN_HOST:/home/private/bin/log-analyzer-publish.py.new"

    echo "📤 Uploading hashed CSS and hash files..."
    scp public/assets/site.*.css public/assets/build-report.*.css "$NFSN_USER@$NFSN_HOST:/home/public/assets/"
    scp public/assets/css.hash public/assets/build-report-css.hash "$NFSN_USER@$NFSN_HOST:/home/public/assets/"

    echo "📤 Uploading weather icons..."
    ssh "$NFSN_USER@$NFSN_HOST" 'mkdir -p /home/public/assets/weather-icons'
    scp public/assets/weather-icons/*.png "$NFSN_USER@$NFSN_HOST:/home/public/assets/weather-icons/" 2>/dev/null || echo "⚠️  No weather icons found (optional)"

    echo "📤 Uploading .htaccess..."
    scp ops/htaccess "$NFSN_USER@$NFSN_HOST:/home/public/.htaccess"

    echo "📤 Uploading robots.txt..."
    scp ops/robots.txt "$NFSN_USER@$NFSN_HOST:/home/public/robots.txt"

    # Upload Bunny backup credentials/config if present in env
    if [ -n "${BUNNY_STORAGE_KEY:-}${BUNNY_STORAGE_ZONE:-}${BUNNY_STORAGE_ENDPOINT:-}${BUNNY_BASE_PATH:-}" ]; then
        if [ -z "${BUNNY_STORAGE_KEY:-}" ] || [ -z "${BUNNY_STORAGE_ZONE:-}" ] || [ -z "${BUNNY_STORAGE_ENDPOINT:-}" ]; then
            echo "❌ Error: If any Bunny variable is set, BUNNY_STORAGE_KEY, BUNNY_STORAGE_ZONE, and BUNNY_STORAGE_ENDPOINT are all required"
            exit 1
        fi

        echo "📤 Uploading Bunny backup config files..."
        echo -n "$BUNNY_STORAGE_KEY" > build/bunny-storage-key.txt
        echo -n "$BUNNY_STORAGE_ZONE" > build/bunny-storage-zone.txt
        echo -n "$BUNNY_STORAGE_ENDPOINT" > build/bunny-storage-endpoint.txt
        echo -n "${BUNNY_BASE_PATH:-analytics-backup/current}" > build/bunny-base-path.txt
        chmod 600 build/bunny-storage-key.txt build/bunny-storage-zone.txt build/bunny-storage-endpoint.txt build/bunny-base-path.txt

        scp build/bunny-storage-key.txt "$NFSN_USER@$NFSN_HOST:/home/private/bunny-storage-key.txt.new"
        scp build/bunny-storage-zone.txt "$NFSN_USER@$NFSN_HOST:/home/private/bunny-storage-zone.txt.new"
        scp build/bunny-storage-endpoint.txt "$NFSN_USER@$NFSN_HOST:/home/private/bunny-storage-endpoint.txt.new"
        scp build/bunny-base-path.txt "$NFSN_USER@$NFSN_HOST:/home/private/bunny-base-path.txt.new"

        rm -f build/bunny-storage-key.txt build/bunny-storage-zone.txt build/bunny-storage-endpoint.txt build/bunny-base-path.txt
    else
        if ssh "$NFSN_USER@$NFSN_HOST" '[ -f /home/private/bunny-storage-key.txt ] && [ -f /home/private/bunny-storage-zone.txt ] && [ -f /home/private/bunny-storage-endpoint.txt ]'; then
            echo "⚠️  Bunny backup env vars not set - keeping existing Bunny config on server"
        else
            echo "❌ Bunny backup config missing on server and BUNNY_* env vars not provided."
            echo "   Set BUNNY_STORAGE_KEY, BUNNY_STORAGE_ZONE, BUNNY_STORAGE_ENDPOINT (and optional BUNNY_BASE_PATH) before deploy."
            exit 1
        fi
    fi

    # Atomically swap new files into place
    echo "🔄 Activating new files..."
    ssh "$NFSN_USER@$NFSN_HOST" '
        mv /home/private/bin/buildsite.new /home/private/bin/buildsite &&
        mv /home/private/bin/log-analyzer.new /home/private/bin/log-analyzer &&
        mv /home/private/bin/cron-generate.sh.new /home/private/bin/cron-generate.sh &&
        mv /home/private/bin/log-analyzer-weekly.sh.new /home/private/bin/log-analyzer-weekly.sh &&
        mv /home/private/bin/log-analyzer-publish.py.new /home/private/bin/log-analyzer-publish.py &&
        mv /home/private/config.toml.new /home/private/config.toml &&
        mv /home/private/templates/index.tmpl.html.new /home/private/templates/index.tmpl.html &&
        chmod +x /home/private/bin/buildsite /home/private/bin/log-analyzer /home/private/bin/cron-generate.sh /home/private/bin/log-analyzer-weekly.sh /home/private/bin/log-analyzer-publish.py
    '

    # Promote Bunny files if staged in this deploy
    ssh "$NFSN_USER@$NFSN_HOST" '
        if [ -f /home/private/bunny-storage-key.txt.new ]; then mv /home/private/bunny-storage-key.txt.new /home/private/bunny-storage-key.txt; fi
        if [ -f /home/private/bunny-storage-zone.txt.new ]; then mv /home/private/bunny-storage-zone.txt.new /home/private/bunny-storage-zone.txt; fi
        if [ -f /home/private/bunny-storage-endpoint.txt.new ]; then mv /home/private/bunny-storage-endpoint.txt.new /home/private/bunny-storage-endpoint.txt; fi
        if [ -f /home/private/bunny-base-path.txt.new ]; then mv /home/private/bunny-base-path.txt.new /home/private/bunny-base-path.txt; fi
        if [ -f /home/private/bunny-storage-key.txt ]; then chmod 600 /home/private/bunny-storage-key.txt; fi
        if [ -f /home/private/bunny-storage-zone.txt ]; then chmod 600 /home/private/bunny-storage-zone.txt; fi
        if [ -f /home/private/bunny-storage-endpoint.txt ]; then chmod 600 /home/private/bunny-storage-endpoint.txt; fi
        if [ -f /home/private/bunny-base-path.txt ]; then chmod 600 /home/private/bunny-base-path.txt; fi
    '

    # Run buildsite to regenerate the site
    echo "🔨 Regenerating site on server..."
    ssh "$NFSN_USER@$NFSN_HOST" '/home/private/bin/buildsite -config /home/private/config.toml -out-dir /home/public -data-dir /home/private/data -template-path /home/private/templates/index.tmpl.html -fetch-mode production'

    # Clean up old CSS files (keep only the latest of each type)
    echo "🧹 Cleaning up old CSS files..."
    ssh "$NFSN_USER@$NFSN_HOST" 'cd /home/public/assets && ls -t site.*.css 2>/dev/null | tail -n +2 | xargs -r rm -f || true'
    ssh "$NFSN_USER@$NFSN_HOST" 'cd /home/public/assets && ls -t build-report.*.css 2>/dev/null | tail -n +2 | xargs -r rm -f || true'

    echo ""
    echo "✅ Deployment complete!"
    echo ""
    echo "📝 Next steps:"
    echo "   1. Verify site at your NFSN URL"
    echo "   2. Setup cron jobs in NFSN web UI:"
    echo "      a) Site generation:"
    echo "         Command: /home/private/bin/cron-generate.sh"
    echo "         Schedule: Every hour"
    echo "         Note: Logs to /home/logs/generate.log, emails only on errors"
    echo "      b) Analytics snapshot (aggregate JSON):"
    echo "         Command: /home/private/bin/log-analyzer-weekly.sh"
    echo "         Schedule: 15 1 * * * (daily at 01:15)"
    echo "         Note: Logs to /home/logs/log-analyzer.log; serves report at /analytics/report.html"

# Deploy to NearlyFreeSpeech.NET (requires NFSN_HOST and NFSN_USER env vars)
deploy: freebsd log-analyzer-freebsd hash-css _deploy-files

# Deploy to NFSN (for CI - assumes binary already built and CSS hashed)
deploy-only: _deploy-files

# Check public analytics freshness via manifest.json
check-analytics-stale MAX_AGE_DAYS="3":
    python3 scripts/check-analytics-staleness.py --max-age-days {{MAX_AGE_DAYS}}

# Generate content-hashed CSS for cache busting
hash-css:
    @echo "🧹 Cleaning up old CSS files..."
    @rm -f public/assets/site.*.css public/assets/build-report.*.css
    @./scripts/hash-assets.sh

# Build and generate site (no server)
generate: build hash-css
    #!/usr/bin/env bash
    set -euo pipefail
    echo ""
    echo "🔨 Building Madrid Events site..."
    echo "   Mode: Development (1hr cache, 5s delays)"
    echo "   Config: config.toml"
    echo ""

    ./build/buildsite -config config.toml

    echo ""
    echo "✅ Site generated successfully!"
    echo ""
    echo "📂 Output files:"
    echo "   ./public/index.html  - Main event listing"
    echo "   ./public/events.json - JSON API"
    echo "   ./data/request-audit.json - HTTP request log"
    echo ""

# Build site and serve locally at :8080
dev: generate serve

# Serve existing ./public at :8080 (skip rebuild)
serve:
    #!/usr/bin/env bash
    if [ ! -d "public" ]; then
        echo "❌ ./public/ not found. Run 'just dev' first to build the site."
        exit 1
    fi
    echo "🌐 Serving ./public at http://localhost:8080"
    echo "   Press Ctrl+C to stop"
    cd public && python3 -m http.server 8080

# Stop the development server
kill:
    #!/usr/bin/env bash
    pkill -f "python3 -m http.server 8080" && echo "✅ Server stopped" || echo "ℹ️  No server running"

# Remove all build artifacts and generated files
clean:
    @echo "🧹 Cleaning build artifacts..."
    @rm -rf build/ public/ data/
    @echo "✅ Cleaned: build/, public/, data/"

# Format all Go source code
fmt:
    @echo "✨ Formatting Go code..."
    @cd generator && go fmt ./...
    @echo "✅ Code formatted"

# Check if code is properly formatted (for CI)
fmt-check:
    #!/usr/bin/env bash
    set -euo pipefail
    echo "🔍 Checking code formatting..."
    FILES=$(cd generator && gofmt -l .)
    if [ -n "$FILES" ]; then
        echo "❌ The following files are not formatted:"
        echo "$FILES"
        exit 1
    fi
    echo "✅ All files properly formatted"

# Run Go linter (go vet)
lint:
    @echo "🔍 Running linter..."
    @cd generator && go vet ./...
    @echo "✅ No issues found"

# Download and verify Go module dependencies
deps:
    @echo "📦 Downloading dependencies..."
    @cd generator && go mod download
    @cd generator && go mod verify
    @echo "✅ Dependencies verified"

# Check for outdated Go module dependencies
outdated:
    @echo "🔍 Checking for outdated dependencies..."
    @cd generator && go list -u -m all

# Run integration tests
test-integration:
    @echo "🧪 Running integration tests..."
    @echo "📦 Installing html-validate (if needed)..."
    @npm install --no-save 2>&1 | grep -v "^up to date" || true
    @cd generator && PLAZAESPANA_NO_API=1 go test -tags=integration ./cmd/buildsite -v

# Fetch test fixtures from upstream APIs (requires AEMET_API_KEY for weather data)
fetch-fixtures:
    @echo "📥 Fetching test fixtures..."
    @./scripts/fetch-fixtures.sh
    @echo "✅ Fixtures updated in generator/testdata/fixtures/"

# Build site for preview deployment with custom base path
# Usage: just preview-build PR5
# Usage: just preview-build abc
preview-build PREVIEW:
    #!/usr/bin/env bash
    set -euo pipefail
    echo "🔨 Building preview: {{PREVIEW}}"
    echo "   Base path: /previews/{{PREVIEW}}"
    echo ""

    # Build binary with git hash
    GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
    cd generator && go build -ldflags="-X github.com/ericphanson/plazaespana.info/internal/version.GitCommit=$GIT_COMMIT" -o ../build/buildsite ./cmd/buildsite
    cd ..

    # Hash CSS
    ./scripts/hash-assets.sh

    # Generate site with preview base path
    ./build/buildsite \
      -config config.toml \
      -base-path /previews/{{PREVIEW}}

    echo ""
    echo "✅ Preview built successfully!"
    echo "   Files in ./public/ are ready for deployment"
    echo ""

# Deploy preview to NFSN (requires NFSN_HOST and NFSN_USER env vars, requires SSH key)
# Usage: just preview-deploy PR5
# Usage: just preview-deploy abc
preview-deploy PREVIEW: (preview-build PREVIEW)
    @./scripts/deploy-preview.sh {{PREVIEW}}

# Clean up preview from NFSN (requires NFSN_HOST and NFSN_USER env vars, requires SSH key)
# Usage: just preview-cleanup PR5
# Usage: just preview-cleanup abc
preview-cleanup PREVIEW:
    @./scripts/cleanup-preview.sh {{PREVIEW}}

# Run all quality scans (links, HTML validation, performance)
# Usage: just scan [URL]
# Examples:
#   just scan                        # Scan localhost:8080 (default)
#   just scan plazaespana.info       # Scan production (https:// added automatically)
# Note: Individual scans may fail, but all will run. Use individual targets to check specific exit codes.
scan URL="http://localhost:8080":
    @just scan-links "{{URL}}" || true
    @just scan-html "{{URL}}" || true
    @just scan-performance "{{URL}}" || true
    @echo ""
    @echo "✅ All scans complete!"
    @echo ""
    @echo "📊 Results summary:"
    @echo "   Links:       scan-results/links.txt"
    @echo "   HTML:        scan-results/html-validation.txt"
    @echo "   Performance: scan-results/lighthouse.report.html"
    @echo ""
    @echo "See docs/scanning.md for interpretation guide"

# Check for broken links and missing assets
# Exits with code 1 if broken links are found, 0 if all links are OK
scan-links URL="http://localhost:8080":
    #!/usr/bin/env bash
    set -euo pipefail
    mkdir -p scan-results
    echo "🔍 [1/3] Checking links and assets..."

    # Add https:// if URL doesn't start with http:// or https://
    SCAN_URL="{{URL}}"
    if [[ ! "$SCAN_URL" =~ ^https?:// ]]; then
        SCAN_URL="https://$SCAN_URL"
    fi

    # Normalize URL (remove trailing slash, will add back for directory URLs)
    SCAN_URL="${SCAN_URL%/}"

    echo "   Target: $SCAN_URL"

    if ! command -v npx &> /dev/null; then
        echo "❌ Error: npx not found. Install Node.js first."
        exit 1
    fi

    # Add trailing slash for directory URLs to avoid 301 redirects
    # broken-link-checker exits with non-zero if broken links are found
    npx broken-link-checker "$SCAN_URL/" \
        --recursive \
        --ordered \
        --exclude-external 2>&1 | tee scan-results/links.txt
    LINK_EXIT=$?

    if [ $LINK_EXIT -eq 0 ]; then
        echo "✅ Link check complete - no broken links"
    else
        echo "❌ Link check complete - broken links found"
    fi

    exit $LINK_EXIT

# Run Lighthouse performance audit
scan-performance URL="http://localhost:8080":
    #!/usr/bin/env bash
    set -euo pipefail
    mkdir -p scan-results
    echo "🔍 [3/3] Running performance audit..."

    # Add https:// if URL doesn't start with http:// or https://
    SCAN_URL="{{URL}}"
    if [[ ! "$SCAN_URL" =~ ^https?:// ]]; then
        SCAN_URL="https://$SCAN_URL"
    fi

    # Normalize URL (remove trailing slash, will add back for directory URLs)
    SCAN_URL="${SCAN_URL%/}"

    echo "   Target: $SCAN_URL"

    if ! command -v npx &> /dev/null; then
        echo "❌ Error: npx not found. Install Node.js first."
        exit 1
    fi

    # Add trailing slash for directory URLs to avoid 301 redirects
    npx lighthouse "$SCAN_URL/" \
        --output=html \
        --output=json \
        --output-path=scan-results/lighthouse \
        --preset=desktop \
        --quiet \
        --chrome-flags="--headless" || true
    echo "✅ Performance audit complete"
    echo "   Report: scan-results/lighthouse.report.html"

# Validate HTML
# Exits with code 1 if validation errors are found, 0 if HTML is valid
scan-html URL="http://localhost:8080":
    #!/usr/bin/env bash
    set -euo pipefail
    mkdir -p scan-results
    echo "🔍 [2/3] Validating HTML..."

    # Add https:// if URL doesn't start with http:// or https://
    SCAN_URL="{{URL}}"
    if [[ ! "$SCAN_URL" =~ ^https?:// ]]; then
        SCAN_URL="https://$SCAN_URL"
    fi

    # Remove trailing slash for consistent URL building
    SCAN_URL="${SCAN_URL%/}"

    echo "   Target: $SCAN_URL"

    if ! command -v npx &> /dev/null; then
        echo "❌ Error: npx not found. Install Node.js first."
        exit 1
    fi

    # Fetch main page (add trailing slash to avoid 301 redirects on directory URLs)
    echo "   Fetching index.html..."
    if ! curl -sS -L "$SCAN_URL/" > scan-results/index.html; then
        echo "❌ Failed to fetch $SCAN_URL/" | tee scan-results/html-validation.txt
        echo "❌ HTML validation complete - fetch failed"
        exit 1
    fi

    # Fetch build report
    echo "   Fetching build-report.html..."
    if ! curl -sS "$SCAN_URL/build-report.html" > scan-results/build-report.html; then
        echo "⚠️  Failed to fetch $SCAN_URL/build-report.html (skipping)"
    fi

    # Validate only our HTML files (not lighthouse report which we don't control)
    npx html-validate scan-results/index.html scan-results/build-report.html 2>&1 | tee scan-results/html-validation.txt
    HTML_EXIT=$?

    if [ $HTML_EXIT -eq 0 ]; then
        echo "✅ HTML validation complete - no errors"
    else
        echo "❌ HTML validation complete - errors found"
    fi

    exit $HTML_EXIT
