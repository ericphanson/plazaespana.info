# Hosted Aggregated Stats Runbook (NFS Hosted + Bunny Backup)

**Date:** 2026-02-21  
**Status:** Proposed  
**Decision:** Serve analytics report from NFS Apache, and use Bunny for backup copies of aggregate JSON artifacts.

---

## 1. Objective

Replace the current git-sync analytics loop with direct hosted publication + bounded backup:

1. Run `log-analyzer` on NFS against server logs.
2. Produce only anonymized/aggregated artifacts.
3. Validate privacy constraints.
4. Publish report and JSON to NFS Apache paths.
5. Mirror JSON to Bunny as backup (no unbounded snapshots).

This removes ongoing PR maintenance while preserving privacy and recoverability.

---

## 2. Requirements and Constraints

1. Repository is public: never publish raw logs, IPs, or unique request identifiers.
2. Bunny credentials must not be committed to git.
3. NFS is the only host with log access and should remain the source of truth.
4. Publication to NFS public paths should be atomic (no partial updates).
5. Fail closed: if privacy checks fail, do not publish or back up.
6. Completed month JSON backups are immutable and must never be overwritten or deleted on Bunny.

---

## 3. Target Architecture

### 3.1 Data flow

1. Cron executes `/home/private/bin/log-analyzer-weekly.sh`.
2. Script runs `/home/private/bin/log-analyzer` and generates output in a temp directory.
3. Privacy validation checks temp artifacts.
4. Temp artifacts are promoted to `/home/private/log-analyzer-data` atomically.
5. Publish to Apache-served path:
   - `/home/public/analytics/report.html` (primary human-facing report)
   - `/home/public/analytics/data/` (public aggregate JSON files)
6. Mirror JSON files to Bunny backup path:
   - `analytics-backup/current/` (single bounded mirror, no timestamped snapshots)
   - immutability guards prevent touching completed month JSON backups

### 3.2 Published artifacts

1. `lifetime.json`
2. `YYYY-MM.json` files
3. `manifest.json` (required; publish metadata, checksums, month list)
4. `report.html` (public via Apache on NFS)

---

## 4. Bunny One-Time Setup (Backup Target)

1. Create a Bunny Storage Zone for analytics backup objects.
2. Pull Zone is optional (only needed if you later want CDN serving from Bunny).
3. Record these values for deployment:
   - Storage zone name
   - Storage endpoint hostname/region
   - Base path prefix (`analytics-backup/current`)
4. Keep lifecycle simple:
   - each run updates only mutable files
   - no blanket delete behavior on Bunny backup path
   - completed month JSON files are write-once

---

## 5. NFS Secret Deployment (Bunny Key to `/home/private`)

### 5.1 Secret files on NFS

Store Bunny publish credentials as rootless private files:

1. `/home/private/bunny-storage-key.txt` (required, mode `600`)
2. `/home/private/bunny-storage-zone.txt` (required, mode `600`)
3. `/home/private/bunny-storage-endpoint.txt` (required, mode `600`)
4. `/home/private/bunny-base-path.txt` (optional, default `analytics-backup/current`, mode `600`)

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
3. Publish to `/home/public/analytics/` (report + JSON) atomically.
4. Mirror JSON files to Bunny `analytics-backup/current/` with immutability checks.
5. If regenerated immutable-month content differs, preserve existing canonical month file (do not overwrite).
6. Never delete completed-month backup files from Bunny.
7. Reject if a completed-month Bunny backup file already exists and differs from canonical local content.
8. Verify mirrored object count and sizes/checksums.
9. Log success/failure.
10. Exit non-zero on any failure.

Mutable vs immutable backup files:

1. Mutable:
   - `lifetime.json`
   - `manifest.json`
   - current month file (`YYYY-MM.json` for current UTC month)
   - previous month file during grace window (default: first 7 UTC days of the new month)
2. Immutable:
   - all older `YYYY-MM.json` files

### 6.2 Update `ops/log-analyzer-weekly.sh`

Use explicit phases:

1. Generate to temp output directory.
2. Run privacy checks on temp output.
3. Atomically update `/home/private/log-analyzer-data`.
4. Run publish script.
5. If publish fails:
   - keep local data update,
   - keep previously served `/home/public/analytics` unchanged (no partial replace),
   - exit non-zero so cron alerting triggers.

---

## 7. Privacy Gate Specification (Hard Fail)

Before any publication:

1. Reject if IPv4 patterns are found in outputs.
2. Reject if IPv6 patterns are found in outputs.
3. Reject if any JSON `path` contains `?` query strings.
4. Reject if any output includes unexpected keys outside a whitelist (optional strict mode, recommended).
5. Reject if input/output file set is incomplete (missing `lifetime.json` or malformed month files).
6. Reject Bunny sync plan if it attempts to overwrite/delete immutable month files.

Log exact failing filename and rule.

---

## 8. Backup and Retention Without Git History

1. Bunny stores one current backup set of JSON files (`analytics-backup/current/`).
2. Sync is guarded:
   - mutable files may be overwritten,
   - immutable month files may be created once but never changed/deleted.
3. Local NFS copy in `/home/private/log-analyzer-data` remains the canonical source.
4. Optional periodic archive (outside Bunny) can be added later if long-term historical rollback is needed.

Note: JSON file count grows only with months covered (`YYYY-MM.json`), which is naturally linear and bounded for practical use.

---

## 9. Migration Plan

### Phase A: Dual-write (1-2 weeks)

1. Keep current git-sync workflow active.
2. Add Bunny publish in parallel.
3. Compare NFS local, Bunny backup copy, and git-synced files.
4. Resolve any diffs before cutover.

### Phase B: Cutover

1. Keep NFS Apache as canonical public endpoint (`/analytics/report.html` + `/analytics/data/*.json`).
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
   - previously served Apache report/data remains unchanged,
   - cron exits non-zero.
2. Privacy failure:
   - no upload attempted,
   - actionable error in `/home/logs/log-analyzer.log`.
3. Immutable-file safety failure:
   - Bunny backup sync is aborted,
   - error identifies the file that would have been mutated.

### 10.3 Monitoring

Track in `/home/logs/log-analyzer.log`:

1. job start and end timestamps
2. generated file counts
3. privacy checks pass/fail
4. upload result
5. Bunny mirror status and file count
6. immutable-file checks performed and pass/fail

External stale check:

1. Read `https://plazaespana.info/analytics/data/manifest.json`.
2. Alert if `published_at` is older than expected schedule.

---

## 11. Changes Needed in This Repository

1. Add `ops/log-analyzer-publish.sh`.
2. Update `ops/log-analyzer-weekly.sh` to call publish script and enforce privacy gates.
3. Update root `justfile` deploy flow to upload Bunny secret/config files to NFS (`/home/private`, `chmod 600`).
4. Update `docs/deployment.md` for Apache-served report + Bunny backup mirroring.
5. Remove git-sync analytics workflow/scripts after cutover.

---

## 12. Pros/Cons vs Current Git-Sync Workflow

### Pros

1. Lower operational overhead (no recurring analytics PR maintenance).
2. Faster availability of fresh analytics.
3. Clear separation: NFS serves public report, Bunny holds backup copy of JSON.
4. Backup is bounded and cheap (no unbounded snapshot growth).

### Cons

1. Git history is no longer the backup mechanism.
2. No PR checkpoint before publish.
3. Requires stronger secret lifecycle management.
4. Requires explicit monitoring of publish + backup sync.

---

## 13. Implementation Checklist

1. Bunny Storage Zone configured (Pull Zone optional).
2. Bunny publish key generated.
3. Deploy flow supports `BUNNY_*` env vars and writes `/home/private/bunny-*.txt`.
4. `ops/log-analyzer-publish.sh` implemented and deployed.
5. `ops/log-analyzer-weekly.sh` updated for validate-then-publish flow.
6. Privacy gates enabled and tested with known bad fixtures.
7. Immutable-file guard logic tested (attempted overwrite/delete must fail).
8. `manifest.json` generated and published with each run.
9. Dual-write comparison completed with zero diffs.
10. Git-sync workflow removed.
11. Stale-data alerting in place and Bunny mirror verified.

---

## 14. Open Decisions

1. Should Bunny backup include `report.html` too, or JSON-only?
2. Keep default 7-day UTC grace period, or tighten to 3 days if late logs are rare.
3. Keep a minimal `log-analyzer-data/` directory in git for documentation, or delete it?
