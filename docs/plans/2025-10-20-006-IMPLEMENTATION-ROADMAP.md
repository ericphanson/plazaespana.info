# Implementation Roadmap: Data Format Fixes

**Created:** 2025-10-20
**Status:** READY TO START

## Overview

Fix three critical issues with Madrid events site:
1. Only 1 event showing (should be 42+)
2. Unicode corruption (`Dèco` → `D�co`)
3. JSON/XML parsing failures

**Key Insight:** Build a structured report FIRST to make debugging visible and organized.

## Documents

1. **`2025-10-20-build-report-design.md`** - Report format and structure
2. **`2025-10-20-data-format-fixes.md`** - Complete investigation and fix plan

## Execution Plan (4.5 hours total)

### Phase 0: Build Reporting Infrastructure (1 hour) ⭐ **START HERE**

**Why first?** Makes everything else easier by providing:
- Clear visibility into each build step
- Automatic grouping of failures
- Timing metrics
- Data quality warnings
- Examples of issues for debugging

**Tasks:**
- [ ] 0.1: Implement report types (30 min)
  - Create `internal/report/report.go`
  - Define BuildReport, FetchReport, ProcessingReport types
  - Implement text writer

- [ ] 0.2: Integrate into main.go (20 min)
  - Track fetch attempts (JSON, XML, CSV)
  - Track processing steps (dedup, geo filter, time filter)
  - Track output generation
  - Write report to `./public/build-report.txt`

- [ ] 0.3: Add data quality checks (10 min)
  - Detect encoding issues (non-UTF-8 strings)
  - Warn on restrictive radius (< 1% kept)
  - Track parse failures

- [ ] 0.4: Generate and review first report (10 min)
  - Build site
  - Read `./public/build-report.txt`
  - Verify all issues are visible

**Deliverable:** Structured report showing:
```
DATA FETCHING
  JSON: FAILED - invalid character '\n' at byte 45231
  XML: FAILED - root element <Contenidos> not <response>
  CSV: SUCCESS - 1001 events parsed

DATA PROCESSING
  Geo filter: 1/1001 kept (0.1%) - WARNING: very restrictive

DATA QUALITY
  Encoding issues: 15 events
    Example: "Madrid Art D�co" should be "Dèco"
```

---

### Phase 1: Quick Wins (30 min)

**Now informed by the report!**

- [ ] Fix default radius
  - Update flag default from 0.35 to 2.0
  - Update justfile
  - Test: report should show ~42 events

- [ ] Add UTF-8 charset to HTML
  - Add `<meta charset="UTF-8">` to template
  - Test: check if this alone fixes rendering

**Deliverable:** More events visible, possibly better encoding

---

### Phase 2: Investigation (30 min)

**Report makes this faster - look at the examples it provides!**

- [ ] Encoding analysis
  - Check CSV source encoding (report shows examples)
  - Identify if ISO-8859-1 or Windows-1252
  - Plan conversion approach

- [ ] JSON analysis
  - Use byte offset from report
  - Understand newline issue
  - Plan preprocessing

- [ ] XML analysis
  - Confirm root element from report
  - Map to correct structure
  - Update types.go

**Deliverable:** Clear understanding of each issue's root cause

---

### Phase 3: Implement Fixes (2 hours)

- [ ] Fix UTF-8 encoding (45 min)
  - Detect CSV encoding
  - Convert to UTF-8
  - Test with report (should show 0 encoding issues)

- [ ] Fix JSON parser (45 min)
  - Implement preprocessing for newlines
  - Handle gracefully if still fails
  - Test with report (should show JSON SUCCESS)

- [ ] Fix XML parser (30 min)
  - Update XMLResponse struct for <Contenidos>
  - Map fields correctly
  - Test with report (should show XML SUCCESS)

**Deliverable:** All three formats working

---

### Phase 4: Testing & Validation (1 hour)

- [ ] Unit tests (30 min)
  - Test UTF-8 conversion
  - Test JSON preprocessing
  - Test XML structure mapping
  - Test report generation

- [ ] Integration tests (15 min)
  - Test fallback priority (JSON → XML → CSV)
  - Test with real Madrid data

- [ ] Final validation (15 min)
  - Generate site
  - Review final report (all SUCCESS)
  - Check rendered site (no corruption)
  - Verify event count correct

**Deliverable:** Production-ready build with comprehensive reporting

---

## Success Metrics

### Before:
```
Build output:
  After filtering: 1 events

Website:
  Events: 1
  Titles: "Madrid Art D�co, 1925: El estilo de una nueva �poca"

Sources:
  JSON: FAILED
  XML: FAILED
  CSV: SUCCESS (but corrupted)
```

### After:
```
Build report:
  DATA FETCHING
    JSON: SUCCESS - 1001 events
    Source used: JSON (highest priority)

  DATA PROCESSING
    Geo filter: 42 events (2.0km radius)
    Time filter: 42 events

  DATA QUALITY
    Encoding issues: 0
    Parse failures: 0

  OUTPUT
    HTML: SUCCESS - 42 events
    JSON: SUCCESS - 42 events

Website:
  Events: 42
  Titles: "Madrid Art Déco, 1925: El estilo de una nueva época" ✓
```

---

## Key Design Decisions

1. **Report-first approach**
   - Provides observability before fixing
   - Makes debugging systematic
   - Validates fixes objectively

2. **Structured report format**
   - Human-readable text
   - Machine-readable JSON (optional)
   - Sections for each pipeline step

3. **Data quality tracking**
   - Automatic detection of common issues
   - Examples provided for debugging
   - Recommendations generated

4. **Graceful degradation**
   - JSON fails → try XML
   - XML fails → try CSV
   - Report shows what succeeded

---

## Files to Create/Modify

### New Files:
- `internal/report/report.go` - Report types
- `internal/report/writer.go` - Output formatting
- `internal/report/report_test.go` - Tests
- `docs/plans/2025-10-20-build-report-design.md` - Design doc
- `docs/plans/2025-10-20-data-format-fixes.md` - Fix plan
- `docs/logs/2025-10-20-data-format-implementation.md` - Execution log

### Modified Files:
- `cmd/buildsite/main.go` - Integrate reporting
- `internal/fetch/client.go` - Encoding fixes, JSON/XML fixes
- `internal/fetch/types.go` - XML structure update
- `templates/index.tmpl.html` - UTF-8 charset
- `justfile` - Default radius

---

## Risk Mitigation

**Risk:** Report implementation takes longer than 1 hour
- Mitigation: Start with minimal viable report, enhance later

**Risk:** Encoding issues persist after UTF-8 fix
- Mitigation: Report will show remaining issues with examples

**Risk:** JSON/XML have additional undiscovered issues
- Mitigation: Report tracks success/failure, CSV fallback always works

**Risk:** Breaking existing functionality
- Mitigation: All existing tests must pass, report validates output

---

## Next Steps

1. Start with Phase 0, Task 0.1 (implement report types)
2. Create execution log: `docs/logs/2025-10-20-data-format-implementation.md`
3. Work through tasks sequentially
4. Update log after each task
5. Generate report after each change to validate progress

**Ready to begin? Start with:**
```bash
# Create log file
touch docs/logs/2025-10-20-data-format-implementation.md

# Create report package
mkdir -p internal/report
touch internal/report/report.go
```
