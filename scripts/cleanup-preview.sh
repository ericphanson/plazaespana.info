#!/usr/bin/env bash
# Clean up preview deployment from NFSN
# Usage: cleanup-preview.sh PREVIEW_NAME
#   Example: cleanup-preview.sh PR5
#   Example: cleanup-preview.sh abc
set -euo pipefail

if [ $# -ne 1 ]; then
  echo "Usage: $0 PREVIEW_NAME"
  echo "  Example: $0 PR5"
  echo "  Example: $0 abc"
  exit 1
fi

PREVIEW_NAME="$1"
REMOTE_DIR="/home/public/previews/$PREVIEW_NAME"

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

echo "🧹 Cleaning up preview '$PREVIEW_NAME' from NearlyFreeSpeech.NET..."
echo "   Host: $NFSN_HOST"
echo "   User: $NFSN_USER"
echo "   Path: $REMOTE_DIR"
echo ""

# Check if preview directory exists
echo "🔍 Checking if preview exists..."
if ssh "$NFSN_USER@$NFSN_HOST" "test -d $REMOTE_DIR"; then
  # Remove preview directory
  echo "🗑️  Removing preview directory..."
  ssh "$NFSN_USER@$NFSN_HOST" "rm -rf $REMOTE_DIR"
else
  echo "ℹ️  Preview directory does not exist (may not have been created)"
  echo "   This is expected for Dependabot PRs or failed deployments"
fi

# Clean up empty parent directories (best effort, ignore failures)
echo "🧹 Cleaning up empty parent directories..."
ssh "$NFSN_USER@$NFSN_HOST" "rmdir /home/public/previews 2>/dev/null || true"

echo ""
echo "✅ Preview cleanup complete!"
echo ""
