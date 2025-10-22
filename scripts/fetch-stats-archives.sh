#!/bin/bash
# Sync AWStats database files from server to git repository
set -euo pipefail

# Check required commands
for cmd in ssh scp gh git; do
    if ! command -v "$cmd" >/dev/null 2>&1; then
        echo "❌ Error: Required command '$cmd' not found in PATH"
        exit 1
    fi
done

# Check required environment variables
if [ -z "${NFSN_HOST:-}" ] || [ -z "${NFSN_USER:-}" ]; then
    echo "❌ Error: NFSN_HOST and NFSN_USER environment variables required"
    echo "   Set in .envrc.local or export manually"
    exit 1
fi

# Validate gh CLI authentication
if ! gh auth status >/dev/null 2>&1; then
    echo "❌ Error: GitHub CLI (gh) is not authenticated"
    echo "   Run: gh auth login"
    echo "   Or set GH_TOKEN environment variable"
    exit 1
fi

REMOTE_DIR="/home/private/awstats-data"
LOCAL_DIR="awstats-data"
CANONICAL_BRANCH="awstats-data"

# Ensure clean git state
if [[ -n $(git status --porcelain) ]]; then
    echo "❌ Error: Working directory has uncommitted changes"
    git status
    exit 1
fi

# Ensure we're on main and up to date
CURRENT_BRANCH=$(git rev-parse --abbrev-ref HEAD)
if [ "$CURRENT_BRANCH" != "main" ]; then
    echo "Switching from $CURRENT_BRANCH to main..."
    git checkout main
fi
git pull origin main

# Create local directory
mkdir -p "$LOCAL_DIR"

echo "📊 Syncing AWStats database from $NFSN_HOST:$REMOTE_DIR"

# Download all .txt files from AWStats data directory
# This includes monthly stats files like awstatsMMYYYY.awstats.txt
scp -q "$NFSN_USER@$NFSN_HOST:$REMOTE_DIR/*.txt" "$LOCAL_DIR/" 2>/dev/null || {
    echo "⚠️  No .txt files found on server (AWStats may not have processed logs yet)"
    exit 0
}

# Count files
FILE_COUNT=$(ls -1 "$LOCAL_DIR"/*.txt 2>/dev/null | wc -l)
echo "✅ Downloaded $FILE_COUNT database file(s)"

# Create README
cat > "$LOCAL_DIR/README.md" << 'EOF'
# AWStats Database Files

Synchronized copy of AWStats aggregate statistics database.

## Privacy

These files contain **aggregate statistics only**:
- ✅ Page views, unique visitors, referrers (counts and percentages)
- ✅ Browser, OS, country statistics (aggregated)
- ✅ Monthly trends and historical data
- ❌ **No individual IP addresses**
- ❌ **No individual requests**
- ❌ **No personal information**

## File Format

- **Monthly stats:** `awstatsMMYYYY.awstats.txt` (e.g., `awstats102025.awstats.txt` for October 2025)
- **DNS cache:** `dnscachelastupdate.awstats.txt` (cached DNS lookups, hostnames only)
- **Other files:** Various AWStats state files

## Viewing Statistics

These are AWStats internal database files (text format). To view human-readable statistics:

1. Visit the live stats page: https://plazaespana.info/stats/
2. Or regenerate HTML locally:
   ```bash
   perl /usr/local/www/awstats/tools/awstats_buildstaticpages.pl \
     -configdir=. -config=awstats -update -dir=./html-output
   ```

## Recovery

To restore from this backup (e.g., after server failure):

```bash
# Copy to production location (NFSN)
scp awstats-data/*.txt user@host:/home/private/awstats-data/

# Regenerate HTML from database
ssh user@host '/home/private/bin/awstats-weekly.sh'
```

The database files preserve all historical visitor trends without storing any personal information.
EOF

# Check if we have any changes
if [[ -z $(git status --porcelain "$LOCAL_DIR") ]]; then
    echo "✅ No changes detected - database is already in sync"
    exit 0
fi

# Show what changed
echo ""
echo "📝 Changes detected:"
git status --short "$LOCAL_DIR"
echo ""

# Check if canonical branch exists locally
if git show-ref --verify --quiet "refs/heads/$CANONICAL_BRANCH"; then
    echo "Canonical branch '$CANONICAL_BRANCH' exists locally, checking out..."
    git checkout "$CANONICAL_BRANCH"
    # Reset to main to ensure clean state
    git reset --hard main
else
    echo "Creating canonical branch '$CANONICAL_BRANCH'..."
    git checkout -b "$CANONICAL_BRANCH"
fi

# Add changes and commit
git add "$LOCAL_DIR"
git commit -m "chore: sync AWStats database files from server

Updated database files with latest statistics.
Total files: $FILE_COUNT

These files contain aggregate statistics only (no IPs or individual requests)."

# Check if PR already exists
PR_NUMBER=$(gh pr list --head "$CANONICAL_BRANCH" --state open --json number --jq '.[0].number' || echo "")

if [ -n "$PR_NUMBER" ]; then
    echo "📝 Updating existing PR #$PR_NUMBER with force push..."
    git push --force origin "$CANONICAL_BRANCH"
    echo "✅ Updated PR #$PR_NUMBER: https://github.com/$(gh repo view --json nameWithOwner -q .nameWithOwner)/pull/$PR_NUMBER"
else
    echo "📝 Creating new pull request..."
    git push origin "$CANONICAL_BRANCH"

    gh pr create \
        --title "Sync AWStats database files" \
        --body "$(cat <<EOFPR
Automated PR to sync AWStats database files from production server.

**Files:** $FILE_COUNT database files

## Privacy-First Approach

These files contain **aggregate statistics only**:
- ✅ Page views, visitor counts, referrer statistics
- ✅ Browser/OS/country breakdowns (aggregated)
- ✅ Monthly trends and historical data
- ❌ **No individual IP addresses**
- ❌ **No individual requests**
- ❌ **No personal information**

The database files preserve historical visitor trends and can be used to regenerate HTML reports without storing raw logs.

---

This PR uses a canonical branch (\`$CANONICAL_BRANCH\`) that is force-pushed with updates. Merge to preserve the latest statistics in git history.
EOFPR
)" \
        --head "$CANONICAL_BRANCH" \
        --label "automated" \
        --label "awstats"

    echo "✅ Pull request created from branch: $CANONICAL_BRANCH"
fi

# Return to main
git checkout main
