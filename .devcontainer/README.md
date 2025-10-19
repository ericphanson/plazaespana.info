# Development Container for Plaza España Calendar

This devcontainer provides a complete Go development environment configured for cross-compiling to FreeBSD/amd64, which is required for deployment to NearlyFreeSpeech.NET.

## What's Included

- **Go 1.23** (latest stable)
- **Go development tools**: gopls, delve debugger, staticcheck, and more
- **FreeBSD cross-compilation support** (GOOS=freebsd GOARCH=amd64)
- **Git and common utilities**
- **Non-root user** (vscode) for security

## Quick Start

1. **Open in Claude Code with unrestricted permissions**:
   - Claude Code will automatically detect the devcontainer configuration
   - When prompted, allow Claude Code to run with unrestricted permissions

2. **Or manually open in VS Code**:
   - Install the "Dev Containers" extension
   - Press `Cmd+Shift+P` (Mac) or `Ctrl+Shift+P` (Windows/Linux)
   - Select "Dev Containers: Reopen in Container"

3. **Build for FreeBSD**:
   ```bash
   # From the workspace root
   ./scripts/build-freebsd.sh
   ```

   Or manually:
   ```bash
   GOOS=freebsd GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o build/buildsite ./cmd/buildsite
   ```

## Why This Configuration?

- **FreeBSD target**: NFSN runs FreeBSD on their servers
- **CGO disabled**: Ensures pure Go binary with no C dependencies for easier cross-compilation
- **Static binary**: The `-trimpath` and `-ldflags` flags create a smaller, portable binary
- **Unrestricted permissions**: Allows Claude Code to execute build commands, run tests, and manage files

## Verifying FreeBSD Support

After the container builds, verify cross-compilation works:

```bash
GOOS=freebsd GOARCH=amd64 go version
```

This should output without errors, confirming Go can target FreeBSD.

## Directory Structure

```
/workspace        # Your project root (mounted from host)
/go              # Go workspace (GOPATH)
/go/bin          # Installed Go tools
/go/pkg          # Downloaded dependencies
```

## Troubleshooting

**Container fails to build?**
- Check Docker is running
- Ensure you have internet access (to download the Go image)

**Can't build for FreeBSD?**
- Verify with: `go tool dist list | grep freebsd`
- Should show `freebsd/amd64` and other FreeBSD targets

**Permission issues?**
- The devcontainer runs as user `vscode` (UID 1000)
- Use `sudo` if you need root access inside the container
