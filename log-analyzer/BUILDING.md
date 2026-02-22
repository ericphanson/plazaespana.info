# Building Janet Projects for FreeBSD

This document explains how to properly build Janet projects with single-threaded mode for cross-compilation.

## The Problem

Janet's standard build includes pthread support, which causes issues when cross-compiling to FreeBSD using Zig because:
- Zig doesn't bundle FreeBSD's pthread headers
- The amalgamation needs to be built with the correct configuration **before** cross-compilation

## The Solution: Proper Amalgamation Build

### Step 1: Enable JANET_SINGLE_THREADED

Edit `janet/src/conf/janetconf.h` and uncomment:

```c
#define JANET_SINGLE_THREADED
```

### Step 2: Build Janet to Generate Amalgamation

```bash
cd /path/to/janet
make clean
make
```

This generates `build/c/janet.c` - the amalgamated source with single-threaded support baked in.

### Step 3: Cross-Compile with Zig

```bash
zig cc \
    -lc \
    -target x86_64-freebsd \
    -Os \
    -I. \
    -Ijanet/src/include \
    -Ijanet/src/core \
    -DJANET_BUILD=\"zig-cross\" \
    -DJANET_SINGLE_THREADED=1 \
    your-app.c \
    janet/build/c/janet.c \
    -o your-app-freebsd
```

## Key Insights

1. **Amalgamation must be built with config** - Don't just concatenate source files. Run `make` to generate `build/c/janet.c` with the configuration embedded.

2. **JANET_SINGLE_THREADED affects the build** - This flag must be set in `janetconf.h` **before** running `make`, not just during compilation.

3. **No pthread dependency** - When `JANET_SINGLE_THREADED` is properly enabled, pthread functions are compiled out via `#ifdef` guards.

## Verification

Check that the amalgamation was built correctly:

```bash
# Should show pthread guards are inactive
grep -c "pthread_mutex_t" janet/build/c/janet.c  # Will show ~7 (all in #ifdef blocks)

# Verify binary is FreeBSD
file your-app-freebsd
# Output: ELF 64-bit LSB executable, x86-64, ...FreeBSD-style

# Check dynamic dependencies (should only show libc, not libpthread)
# On FreeBSD: ldd your-app-freebsd
```

## Resources

- [Janet C API Configuration](https://janet-lang.org/capi/configuration.html)
- [Janet Embedding Guide](https://janet-lang.org/capi/embedding.html)
- [Janet GitHub - Cross-compiling](https://monzool.net/blog/2019/12/13/cross-compiling-janet-lang/)

## Our Implementation

See `build-freebsd-zig.sh` for the automated build script that:
1. Clones Janet source
2. Modifies `janetconf.h` to enable `JANET_SINGLE_THREADED`
3. Runs `make` to generate the proper amalgamation
4. Cross-compiles with Zig using the generated `janet.c`
