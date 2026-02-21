#!/usr/bin/env bash
# Build log-analyzer for FreeBSD (cross-compile) and macOS (native) using Zig
set -euo pipefail

echo "🔨 Building log-analyzer for FreeBSD and macOS using Zig..."
echo ""

# Check if Zig is installed
if ! command -v zig &> /dev/null; then
    echo "❌ Zig not found. Install with: brew install zig"
    exit 1
fi

# Always regenerate analyzer C source from current Janet code before compiling.
echo "📝 Generating C source with jpm..."
jpm build > /dev/null

# Janet source pinned to a known-good release.
JANET_VERSION="v1.41.2"
JANET_REPO="https://github.com/janet-lang/janet.git"
JANET_SRC="/tmp/janet-${JANET_VERSION}"
JANET_AMALG="/tmp/janet-single-threaded-${JANET_VERSION}.c"
JANETCONF="${JANET_SRC}/src/conf/janetconf.h"
JANETCONF_BACKUP=""

restore_janetconf() {
    if [ -n "${JANETCONF_BACKUP}" ] && [ -f "${JANETCONF_BACKUP}" ]; then
        mv "${JANETCONF_BACKUP}" "${JANETCONF}"
    fi
}
trap restore_janetconf EXIT

if [ ! -d "${JANET_SRC}/.git" ]; then
    echo "📥 Cloning Janet source (${JANET_VERSION})..."
    rm -rf "${JANET_SRC}"
    git clone --branch "${JANET_VERSION}" --depth=1 "${JANET_REPO}" "${JANET_SRC}"
fi

CURRENT_TAG="$(git -C "${JANET_SRC}" describe --tags --exact-match 2>/dev/null || true)"
if [ "${CURRENT_TAG}" != "${JANET_VERSION}" ]; then
    echo "🔄 Refreshing Janet source at ${JANET_VERSION}..."
    rm -rf "${JANET_SRC}"
    git clone --branch "${JANET_VERSION}" --depth=1 "${JANET_REPO}" "${JANET_SRC}"
fi

# Use properly built single-threaded Janet amalgamation
if [ ! -f "${JANET_AMALG}" ]; then
    echo "🔧 Building Janet amalgamation (single-threaded)..."
    cd "${JANET_SRC}"

    # Ensure single-threaded mode is enabled regardless of janetconf.h formatting.
    if ! grep -Eq '^[[:space:]]*#define[[:space:]]+JANET_SINGLE_THREADED([[:space:]]|$)' "${JANETCONF}"; then
        JANETCONF_BACKUP="${JANETCONF}.bak"
        cp "${JANETCONF}" "${JANETCONF_BACKUP}"
        cat >> "${JANETCONF}" <<'EOF'

#ifndef JANET_SINGLE_THREADED
#define JANET_SINGLE_THREADED
#endif
EOF
    fi

    # Build Janet (this generates build/c/janet.c with the config)
    make clean > /dev/null 2>&1
    make > /dev/null 2>&1

    # Copy the amalgamated source
    cp build/c/janet.c "${JANET_AMALG}"

    cd - > /dev/null
    echo "✓ Generated single-threaded Janet amalgamation (${JANET_VERSION})"
fi

echo "🎯 Compiling for FreeBSD/amd64..."
echo ""

# Use Zig to cross-compile
# -target x86_64-freebsd: Target FreeBSD on x86_64
# -Os: Optimize for size
zig cc \
    -lc \
    -target x86_64-freebsd \
    -Os \
    -I"${JANET_SRC}/src/include" \
    -I"${JANET_SRC}/src/core" \
    -I"${JANET_SRC}/src/conf" \
    -DJANET_SINGLE_THREADED=1 \
    build/log-analyzer.c \
    "${JANET_AMALG}" \
    -o build/log-analyzer-freebsd

if [ -f "build/log-analyzer-freebsd" ]; then
    echo ""
    echo "✅ FreeBSD build complete!"
    echo ""
    echo "Binary info:"
    file build/log-analyzer-freebsd 2>/dev/null || echo "  FreeBSD x86-64 executable"
    ls -lh build/log-analyzer-freebsd
else
    echo ""
    echo "❌ FreeBSD build failed"
    exit 1
fi

echo ""
echo "🎯 Compiling for macOS (native)..."
echo ""

# Detect native architecture
ARCH=$(uname -m)
if [ "$ARCH" = "arm64" ]; then
    TARGET="aarch64-macos"
    ARCH_LABEL="Apple Silicon (arm64)"
else
    TARGET="x86_64-macos"
    ARCH_LABEL="Intel (x86_64)"
fi

zig cc \
    -lc \
    -target "$TARGET" \
    -Os \
    -I"${JANET_SRC}/src/include" \
    -I"${JANET_SRC}/src/core" \
    -I"${JANET_SRC}/src/conf" \
    -DJANET_SINGLE_THREADED=1 \
    build/log-analyzer.c \
    "${JANET_AMALG}" \
    -o build/log-analyzer

if [ -f "build/log-analyzer" ]; then
    echo ""
    echo "✅ macOS build complete!"
    echo ""
    echo "Binary info:"
    file build/log-analyzer
    ls -lh build/log-analyzer
    echo ""
    echo "📦 Summary:"
    echo "  - FreeBSD: build/log-analyzer-freebsd (for NearlyFreeSpeech.NET)"
    echo "  - macOS:   build/log-analyzer ($ARCH_LABEL)"
    echo ""
    echo "To deploy FreeBSD binary:"
    echo "  scp build/log-analyzer-freebsd \$NFSN_USER@\$NFSN_HOST:/home/private/bin/log-analyzer"
else
    echo ""
    echo "❌ macOS build failed"
    exit 1
fi
