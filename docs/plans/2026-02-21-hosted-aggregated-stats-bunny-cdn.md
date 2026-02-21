# Hosted Aggregated Stats Runbook (NFS + Bunny CDN)

**Date:** 2026-02-21  
**Status:** Proposed  
**Decision:** Use Bunny CDN as the public distribution path for aggregate analytics, with generation and publish controlled from NFS.

---

## 1. Objective

Replace the current git-sync analytics loop with direct hosted publication:

1. Run `log-analyzer` on NFS against server logs.
2. Produce only anonymized/aggregated artifacts.
3. Validate privacy constraints.
4. Publish to Bunny CDN (`latest` + timestamped snapshot).
5. Keep operational backups outside git.

This removes ongoing PR maintenance while preserving privacy and rollback ability.

---

## 2. Requirements and Constraints

1. Repository is public: never publish raw logs, IPs, or unique request identifiers.
2. Bunny credentials must not be committed to git.
3. NFS is the only host with log access and should remain the source of truth.
4. Publication must be atomic for `latest` (no partial updates).
5. Fail closed: if privacy checks fail, do not publish.

---

## 3. Target Architecture

### 3.1 Data flow

1. Cron executes `/home/private/bin/log-analyzer-weekly.sh`.
2. Script runs `/home/private/bin/log-analyzer` and generates output in a temp directory.
3. Privacy validation checks temp artifacts.
4. Temp artifacts are promoted to `/home/private/log-analyzer-data` atomically.
5. Publish script uploads to Bunny:
   - `analytics/snapshots/<UTC_TIMESTAMP>/...` (immutable)
   - `analytics/latest/...` (current pointer)
6. Optional: mirror `latest` to `/home/public/analytics/` as fallback.

### 3.2 Published artifacts

1. `lifetime.json`
2. `YYYY-MM.json` files
3. `manifest.json` (required; publish metadata, checksums, month list)
4. `report.html` (optional public artifact; JSON-only is acceptable)

---

## 4. Bunny CDN One-Time Setup

1. Create a Bunny Storage Zone for analytics objects.
2. Create or reuse a Pull Zone backed by that storage zone.
3. Decide public endpoint:
   - Bunny hostname, or
   - custom domain (for example `analytics.plazaespana.info` CNAME to Bunny).
4. Configure caching policy:
   - `snapshots/*`: long TTL (immutable)
   - `latest/*`: short TTL
   - `manifest.json`: short TTL
5. Record these values for deployment:
   - Storage zone name
   - Storage endpoint hostname/region
   - Base path prefix (`analytics`)

---

## 5. NFS Secret Deployment (Key to `/home/private`)

### 5.1 Secret files on NFS

Store Bunny publish credentials as rootless private files:

1. `/home/private/bunny-storage-key.txt` (required, mode `600`)
2. `/home/private/bunny-storage-zone.txt` (required, mode `600`)
3. `/home/private/bunny-storage-endpoint.txt` (required, mode `600`)
4. `/home/private/bunny-base-path.txt` (optional, default `analytics`, mode `600`)

### 5.2 Deploy mechanism (recommended)

Update deploy flow (`justfile`) so `just deploy` or `just deploy-only` can:

1. Read local env vars:
   - `BUNNY_STORAGE_KEY`
   - `BUNNY_STORAGE_ZONE`
   - `BUNNY_STORAGE_ENDPOINT`
   - `BUNNY_BASE_PATH` (optional)
2. Write temporary local files under `build/`.
3. `scp` to `/home/private/*.txt` on NFS.
4. `ssh chmod 600` on all Bunny secret/config files.
5. Remove temporary local files.

This mirrors the existing AEMET key pattern and avoids manual drift.

---

## 6. Script Changes

### 6.1 Add `ops/log-analyzer-publish.sh`

Responsibilities:

1. Read Bunny config/key from `/home/private/*.txt`.
2. Validate artifacts before upload.
3. Upload all files to `snapshots/<timestamp>/`.
4. Verify uploaded object count and sizes/checksums.
5. Promote by uploading/copying to `latest/`.
6. Log success/failure and published snapshot ID.
7. Exit non-zero on any failure.

### 6.2 Update `ops/log-analyzer-weekly.sh`

Use explicit phases:

1. Generate to temp output directory.
2. Run privacy checks on temp output.
3. Atomically update `/home/private/log-analyzer-data`.
4. Run publish script.
5. If publish fails:
   - keep local data update,
   - leave Bunny `latest` unchanged,
   - exit non-zero so cron alerting triggers.

---

## 7. Privacy Gate Specification (Hard Fail)

Before any publication:

1. Reject if IPv4 patterns are found in outputs.
2. Reject if IPv6 patterns are found in outputs.
3. Reject if any JSON `path` contains `?` query strings.
4. Reject if any output includes unexpected keys outside a whitelist (optional strict mode, recommended).
5. Reject if input/output file set is incomplete (missing `lifetime.json` or malformed month files).

Log exact failing filename and rule.

---

## 8. Backup and Retention Without Git History

1. Keep immutable Bunny snapshots (`snapshots/<timestamp>`).
2. Add retention job:
   - keep the most recent 52 weekly snapshots;
   - optionally keep one snapshot per month indefinitely.
3. Optional second copy:
   - periodic `tar.gz` of `/home/private/log-analyzer-data` to separate storage.
4. Keep `manifest.json` in each snapshot to simplify restore and audits.

---

## 9. Migration Plan

### Phase A: Dual-write (1-2 weeks)

1. Keep current git-sync workflow active.
2. Add Bunny publish in parallel.
3. Compare NFS local, Bunny `latest`, and git-synced files.
4. Resolve any diffs before cutover.

### Phase B: Cutover

1. Make Bunny URL the canonical public analytics endpoint.
2. Disable:
   - `.github/workflows/fetch-log-analyzer-stats.yml`
   - `scripts/fetch-log-analyzer-stats.sh`
   - `just fetch-log-analyzer-stats`

### Phase C: Cleanup

1. Update docs to remove PR-based analytics sync instructions.
2. Keep `log-analyzer-data/` in repo only as docs/placeholder, or remove entirely.
3. Add final runbook section for incident response and key rotation.

---

## 10. Operational Runbook

### 10.1 Key rotation

1. Generate new Bunny API key in Bunny dashboard.
2. Update local/CI secret values.
3. Re-run deploy to rewrite `/home/private/bunny-*.txt`.
4. Run publish job manually once to validate.
5. Revoke old Bunny key.

### 10.2 Failure behavior

1. Generation success + publish failure:
   - NFS local files update,
   - Bunny `latest` remains previous version,
   - cron exits non-zero.
2. Privacy failure:
   - no upload attempted,
   - actionable error in `/home/logs/log-analyzer.log`.

### 10.3 Monitoring

Track in `/home/logs/log-analyzer.log`:

1. job start and end timestamps
2. generated file counts
3. privacy checks pass/fail
4. upload result
5. published snapshot ID

External stale check:

1. Read `analytics/latest/manifest.json`.
2. Alert if `published_at` is older than expected schedule.

---

## 11. Changes Needed in This Repository

1. Add `ops/log-analyzer-publish.sh`.
2. Update `ops/log-analyzer-weekly.sh` to call publish script and enforce privacy gates.
3. Update root `justfile` deploy flow to upload Bunny secret/config files to NFS (`/home/private`, `chmod 600`).
4. Update `docs/deployment.md` for Bunny-based analytics publishing.
5. Remove git-sync analytics workflow/scripts after cutover.

---

## 12. Pros/Cons vs Current Git-Sync Workflow

### Pros

1. Lower operational overhead (no recurring analytics PR maintenance).
2. Faster availability of fresh analytics.
3. Clear separation: generation/publish on NFS, distribution on Bunny CDN.
4. Better cache behavior and low-cost delivery for public JSON.

### Cons

1. Git history is no longer the backup mechanism.
2. No PR checkpoint before publish.
3. Requires stronger secret lifecycle management.
4. Requires explicit monitoring and retention jobs.

---

## 13. Implementation Checklist

1. Bunny Storage + Pull Zone configured.
2. Bunny publish key generated.
3. Deploy flow supports `BUNNY_*` env vars and writes `/home/private/bunny-*.txt`.
4. `ops/log-analyzer-publish.sh` implemented and deployed.
5. `ops/log-analyzer-weekly.sh` updated for validate-then-publish flow.
6. Privacy gates enabled and tested with known bad fixtures.
7. `manifest.json` generated and published with each run.
8. Dual-write comparison completed with zero diffs.
9. Git-sync workflow removed.
10. Retention and stale-data alerting in place.

---

## 14. Open Decisions

1. Should `report.html` be public or private-only?
2. Keep `/home/public/analytics` mirror as fallback, or Bunny-only?
3. Retention policy details: 52 weeks only, or monthly forever?
4. Keep a minimal `log-analyzer-data/` directory in git for documentation, or delete it?
