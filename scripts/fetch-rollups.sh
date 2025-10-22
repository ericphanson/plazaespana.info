#!/bin/bash
# Download new AWStats rollups from server via SCP and update PR
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

REMOTE_DIR="/home/private/rollups"
ARCHIVE_DIR="awstats-archives"
CANONICAL_BRANCH="awstats-rollups"

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

mkdir -p "$ARCHIVE_DIR"

echo "Checking for new rollups on $NFSN_HOST:$REMOTE_DIR"

# Get list of available rollups via SSH
AVAILABLE=$(ssh "$NFSN_USER@$NFSN_HOST" "ls -1 $REMOTE_DIR/*.txt.gz 2>/dev/null | xargs -n1 basename" || true)

if [ -z "$AVAILABLE" ]; then
    echo "No rollups found on server"
    exit 0
fi

# Find new rollups (not in repo)
NEW_ROLLUPS=()
for rollup in $AVAILABLE; do
    if [ ! -f "$ARCHIVE_DIR/$rollup" ]; then
        NEW_ROLLUPS+=("$rollup")
    fi
done

if [ ${#NEW_ROLLUPS[@]} -eq 0 ]; then
    echo "No new rollups to archive"
    exit 0
fi

# Download new rollups via SCP
echo "Found ${#NEW_ROLLUPS[@]} new rollup(s)"
for rollup in "${NEW_ROLLUPS[@]}"; do
    echo "Downloading $rollup..."
    scp -q "$NFSN_USER@$NFSN_HOST:$REMOTE_DIR/$rollup" "$ARCHIVE_DIR/$rollup"
done

# Create or update README
cat > "$ARCHIVE_DIR/README.md" << 'EOF'
# AWStats Weekly Rollups

Compressed Apache access logs, archived weekly for permanent record.

## File Format
- Filename: `YYYY-Www.txt.gz` (ISO 8601 week numbering)
- Example: `2025-W43.txt.gz` = Week 43 of 2025
- Contents: Compressed Apache combined log format

## Viewing
```bash
# Extract and view
gunzip -c 2025-W43.txt.gz | less

# Count requests
gunzip -c 2025-W43.txt.gz | wc -l

# Top IPs
gunzip -c 2025-W43.txt.gz | awk '{print $1}' | sort | uniq -c | sort -rn | head -20
```

## Recovery
These archives can be replayed through AWStats to rebuild statistics:
```bash
gunzip -c 2025-W43.txt.gz | perl /usr/local/www/awstats/cgi-bin/awstats.pl -config=nfsn -update -LogFile=-
```
EOF

# Count total archives
TOTAL_COUNT=$(ls -1 "$ARCHIVE_DIR"/*.txt.gz 2>/dev/null | wc -l)
echo "" >> "$ARCHIVE_DIR/README.md"
echo "Total archived weeks: $TOTAL_COUNT" >> "$ARCHIVE_DIR/README.md"

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
git add "$ARCHIVE_DIR"
git commit -m "chore: archive AWStats rollups for ${NEW_ROLLUPS[*]}"

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
        --title "Archive AWStats weekly rollups" \
        --body "$(cat <<EOFPR
Automated PR to archive weekly AWStats rollups.

**New rollups:**
$(printf '- `%s`\n' "${NEW_ROLLUPS[@]}")

**Total archived:** $TOTAL_COUNT weeks

These compressed access logs provide a permanent record of site traffic and can be used to rebuild AWStats statistics if needed.

---

This PR uses a canonical branch (\`$CANONICAL_BRANCH\`) that is force-pushed with new rollups. Merge this PR to archive the rollups permanently.
EOFPR
)" \
        --head "$CANONICAL_BRANCH" \
        --label "automated" \
        --label "awstats"

    echo "✅ Pull request created from branch: $CANONICAL_BRANCH"
fi

# Return to main
git checkout main
