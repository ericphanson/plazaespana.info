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

# Fetch remote branches to check if canonical branch exists on remote
git fetch origin

# Check if canonical branch exists on remote
if git show-ref --verify --quiet "refs/remotes/origin/$CANONICAL_BRANCH"; then
    echo "Canonical branch '$CANONICAL_BRANCH' exists on remote, checking out..."
    # Checkout remote branch (creates local tracking branch if needed)
    git checkout -B "$CANONICAL_BRANCH" "origin/$CANONICAL_BRANCH"
    # Reset to main to ensure clean state
    git reset --hard main
else
    echo "Creating new canonical branch '$CANONICAL_BRANCH'..."
    git checkout -b "$CANONICAL_BRANCH"
fi

# Create local directory
mkdir -p "$LOCAL_DIR"

echo "📊 Syncing AWStats database from $NFSN_HOST:$REMOTE_DIR"

# Download all .txt files from AWStats data directory
# This includes monthly stats files like awstatsMMYYYY.awstats.txt
scp -q "$NFSN_USER@$NFSN_HOST:$REMOTE_DIR/*.txt" "$LOCAL_DIR/" 2>/dev/null || {
    echo "⚠️  No .txt files found on server (AWStats may not have processed logs yet)"
    git checkout main  # Return to main before exiting
    exit 0
}

# Count files
FILE_COUNT=$(ls -1 "$LOCAL_DIR"/*.txt 2>/dev/null | wc -l)
echo "✅ Downloaded $FILE_COUNT database file(s)"

# Strip out sections containing individual IPs/personal data
# Keep aggregate statistics sections only
echo "🔒 Removing individual visitor data (keeping aggregate stats only)..."
for file in "$LOCAL_DIR"/*.txt; do
    if [ -f "$file" ]; then
        # Create temp file with only aggregate sections
        # Remove: BEGIN_VISITOR, BEGIN_SIDER_*, BEGIN_ROBOT, BEGIN_WORMS, BEGIN_EMAILSENDER, BEGIN_EMAILRECEIVER
        # Keep: BEGIN_GENERAL, BEGIN_TIME, BEGIN_DAY, BEGIN_DOMAIN, BEGIN_BROWSER, BEGIN_OS, BEGIN_REFERER, etc.
        awk '
            /^BEGIN_VISITOR|^BEGIN_SIDER_|^BEGIN_ROBOT|^BEGIN_WORMS|^BEGIN_EMAILSENDER|^BEGIN_EMAILRECEIVER/ {
                skip=1
                next
            }
            /^END_VISITOR|^END_SIDER_|^END_ROBOT|^END_WORMS|^END_EMAILSENDER|^END_EMAILRECEIVER/ {
                skip=0
                next
            }
            !skip {
                print
            }
        ' "$file" > "$file.tmp"
        mv "$file.tmp" "$file"
    fi
done
echo "✅ Privacy filtering complete - individual IPs removed"

# Check if we have any changes
if [[ -z $(git status --porcelain "$LOCAL_DIR") ]]; then
    echo "✅ No changes detected - database is already in sync"
    git checkout main  # Return to main before exiting
    exit 0
fi

# Show what changed
echo ""
echo "📝 Changes detected:"
git status --short "$LOCAL_DIR"
echo ""

# Extract statistics from the most recent month's database file
# Parse BEGIN_GENERAL section for summary stats
LATEST_DB=$(ls -1t "$LOCAL_DIR"/awstats*.txt 2>/dev/null | head -1)
STATS_SUMMARY=""
if [ -f "$LATEST_DB" ]; then
    # Extract key stats from BEGIN_GENERAL section
    TOTAL_VISITS=$(awk '/BEGIN_GENERAL/,/END_GENERAL/ {if ($1 == "TotalVisits") print $2}' "$LATEST_DB")
    TOTAL_UNIQUE=$(awk '/BEGIN_GENERAL/,/END_GENERAL/ {if ($1 == "TotalUnique") print $2}' "$LATEST_DB")
    FIRST_TIME=$(awk '/BEGIN_GENERAL/,/END_GENERAL/ {if ($1 == "FirstTime") print $2}' "$LATEST_DB")
    LAST_TIME=$(awk '/BEGIN_GENERAL/,/END_GENERAL/ {if ($1 == "LastTime") print $2}' "$LATEST_DB")

    # Format dates (YYYYMMDDHHMMSS -> YYYY-MM-DD HH:MM)
    if [ -n "$FIRST_TIME" ] && [ -n "$LAST_TIME" ]; then
        FIRST_DATE=$(echo "$FIRST_TIME" | sed 's/^\([0-9]\{4\}\)\([0-9]\{2\}\)\([0-9]\{2\}\)\([0-9]\{2\}\)\([0-9]\{2\}\).*/\1-\2-\3 \4:\5/')
        LAST_DATE=$(echo "$LAST_TIME" | sed 's/^\([0-9]\{4\}\)\([0-9]\{2\}\)\([0-9]\{2\}\)\([0-9]\{2\}\)\([0-9]\{2\}\).*/\1-\2-\3 \4:\5/')

        STATS_SUMMARY="## Latest Statistics

- **Total visits:** ${TOTAL_VISITS:-0}
- **Unique visitors:** ${TOTAL_UNIQUE:-0}
- **Period:** $FIRST_DATE to $LAST_DATE

"
    fi
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
    git push --force origin "$CANONICAL_BRANCH"

    gh pr create \
        --title "Sync AWStats database files" \
        --body "$(cat <<EOFPR
Automated PR to sync AWStats database files from production server.

${STATS_SUMMARY}**Database files:** $FILE_COUNT total

See \`awstats-data/README.md\` for details on privacy configuration and data format.

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
