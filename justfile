# Default recipe (runs when you type 'just')
default:
    @just --list

# Build the site generator binary
build:
    go build -o build/buildsite ./cmd/buildsite

# Run all tests
test:
    go test ./...

# Run tests with coverage
test-coverage:
    go test -cover ./...

# Build for FreeBSD/amd64 (production)
freebsd:
    ./scripts/build-freebsd.sh

# Generate CSS with content hash
hash-css:
    ./scripts/hash-assets.sh

# Generate site with test data and serve locally
dev: build hash-css
    #!/usr/bin/env bash
    set -euo pipefail
    echo "🔨 Building site..."
    ./build/buildsite \
      -json-url https://datos.madrid.es/egob/catalogo/300107-0-agenda-actividades-eventos.json \
      -xml-url https://datos.madrid.es/egob/catalogo/300107-0-agenda-actividades-eventos.xml \
      -csv-url https://datos.madrid.es/egob/catalogo/300107-0-agenda-actividades-eventos.csv \
      -out-dir ./public \
      -data-dir ./data \
      -lat 40.42338 -lon -3.71217 -radius-km 0.35 \
      -timezone Europe/Madrid

    echo ""
    echo "✅ Site generated!"
    echo "📂 Files in ./public/"
    echo ""
    echo "🌐 Starting local server at http://localhost:8080"
    echo "   Press Ctrl+C to stop"
    echo ""

    cd public && python3 -m http.server 8080

# Quick dev server (assumes site already built)
serve:
    #!/usr/bin/env bash
    echo "🌐 Serving ./public at http://localhost:8080"
    echo "   Press Ctrl+C to stop"
    cd public && python3 -m http.server 8080

# Clean build artifacts
clean:
    rm -rf build/
    rm -rf public/
    rm -rf data/

# Format Go code
fmt:
    go fmt ./...

# Run Go linter
lint:
    go vet ./...

# Install dependencies (none for this project, but good to have)
deps:
    go mod download
    go mod verify

# Check for outdated dependencies
outdated:
    go list -u -m all

# Run integration tests
test-integration:
    go test -tags=integration ./cmd/buildsite
