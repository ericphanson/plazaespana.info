# PR Preview Validation

## Overview

When a pull request is opened or updated, the PR preview deployment workflow automatically:

1. **Deploys** the site to `https://plazaespana.info/previews/PR{number}/`
2. **Validates** the deployed preview
3. **Reports** validation results in a PR comment
4. **Fails the check** if validation issues are found

## Validation Checks

### 1. Link Checking
- Scans all internal links on the preview site
- Detects broken links, missing assets (CSS, images, icons)
- Uses `broken-link-checker` (pinned version)

### 2. HTML Validation
- Validates HTML syntax and structure
- Checks for:
  - DOCTYPE correctness
  - Duplicate IDs
  - Missing required attributes
  - Trailing whitespace
- Uses `html-validate@10.2.1` (pinned version)

## Workflow Details

### Timing
- **Wait period**: 5 seconds after deployment (server readiness)
- **Validation duration**: ~10-30 seconds depending on site size

### Check Behavior
- ✅ **Pass**: All links work, HTML validates → Green checkmark on PR
- ❌ **Fail**: Broken links or validation errors → Red X on PR, blocks merge

### PR Comment
The workflow updates a comment on the PR with:
- Deployment status
- Preview URL (clickable)
- Validation results (expandable details)
- Link to job logs for detailed errors

Example comment:
```markdown
## 🚀 Preview Deployment

**Status:** ✅ Deployed and validated!

**Preview URL:** https://plazaespana.info/previews/PR26/

### Validation Results

✅ All checks passed!

<details>
<summary>✅ Link Check</summary>
✅ No broken links found
</details>

<details>
<summary>✅ HTML Validation</summary>
✅ HTML validates successfully
</details>
```

## Local Testing

Test the same validations locally:

```bash
# Full scan (all checks)
just scan https://plazaespana.info/previews/PR26

# Individual checks
just scan-links https://plazaespana.info/previews/PR26
just scan-html https://plazaespana.info/previews/PR26
```

**Note**: URLs with or without trailing slashes both work correctly (auto-normalized).

## Troubleshooting

### Broken Links
- Check job logs for specific broken URLs
- Common causes:
  - Missing weather icons (empty sky codes)
  - Incorrect asset paths
  - Typos in href attributes

### HTML Validation Errors
- Check job logs for line numbers and error details
- Common issues:
  - Duplicate IDs (events in multiple time groups)
  - Missing lang attribute
  - Incorrect DOCTYPE format
  - Trailing whitespace in template

### False Positives
If validation fails incorrectly:
1. Check if the preview URL is accessible
2. Verify the server had enough time to start (5s wait)
3. Re-run the workflow (sometimes transient network issues)

## Configuration

### Pinned Dependencies
Dependencies are pinned in `package.json` to prevent breaking changes:
- `html-validate@10.2.1` - HTML validator
- `broken-link-checker@0.7.8` - Link checker

To update versions:
```bash
npm update html-validate
npm update broken-link-checker
git add package.json package-lock.json
git commit -m "chore: update validation tools"
```

### Workflow File
`.github/workflows/pr-preview.yml`

Key configuration:
- `continue-on-error: true` - Allows comment to update even if validation fails
- `if: always()` - Ensures comment updates regardless of validation outcome
- Node.js 20 with npm cache for faster subsequent runs

## Related

- [Scanning Workflow](./scanning.md) - Manual scanning tools
- [CI Workflow](../../.github/workflows/ci.yml) - Automated testing on all PRs
- [Integration Tests](../testing.md) - HTML validation in tests
