# Madrid Events Site Generator - Quick Commands
# Run 'just' to see all available commands

# Show this help message
default:
    @echo "Madrid Events Site Generator - Available Commands:"
    @echo ""
    @echo "🚀 Getting Started:"
    @echo "  just dev          - Build site and serve locally (http://localhost:8080)"
    @echo "  just test         - Run all tests"
    @echo ""
    @echo "🔨 Build Commands:"
    @echo "  just build        - Build binary for local use"
    @echo "  just freebsd      - Build for FreeBSD (for NFSN deployment)"
    @echo "  just hash-css     - Generate content-hashed CSS"
    @echo ""
    @echo "🧪 Testing:"
    @echo "  just test         - Run all tests"
    @echo "  just test-coverage - Run tests with coverage report"
    @echo ""
    @echo "🌐 Development:"
    @echo "  just serve        - Serve ./public (if already built)"
    @echo "  just kill         - Stop running dev server"
    @echo ""
    @echo "🧹 Maintenance:"
    @echo "  just clean        - Remove build artifacts"
    @echo "  just fmt          - Format Go code"
    @echo "  just lint         - Run Go linter"
    @echo ""
    @echo "📝 Configuration:"
    @echo "  just config       - Validate config.toml"
    @echo ""
    @echo "💡 Tips:"
    @echo "  - 'just dev' uses development mode (1hr cache, safe for rapid testing)"
    @echo "  - For production, add '-fetch-mode production' to cron command"
    @echo "  - See README.md for detailed documentation"

# Validate config.toml syntax and settings
config:
    @echo "🔍 Validating config.toml..."
    @./build/buildsite -config config.toml -validate || (echo "❌ Config validation failed" && exit 1)
    @echo "✅ Config is valid!"

# Build the site generator binary for local use
build:
    @echo "🔨 Building binary..."
    @go build -o build/buildsite ./cmd/buildsite
    @echo "✅ Built: build/buildsite"

# Run all tests (fast - uses cached results when possible)
test:
    @echo "🧪 Running tests..."
    @go test ./...

# Run tests with coverage report
test-coverage:
    @echo "🧪 Running tests with coverage..."
    @go test -cover ./...

# Build for FreeBSD/amd64 (for NearlyFreeSpeech.NET deployment)
freebsd:
    @echo "🔨 Cross-compiling for FreeBSD..."
    @./scripts/build-freebsd.sh
    @echo "✅ Built: build/buildsite (FreeBSD binary)"
    @ls -lh build/buildsite

# Generate CSS with content hash for cache busting
hash-css:
    @./scripts/hash-assets.sh

# 🚀 Build site and serve locally (MAIN COMMAND)
# Uses development mode: 1hr cache TTL, safe for rapid testing
dev: build hash-css
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
    echo "🌐 Starting local server at http://localhost:8080"
    echo "   Press Ctrl+C to stop"
    echo ""

    cd public && python3 -m http.server 8080

# Serve existing site (skip rebuild, faster startup)
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

# Clean all build artifacts and generated files
clean:
    @echo "🧹 Cleaning build artifacts..."
    @rm -rf build/ public/ data/
    @echo "✅ Cleaned: build/, public/, data/"

# Format all Go source code
fmt:
    @echo "✨ Formatting Go code..."
    @go fmt ./...
    @echo "✅ Code formatted"

# Run Go linter to check for issues
lint:
    @echo "🔍 Running linter..."
    @go vet ./...
    @echo "✅ No issues found"

# Download and verify Go module dependencies
deps:
    @echo "📦 Downloading dependencies..."
    @go mod download
    @go mod verify
    @echo "✅ Dependencies verified"

# Check for outdated Go module dependencies
outdated:
    @echo "🔍 Checking for outdated dependencies..."
    @go list -u -m all

# Run integration tests (if any)
test-integration:
    @echo "🧪 Running integration tests..."
    @go test -tags=integration ./cmd/buildsite
