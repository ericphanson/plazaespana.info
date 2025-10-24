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
    @cd generator && go test ./...

# Run tests with coverage report
test-coverage:
    @echo "🧪 Running tests with coverage..."
    @cd generator && go test -cover ./...

# Build for FreeBSD/amd64 (for NFSN deployment)
freebsd:
    @echo "🔨 Cross-compiling for FreeBSD..."
    @./scripts/build-freebsd.sh
    @echo "✅ Built: build/buildsite (FreeBSD binary)"
    @ls -lh build/buildsite

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
    ssh "$NFSN_USER@$NFSN_HOST" 'mkdir -p /home/private/bin /home/private/templates /home/private/data /home/public/assets /home/public/stats'

    # Upload new files with .new suffix (atomic swap later)
    echo "📤 Uploading binary..."
    scp build/buildsite "$NFSN_USER@$NFSN_HOST:/home/private/bin/buildsite.new"

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

    echo "📤 Uploading AWStats config..."
    scp ops/awstats.conf "$NFSN_USER@$NFSN_HOST:/home/private/awstats.conf"

    echo "📤 Uploading AWStats weekly script..."
    scp ops/awstats-weekly.sh "$NFSN_USER@$NFSN_HOST:/home/private/bin/awstats-weekly.sh.new"

    echo "📤 Uploading AWStats stats directory htaccess..."
    scp ops/stats.htaccess "$NFSN_USER@$NFSN_HOST:/home/public/stats/.htaccess"

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

    # Atomically swap new files into place
    echo "🔄 Activating new files..."
    ssh "$NFSN_USER@$NFSN_HOST" 'mv /home/private/bin/buildsite.new /home/private/bin/buildsite && mv /home/private/bin/cron-generate.sh.new /home/private/bin/cron-generate.sh && mv /home/private/bin/awstats-weekly.sh.new /home/private/bin/awstats-weekly.sh && mv /home/private/config.toml.new /home/private/config.toml && mv /home/private/templates/index.tmpl.html.new /home/private/templates/index.tmpl.html && chmod +x /home/private/bin/buildsite /home/private/bin/cron-generate.sh /home/private/bin/awstats-weekly.sh'

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
    echo "      b) AWStats weekly rollup:"
    echo "         Command: /home/private/bin/awstats-weekly.sh"
    echo "         Schedule: 0 1 * * 0 (Sunday 1 AM)"
    echo "         Note: Logs to /home/logs/awstats.log"
    echo "   3. Setup Basic Auth for /stats/:"
    echo "      SSH to NFSN and run: htpasswd -c /home/private/.htpasswd username"
    echo "      Set permissions: chmod 600 /home/private/.htpasswd && chmod 711 /home/private"

# Deploy to NearlyFreeSpeech.NET (requires NFSN_HOST and NFSN_USER env vars)
deploy: freebsd hash-css _deploy-files

# Deploy to NFSN (for CI - assumes binary already built and CSS hashed)
deploy-only: _deploy-files

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
    @cd generator && go test -tags=integration ./cmd/buildsite -v

# Fetch test fixtures from upstream APIs (requires AEMET_API_KEY for weather data)
fetch-fixtures:
    @echo "📥 Fetching test fixtures..."
    @./scripts/fetch-fixtures.sh
    @echo "✅ Fixtures updated in generator/testdata/fixtures/"

# Fetch new AWStats statistics archives and update/create PR (requires NFSN_HOST and NFSN_USER env vars)
fetch-stats-archives:
    @echo "📊 Fetching AWStats database archives..."
    @./scripts/fetch-stats-archives.sh

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
scan URL="http://localhost:8080": (scan-links URL) (scan-html URL) (scan-performance URL)
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

    echo "   Target: $SCAN_URL"

    if ! command -v npx &> /dev/null; then
        echo "❌ Error: npx not found. Install Node.js first."
        exit 1
    fi

    npx broken-link-checker "$SCAN_URL" \
        --recursive \
        --ordered \
        --exclude-external 2>&1 | tee scan-results/links.txt || true
    echo "✅ Link check complete"

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

    echo "   Target: $SCAN_URL"

    if ! command -v npx &> /dev/null; then
        echo "❌ Error: npx not found. Install Node.js first."
        exit 1
    fi

    npx lighthouse "$SCAN_URL" \
        --output=html \
        --output=json \
        --output-path=scan-results/lighthouse \
        --preset=desktop \
        --quiet \
        --chrome-flags="--headless" || true
    echo "✅ Performance audit complete"
    echo "   Report: scan-results/lighthouse.report.html"

# Validate HTML
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

    # Fetch main page
    echo "   Fetching index.html..."
    if ! curl -sS "$SCAN_URL" > scan-results/index.html; then
        echo "❌ Failed to fetch $SCAN_URL" | tee scan-results/html-validation.txt
        echo "✅ HTML validation complete (skipped - fetch failed)"
        exit 0
    fi

    # Fetch build report
    echo "   Fetching build-report.html..."
    if ! curl -sS "$SCAN_URL/build-report.html" > scan-results/build-report.html; then
        echo "⚠️  Failed to fetch $SCAN_URL/build-report.html (skipping)"
    fi

    # Validate all fetched HTML files
    npx html-validate scan-results/*.html 2>&1 | tee scan-results/html-validation.txt || true
    echo "✅ HTML validation complete"
