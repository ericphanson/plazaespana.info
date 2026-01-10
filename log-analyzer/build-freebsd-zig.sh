#!/usr/bin/env bash
# Cross-compile log-analyzer for FreeBSD using Zig
set -euo pipefail

echo "🔨 Cross-compiling log-analyzer for FreeBSD using Zig..."
echo ""

# Check if Zig is installed
if ! command -v zig &> /dev/null; then
    echo "❌ Zig not found. Install with: brew install zig"
    exit 1
fi

# Check if we have the generated C file
if [ ! -f "build/log-analyzer.c" ]; then
    echo "📝 Generating C source with jpm..."
    jpm build
fi

# Check if we have Janet source
JANET_SRC="/tmp/janet-temp"
if [ ! -d "$JANET_SRC" ]; then
    echo "📥 Cloning Janet source..."
    git clone --depth=1 https://github.com/janet-lang/janet.git "$JANET_SRC"
fi

# Use properly built single-threaded Janet amalgamation
JANET_AMALG="/tmp/janet-single-threaded.c"
if [ ! -f "$JANET_AMALG" ]; then
    echo "🔧 Building Janet amalgamation (single-threaded)..."
    cd "$JANET_SRC"

    # Enable JANET_SINGLE_THREADED in janetconf.h
    sed -i.bak 's|/\* #define JANET_SINGLE_THREADED \*/|#define JANET_SINGLE_THREADED|' src/conf/janetconf.h

    # Build Janet (this generates build/c/janet.c with the config)
    make clean > /dev/null 2>&1
    make > /dev/null 2>&1

    # Copy the amalgamated source
    cp build/c/janet.c "$JANET_AMALG"

    # Restore original janetconf.h
    mv src/conf/janetconf.h.bak src/conf/janetconf.h

    cd - > /dev/null
    echo "✓ Generated single-threaded Janet amalgamation"
fi

echo "🎯 Compiling for FreeBSD/amd64..."
echo ""

# Use Zig to cross-compile
# -target x86_64-freebsd: Target FreeBSD on x86_64
# -Os: Optimize for size
# Use local janetconf.h with JANET_SINGLE_THREADED defined
zig cc \
    -lc \
    -target x86_64-freebsd \
    -Os \
    -I. \
    -I"$JANET_SRC/src/include" \
    -I"$JANET_SRC/src/core" \
    -DJANET_BUILD=\"zig-cross\" \
    -DJANET_SINGLE_THREADED=1 \
    build/log-analyzer.c \
    "$JANET_AMALG" \
    -o build/log-analyzer-freebsd

if [ -f "build/log-analyzer-freebsd" ]; then
    echo ""
    echo "✅ Build complete!"
    echo ""
    echo "Binary info:"
    file build/log-analyzer-freebsd 2>/dev/null || echo "  FreeBSD x86-64 executable"
    ls -lh build/log-analyzer-freebsd
    echo ""
    echo "📦 Ready for deployment to NearlyFreeSpeech.NET"
    echo ""
    echo "To deploy:"
    echo "  scp build/log-analyzer-freebsd \$NFSN_USER@\$NFSN_HOST:/home/private/bin/log-analyzer"
else
    echo ""
    echo "❌ Build failed"
    exit 1
fi
