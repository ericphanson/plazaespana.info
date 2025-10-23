#!/usr/bin/env bash
# Convert GitHub Actions tag references to pinned commit hashes
# Usage: ./scripts/pin-action-versions.sh
set -euo pipefail

echo "🔐 Pinning GitHub Actions to commit hashes..."
echo ""

# Function to get commit hash for a GitHub Action tag
get_action_hash() {
  local action=$1
  local tag=$2

  # Extract owner/repo from action
  local repo=$(echo "$action" | cut -d'@' -f1)

  echo "Looking up $repo@$tag..." >&2

  # Use GitHub API to get the commit hash for the tag
  local hash=$(curl -s "https://api.github.com/repos/$repo/git/ref/tags/$tag" | \
    jq -r '.object.sha // empty')

  # If tag lookup fails, try as a release
  if [ -z "$hash" ]; then
    hash=$(curl -s "https://api.github.com/repos/$repo/releases/tags/$tag" | \
      jq -r '.target_commitish // empty')
  fi

  if [ -z "$hash" ]; then
    echo "  ⚠️  Could not find hash for $repo@$tag" >&2
    return 1
  fi

  echo "$hash"
}

# Check for required tools
if ! command -v jq &> /dev/null; then
  echo "❌ Error: jq is required but not installed"
  echo "   Install with: apt-get install jq (or brew install jq on macOS)"
  exit 1
fi

if ! command -v curl &> /dev/null; then
  echo "❌ Error: curl is required but not installed"
  exit 1
fi

# Actions to pin (extracted from workflows)
declare -A actions=(
  ["actions/checkout"]="v4"
  ["actions/setup-go"]="v5"
  ["actions/upload-artifact"]="v4"
  ["actions/download-artifact"]="v4"
  ["peter-evans/find-comment"]="v3"
  ["peter-evans/create-or-update-comment"]="v4"
)

echo "Actions to pin:"
for action in "${!actions[@]}"; do
  echo "  - $action@${actions[$action]}"
done
echo ""

# Get commit hashes
declare -A hashes
for action in "${!actions[@]}"; do
  tag="${actions[$action]}"
  hash=$(get_action_hash "$action" "$tag" || echo "")
  if [ -n "$hash" ]; then
    # Get short hash (first 40 chars like setup-just example)
    short_hash=$(echo "$hash" | cut -c1-40)
    hashes["$action@$tag"]="$short_hash"
    echo "  ✅ $action@$tag -> $short_hash"
  else
    echo "  ❌ Failed to get hash for $action@$tag"
  fi
done
echo ""

# Update workflow files
echo "Updating workflow files..."
for workflow in .github/workflows/*.yml; do
  echo "  Processing $workflow..."

  # Create backup
  cp "$workflow" "$workflow.bak"

  # Replace tag references with hashes
  for action_tag in "${!hashes[@]}"; do
    action=$(echo "$action_tag" | cut -d'@' -f1)
    tag=$(echo "$action_tag" | cut -d'@' -f2)
    hash="${hashes[$action_tag]}"

    # Replace "uses: action@tag" with "uses: action@hash # tag"
    sed -i.tmp "s|uses: $action@$tag\$|uses: $action@$hash # $tag|g" "$workflow"
    rm -f "$workflow.tmp"
  done
done

# Remove backups if successful
rm -f .github/workflows/*.yml.bak

echo ""
echo "✅ All workflows updated with pinned commit hashes!"
echo ""
echo "Next steps:"
echo "  1. Review the changes: git diff .github/workflows/"
echo "  2. Test the workflows still work"
echo "  3. Commit: git add .github/workflows/ && git commit -m 'chore: pin GitHub Actions to commit hashes'"
echo ""
echo "Dependabot will now keep these hashes updated to match new versions."
