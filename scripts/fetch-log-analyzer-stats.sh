#!/bin/bash
# Sync log-analyzer aggregate stats from server to git repository
set -euo pipefail

for cmd in ssh scp gh git jq grep; do
    if ! command -v "$cmd" >/dev/null 2>&1; then
        echo "❌ Error: Required command '$cmd' not found in PATH"
        exit 1
    fi
done

if [ -z "${NFSN_HOST:-}" ] || [ -z "${NFSN_USER:-}" ]; then
    echo "❌ Error: NFSN_HOST and NFSN_USER environment variables required"
    exit 1
fi

if ! gh auth status >/dev/null 2>&1; then
    echo "❌ Error: GitHub CLI (gh) is not authenticated"
    echo "   Run: gh auth login (or set GH_TOKEN)"
    exit 1
fi

REMOTE_DIR="/home/private/log-analyzer-data"
LOCAL_DIR="log-analyzer-data"
CANONICAL_BRANCH="log-analyzer-data"

if [[ -n $(git status --porcelain) ]]; then
    echo "❌ Error: Working directory has uncommitted changes"
    git status --short
    exit 1
fi

CURRENT_BRANCH="$(git rev-parse --abbrev-ref HEAD)"
if [ "$CURRENT_BRANCH" != "main" ]; then
    echo "Switching from $CURRENT_BRANCH to main..."
    git checkout main
fi

git pull origin main
git fetch origin main
git checkout -B "$CANONICAL_BRANCH" origin/main

mkdir -p "$LOCAL_DIR"
rm -f "$LOCAL_DIR"/*.json "$LOCAL_DIR"/report.html "$LOCAL_DIR"/README.md

echo "📊 Syncing log-analyzer stats from $NFSN_HOST:$REMOTE_DIR"
if ! scp -q "$NFSN_USER@$NFSN_HOST:$REMOTE_DIR/*.json" "$LOCAL_DIR/"; then
    echo "⚠️  No JSON files found on server at $REMOTE_DIR"
    git checkout main
    exit 0
fi

# report.html is optional for sync
scp -q "$NFSN_USER@$NFSN_HOST:$REMOTE_DIR/report.html" "$LOCAL_DIR/" 2>/dev/null || true

# Restore README tracked in main branch.
git checkout origin/main -- "$LOCAL_DIR/README.md"

FILE_COUNT=$(ls -1 "$LOCAL_DIR"/*.json 2>/dev/null | wc -l)
echo "✅ Downloaded $FILE_COUNT JSON file(s)"

echo "🔒 Running privacy checks..."

# Check for plain IPv4 addresses.
IP_CHECK_FILES=("$LOCAL_DIR"/*.json)
if [ -f "$LOCAL_DIR/report.html" ]; then
    IP_CHECK_FILES+=("$LOCAL_DIR/report.html")
fi

if grep -nE '\b([0-9]{1,3}\.){3}[0-9]{1,3}\b' "${IP_CHECK_FILES[@]}" >/dev/null 2>&1; then
    echo "❌ Privacy check failed: potential IPv4 address found in synced outputs"
    grep -nE '\b([0-9]{1,3}\.){3}[0-9]{1,3}\b' "${IP_CHECK_FILES[@]}" || true
    git checkout main
    exit 1
fi

# Check that path values do not contain raw query strings.
if grep -nE '"path":[[:space:]]*"[^"]*\?' "$LOCAL_DIR"/*.json >/dev/null 2>&1; then
    echo "❌ Privacy check failed: top path includes query string"
    grep -nE '"path":[[:space:]]*"[^"]*\?' "$LOCAL_DIR"/*.json || true
    git checkout main
    exit 1
fi

echo "✅ Privacy checks passed"

if [[ -z $(git status --porcelain "$LOCAL_DIR") ]]; then
    echo "✅ No changes detected - stats are already in sync"
    git checkout main
    exit 0
fi

echo ""
echo "📝 Changes detected:"
git status --short "$LOCAL_DIR"
echo ""

TOTAL_REQUESTS="$(jq -r '.total_requests // 0' "$LOCAL_DIR/lifetime.json" 2>/dev/null || echo 0)"
TOTAL_UNIQUE="$(jq -r '.total_unique_visitors_estimate // 0' "$LOCAL_DIR/lifetime.json" 2>/dev/null || echo 0)"
GENERATED_AT="$(jq -r '.generated_at // "unknown"' "$LOCAL_DIR/lifetime.json" 2>/dev/null || echo unknown)"

git add "$LOCAL_DIR"
git commit -m "chore: sync log-analyzer aggregate stats

Updated aggregated analytics from production.
JSON files: $FILE_COUNT
Generated at: $GENERATED_AT

Contains aggregate/anonymized data only (no raw logs, no IPs)."

PR_NUMBER="$(gh pr list --head "$CANONICAL_BRANCH" --state open --json number --jq '.[0].number' || echo "")"

if [ -n "$PR_NUMBER" ]; then
    echo "📝 Updating existing PR #$PR_NUMBER with force push..."
    git push --force origin "$CANONICAL_BRANCH"
    echo "✅ Updated PR #$PR_NUMBER"
else
    echo "📝 Creating new pull request..."
    git push --force origin "$CANONICAL_BRANCH"

    gh pr create \
        --title "Sync log-analyzer aggregate stats" \
        --body "$(cat <<EOFPR
Automated PR to sync log-analyzer aggregate statistics from production.

## Latest snapshot

- **Generated at:** $GENERATED_AT
- **Total requests:** $TOTAL_REQUESTS
- **Total unique visitors (estimate):** $TOTAL_UNIQUE
- **JSON files:** $FILE_COUNT

Data source: \`/home/private/log-analyzer-data\` on NFSN.

Privacy checks performed in sync script:
- no IPv4 address patterns
- no query-string paths in top-path output

EOFPR
)" \
        --head "$CANONICAL_BRANCH" \
        --label "automated" \
        --label "analytics"

    echo "✅ Pull request created from branch: $CANONICAL_BRANCH"
fi

git checkout main
