#!/usr/bin/env bash
# Deploy preview build to NFSN subdirectory
# Usage: deploy-preview.sh PREVIEW_NAME
#   Example: deploy-preview.sh PR5
#   Example: deploy-preview.sh abc
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

echo "🚀 Deploying preview '$PREVIEW_NAME' to NearlyFreeSpeech.NET..."
echo "   Host: $NFSN_HOST"
echo "   User: $NFSN_USER"
echo "   Path: $REMOTE_DIR"
echo ""

# Create remote directory if needed
echo "📁 Creating remote directory..."
ssh "$NFSN_USER@$NFSN_HOST" "mkdir -p $REMOTE_DIR/assets"

# Upload static files (HTML, CSS, JSON)
echo "📤 Uploading preview files..."
scp public/index.html "$NFSN_USER@$NFSN_HOST:$REMOTE_DIR/index.html"
scp public/build-report.html "$NFSN_USER@$NFSN_HOST:$REMOTE_DIR/build-report.html"
scp public/events.json "$NFSN_USER@$NFSN_HOST:$REMOTE_DIR/events.json"

# Upload hashed CSS files
echo "📤 Uploading CSS assets..."
scp public/assets/site.*.css "$NFSN_USER@$NFSN_HOST:$REMOTE_DIR/assets/"
scp public/assets/build-report.*.css "$NFSN_USER@$NFSN_HOST:$REMOTE_DIR/assets/"

echo ""
echo "✅ Preview deployed successfully!"
echo ""
echo "🌐 Preview URL: https://plazaespana.info/previews/$PREVIEW_NAME/"
echo ""
