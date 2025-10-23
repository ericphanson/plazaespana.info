# Open Source Risk Assessment

**Repository:** plaza-espana-calendar
**Assessment Date:** 2025-10-23
**Prepared by:** Claude Code Analysis

---

## Table of Contents

1. [Executive Summary](#executive-summary)
2. [Quick Action Summary](#quick-action-summary)
3. [Risk Scoring Methodology](#risk-scoring-methodology)
4. [Risk Assessment Matrix](#risk-assessment-matrix)
   - [Secrets and Credentials](#1-secrets-and-credentials)
   - [Deployment and Infrastructure Exposure](#2-deployment-and-infrastructure-exposure)
   - [Personal Information and Privacy](#3-personal-information-and-privacy)
   - [Security Vulnerabilities](#4-security-vulnerabilities)
   - [Intellectual Property and Licensing](#5-intellectual-property-and-licensing)
   - [Operational and Availability Risks](#6-operational-and-availability-risks)
   - [Reputational and Social Risks](#7-reputational-and-social-risks)
5. [Summary Risk Matrix](#summary-risk-matrix)
6. [Pre-Release Checklist](#pre-release-checklist)
7. [Post-Release Monitoring](#post-release-monitoring)
8. [Conclusion](#conclusion)

---

## Executive Summary

This document assesses the risks associated with open sourcing the plaza-espana-calendar repository. The project is a static site generator for Madrid events near Plaza de España, currently deployed to a personal domain (plazaespana.info).

### Overall Risk Assessment

**Current Status:** ⚠️ **MODERATE to HIGH RISK** (primarily due to missing LICENSE file)
**Post-Mitigation Status:** ✅ **LOW RISK** (safe to open source after critical mitigations)

### Key Findings

**Strengths:**
- ✅ No hardcoded secrets or credentials in tracked files
- ✅ Good security practices (strict CSP, privacy filtering, no user input)
- ✅ Zero external dependencies (stdlib only)
- ✅ Privacy-focused analytics (IPs filtered from AWStats data)
- ✅ Well-documented architecture and clean code

**Concerns:**
- ❌ **Missing LICENSE file** - CRITICAL blocker
- ⚠️ Extensive deployment documentation exposes infrastructure details
- ⚠️ Personal identifiers (username, domain) throughout codebase
- ⚠️ No SECURITY.md documenting security model
- ⚠️ No explicit support/maintenance expectations set

### Recommendation

**SAFE TO OPEN SOURCE** after completing the 4 critical checklist items below. The repository shows good security practices overall, but licensing must be resolved before public release.

---

## Quick Action Summary

**Before you can safely open source this repository, you MUST complete these 4 critical items:**

1. **Add LICENSE file** (30 minutes)
   - Recommended: MIT License
   - Blocks: All downstream use without license
   - See detailed text in [Section 5](#5-intellectual-property-and-licensing)

2. **Scan git history for secrets** (15 minutes)
   ```bash
   # Install and run gitleaks
   gitleaks detect --source . --verbose
   ```
   - If secrets found: rotate them immediately
   - Purge from history using `git filter-repo`

3. **Verify AWStats privacy filtering** (5 minutes)
   ```bash
   # Ensure no IPs in tracked data
   grep -r "BEGIN_VISITOR" awstats-data/
   # Should return empty (no results)
   ```

4. **Add security disclaimer to deployment docs** (10 minutes)
   - Add warning to top of `docs/deployment.md`
   - See template in [Section 2](#2-deployment-and-infrastructure-exposure)

**Estimated time to minimum viable release: 1 hour**

**High-priority (strongly recommended) follow-up actions:**
- Create SECURITY.md (15 min)
- Create ATTRIBUTION.md (10 min)
- Add project status section to README (10 min)
- Add `govulncheck` to CI (5 min)

---

## Risk Scoring Methodology

**Likelihood Scale (1-5):**
- 1 = Very Unlikely (< 5% chance)
- 2 = Unlikely (5-25% chance)
- 3 = Possible (25-50% chance)
- 4 = Likely (50-75% chance)
- 5 = Very Likely (> 75% chance)

**Severity Scale (1-5):**
- 1 = Negligible (minor inconvenience)
- 2 = Low (temporary disruption)
- 3 = Moderate (significant impact, recoverable)
- 4 = High (major impact, difficult recovery)
- 5 = Critical (severe impact, potential data loss/breach)

**Risk Score = Likelihood × Severity**
- 1-4: Low Risk (acceptable)
- 5-9: Moderate Risk (mitigate if feasible)
- 10-16: High Risk (must mitigate)
- 17-25: Critical Risk (must resolve before release)

---

## Risk Assessment Matrix

### 1. Secrets and Credentials

| Risk | Pre-Mitigation L | Pre-Mitigation S | Pre-Score | Post-Mitigation L | Post-Mitigation S | Post-Score |
|------|------------------|------------------|-----------|-------------------|-------------------|------------|
| Hardcoded API keys in source code | 1 | 5 | 5 | 1 | 5 | 5 |
| SSH private keys in repository | 1 | 5 | 5 | 1 | 5 | 5 |
| Database credentials | 1 | 4 | 4 | 1 | 4 | 4 |
| Leaked GitHub secrets in git history | 2 | 4 | 8 | 1 | 4 | 4 |

**Current Status:** ✅ **LOW RISK**

**Findings:**
- `.gitignore` properly excludes `.envrc.local`, `data/`, and sensitive files
- No hardcoded credentials found in source code
- All API endpoints use public open data (no authentication required)
- GitHub Actions uses secrets properly (not committed)
- SSH keys referenced in documentation but not committed

**Mitigations Applied:**
- ✅ `.gitignore` includes environment files
- ✅ Example config files (`.envrc.local.example`, `config.toml.example`) show structure without credentials
- ✅ Documentation instructs users to use GitHub Secrets for CI/CD

**Additional Mitigations Required:**
1. **Git history audit:** Scan entire git history for accidentally committed secrets
   ```bash
   # Use gitleaks or similar tool
   gitleaks detect --source . --verbose
   ```
2. **Rotate any credentials:** If secrets found in history, rotate them and use `git filter-repo` to purge
3. **Add pre-commit hook:** Consider adding secret detection to prevent future commits

**Post-Mitigation Actions:**
- [ ] Run gitleaks scan on entire repository history
- [ ] Document results in this file
- [ ] Rotate any discovered credentials (if found)

---

### 2. Deployment and Infrastructure Exposure

| Risk | Pre-Mitigation L | Pre-Mitigation S | Pre-Score | Post-Mitigation L | Post-Mitigation S | Post-Score |
|------|------------------|------------------|-----------|-------------------|-------------------|------------|
| Exposure of hosting provider (NFSN) | 5 | 2 | 10 | 5 | 1 | 5 |
| SSH server hostname disclosure | 5 | 2 | 10 | 5 | 1 | 5 |
| Directory structure disclosure | 5 | 2 | 10 | 3 | 1 | 3 |
| Cron job implementation details | 5 | 2 | 10 | 3 | 1 | 3 |
| Deployment workflow exposure | 5 | 2 | 10 | 3 | 2 | 6 |

**Current Status:** ⚠️ **MODERATE RISK** (requires mitigation)

**Findings:**
- `docs/deployment.md` contains extensive NFSN-specific deployment instructions
- SSH hostnames: `ssh.phx.nearlyfreespeech.net` (public information)
- Directory paths: `/home/public/`, `/home/private/`, `/home/protected/` (NFSN standard structure)
- Cron scripts in `ops/` directory reveal operational cadence
- GitHub Actions workflows show deployment automation
- AWStats configuration reveals analytics implementation

**Why This Matters:**
- Attackers can target specific hosting provider vulnerabilities
- Directory structure knowledge aids in crafting targeted attacks
- Cron timing information enables timing-based attacks
- Deployment workflow details could reveal weak points

**Mitigations Required:**

1. **Generalize deployment documentation** (severity reduction: 2→1)
   - Create `docs/deployment-example.md` with generic instructions
   - Move NFSN-specific details to private documentation
   - Keep `ops/` scripts (they show good practices) but add disclaimer

2. **Abstract infrastructure details** (likelihood reduction: 5→3)
   - Replace specific paths with variables in documentation
   - Example: `/home/public/` → `${DOCUMENT_ROOT}/`
   - Remove specific hostnames where possible

3. **Add security disclaimer** (severity reduction: 2→1)
   - Note that deployment examples may not suit all environments
   - Recommend security hardening for production use
   - Document assumes reader will adapt to their infrastructure

**Mitigation Implementation:**
```markdown
# Add to top of deployment.md:
> **⚠️ SECURITY NOTICE:** This deployment guide is specific to the original
> author's hosting environment and serves as an example. Do not blindly copy
> these configurations to production. Adapt paths, hostnames, and security
> settings to your infrastructure. Consider this a learning resource, not
> production-ready configuration.
```

**Post-Mitigation Actions:**
- [ ] Add security disclaimer to `docs/deployment.md`
- [ ] Review and generalize infrastructure-specific paths
- [ ] Consider moving some deployment details to a separate private repo or wiki

---

### 3. Personal Information and Privacy

| Risk | Pre-Mitigation L | Pre-Mitigation S | Pre-Score | Post-Mitigation L | Post-Mitigation S | Post-Score |
|------|------------------|------------------|-----------|-------------------|-------------------|------------|
| GitHub username exposure (ericphanson) | 5 | 1 | 5 | 5 | 1 | 5 |
| Domain name disclosure (plazaespana.info) | 5 | 1 | 5 | 5 | 1 | 5 |
| Email address in git commits | 4 | 1 | 4 | 4 | 1 | 4 |
| IP addresses in AWStats data | 1 | 4 | 4 | 1 | 4 | 4 |
| User analytics/tracking data | 1 | 5 | 5 | 1 | 5 | 5 |

**Current Status:** ✅ **LOW RISK**

**Findings:**
- **GitHub username:** `ericphanson` appears in 40+ files (module paths, URLs, documentation)
- **Domain name:** `plazaespana.info` referenced extensively (deployment docs, configs, tests)
- **AWStats data:** Privacy-filtered (IPs removed) before git commit
- **No PII:** No visitor personal information, email addresses, or user data collected
- **Analytics:** AWStats configured with privacy-first settings (aggregate stats only)

**Why This Is Low Risk:**
- GitHub username is already public (repository owner)
- Domain registration is public information (WHOIS)
- No sensitive personal information exposed
- AWStats privacy filtering (`scripts/fetch-stats-archives.sh:80-103`) removes individual visitor data

**Mitigations Applied:**
- ✅ AWStats privacy filtering in place (removes `BEGIN_VISITOR`, `BEGIN_ROBOT`, etc.)
- ✅ `.gitignore` excludes runtime logs and temporary data
- ✅ No analytics cookies or client-side tracking
- ✅ Strict CSP headers prevent third-party tracking scripts

**Additional Mitigations (Optional):**

1. **Anonymize examples** (likelihood reduction: 5→3, optional)
   - Replace `plazaespana.info` with `example.com` in documentation
   - Replace `ericphanson` with `your-username` in example paths
   - Keep actual values in code/configs (they work correctly)
   - **Trade-off:** Reduces clarity of working examples

2. **Verify AWStats privacy** (confidence building)
   ```bash
   # Verify no IPs in tracked AWStats data
   grep -r "BEGIN_VISITOR" awstats-data/
   # Should return no results if filtering works correctly
   ```

**Decision:** Recommend NOT anonymizing username/domain as they're already public and part of project identity. Privacy filtering of analytics data is sufficient.

**Post-Mitigation Actions:**
- [ ] Verify AWStats data contains no IPs: `grep -r "BEGIN_VISITOR" awstats-data/`
- [ ] Document privacy approach in README or separate privacy policy

---

### 4. Security Vulnerabilities

| Risk | Pre-Mitigation L | Pre-Mitigation S | Pre-Score | Post-Mitigation L | Post-Mitigation S | Post-Score |
|------|------------------|------------------|-----------|-------------------|-------------------|------------|
| Dependency vulnerabilities | 3 | 3 | 9 | 2 | 3 | 6 |
| Injection attacks (XSS, etc.) | 2 | 4 | 8 | 1 | 4 | 4 |
| Denial of service (DoS) | 3 | 2 | 6 | 2 | 2 | 4 |
| Supply chain attacks | 2 | 4 | 8 | 1 | 4 | 4 |
| Exposed attack surface | 3 | 3 | 9 | 2 | 2 | 4 |

**Current Status:** ⚠️ **MODERATE RISK** (requires mitigation)

**Findings:**

**Dependencies:**
- ✅ **Zero external dependencies** (`go.mod` shows stdlib only)
- ✅ **No npm/JavaScript dependencies** (pure CSS/HTML site)
- ⚠️ Go version dependency (currently 1.25, check for CVEs)

**Injection Protections:**
- ✅ **HTML template escaping** (`html/template` package auto-escapes)
- ✅ **Strict CSP** (`ops/htaccess:13` - blocks inline scripts, external resources)
- ✅ **No JavaScript** (eliminates XSS attack surface)
- ✅ **No user input** (static site, no forms or dynamic content)

**DoS Protections:**
- ⚠️ **Rate limiting:** Not implemented in application (relies on Apache/NFSN)
- ✅ **Caching:** Aggressive HTTP caching reduces server load
- ✅ **Static files:** No expensive computations at request time
- ⚠️ **Build process:** Could be abused if triggered frequently

**Attack Surface:**
- ✅ **No database** (eliminates SQL injection)
- ✅ **No authentication** (no credential attacks)
- ✅ **Read-only API** (JSON endpoint is static file)
- ⚠️ **Upstream fetching** (could be abused to make requests to datos.madrid.es)

**Mitigations Required:**

1. **Document security architecture** (confidence building)
   - Create `SECURITY.md` describing security model
   - Document CSP policy and rationale
   - Explain zero-trust approach (no user input = no injection)

2. **Add dependency scanning** (likelihood reduction: 3→2)
   - Enable Dependabot (already done, see `.github/workflows/ci.yml`)
   - Add `govulncheck` to CI pipeline
   ```bash
   # Add to CI workflow
   go install golang.org/x/vuln/cmd/govulncheck@latest
   govulncheck ./...
   ```

3. **Rate limit documentation** (severity reduction: 3→2)
   - Document reliance on hosting provider rate limiting
   - Add recommendations for production deployments
   - Note that NFSN provides basic DDoS protection

4. **Upstream abuse mitigation** (likelihood reduction: 3→2)
   - Already implemented: respectful fetching with delays
   - Already implemented: caching to reduce upstream requests
   - Document rate limiting in README

**Mitigation Implementation:**

Create `SECURITY.md`:
```markdown
# Security Policy

## Security Model

This is a static site generator with the following security characteristics:

- **No user input:** Site is generated server-side, no client-side forms
- **No authentication:** Public event listings only
- **No database:** All data from public APIs, regenerated hourly
- **No JavaScript:** Pure CSS/HTML eliminates XSS attack surface
- **Strict CSP:** Content-Security-Policy blocks external resources

## Reporting Vulnerabilities

Please report security vulnerabilities via GitHub Security Advisories.

## Known Limitations

- **No application-level rate limiting:** Relies on hosting provider
- **Public deployment details:** Infrastructure documented for educational purposes
```

**Post-Mitigation Actions:**
- [ ] Create `SECURITY.md` with security model documentation
- [ ] Add `govulncheck` to CI workflow
- [ ] Verify Dependabot is enabled
- [ ] Document rate limiting approach

---

### 5. Intellectual Property and Licensing

| Risk | Pre-Mitigation L | Pre-Mitigation S | Pre-Score | Post-Mitigation L | Post-Mitigation S | Post-Score |
|------|------------------|------------------|-----------|-------------------|-------------------|------------|
| Missing license file | 5 | 3 | 15 | 1 | 1 | 1 |
| Unclear copyright | 4 | 2 | 8 | 1 | 1 | 1 |
| Third-party attribution | 2 | 2 | 4 | 1 | 2 | 2 |
| Data source licensing | 3 | 3 | 9 | 1 | 2 | 2 |

**Current Status:** ⚠️ **HIGH RISK** (must resolve before release)

**Findings:**
- ❌ **No LICENSE file** in repository
- ❌ **No copyright notices** in source files
- ✅ **Attribution to Madrid data sources** in HTML template
- ⚠️ **Unclear reuse terms** without explicit license

**Third-Party Licensing:**
- Madrid open data: Requires attribution to "Ayuntamiento de Madrid – datos.madrid.es" ✅ (present in template)
- EsMadrid.com: Open data, attribution recommended
- Go standard library: BSD-3-Clause (compatible with most open source licenses)

**Mitigations Required:**

1. **Add LICENSE file** (CRITICAL - likelihood: 5→1, severity: 3→1)

   Recommended license: **MIT License** (permissive, widely compatible)

   Rationale:
   - Simple and permissive
   - Compatible with Madrid open data terms
   - No copyleft requirements
   - Well understood by community

   Alternative: **Apache 2.0** (if patent protection desired)

2. **Add copyright notices** (severity reduction: 2→1)
   ```go
   // Copyright 2025 Eric Hanson
   // SPDX-License-Identifier: MIT
   ```
   Add to main package files (not every file, just key ones)

3. **Document data attribution** (likelihood reduction: 3→1)
   - Create `ATTRIBUTION.md` documenting data sources
   - Ensure HTML template maintains attribution
   - Add note to README about open data sources

**Mitigation Implementation:**

Create `LICENSE` (MIT):
```text
MIT License

Copyright (c) 2025 Eric Hanson

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
```

Create `ATTRIBUTION.md`:
```markdown
# Data Attribution

This project uses open data from:

- **Ayuntamiento de Madrid** - Cultural events data
  - Source: https://datos.madrid.es/
  - License: Open data with attribution requirement
  - Attribution: "Ayuntamiento de Madrid – datos.madrid.es"

- **EsMadrid.com** - City events data
  - Source: https://www.esmadrid.com/opendata/
  - License: Open data
  - Attribution: Tourism board of Madrid

All event data is property of their respective sources and is used here
under open data terms with proper attribution displayed on the website.
```

**Post-Mitigation Actions:**
- [ ] Add LICENSE file (MIT recommended)
- [ ] Create ATTRIBUTION.md documenting data sources
- [ ] Add copyright notice to main.go
- [ ] Update README to mention license

---

### 6. Operational and Availability Risks

| Risk | Pre-Mitigation L | Pre-Mitigation S | Pre-Score | Post-Mitigation L | Post-Mitigation S | Post-Score |
|------|------------------|------------------|-----------|-------------------|-------------------|------------|
| Upstream API dependency | 4 | 2 | 8 | 3 | 2 | 6 |
| Domain/hosting coupling | 3 | 2 | 6 | 2 | 2 | 4 |
| Maintenance burden | 3 | 2 | 6 | 3 | 2 | 6 |
| Support expectations | 4 | 2 | 8 | 2 | 2 | 4 |

**Current Status:** ⚠️ **MODERATE RISK** (acceptable with documentation)

**Findings:**
- **Upstream dependencies:** Site requires datos.madrid.es and esmadrid.com APIs
- **Fallback handling:** ✅ Snapshot mechanism provides resilience
- **Domain coupling:** Configuration examples use plazaespana.info extensively
- **No SLA:** Personal project, no uptime guarantees

**Mitigations Required:**

1. **Document dependencies** (severity reduction: 2→2, confidence building)
   - Add "Dependencies" section to README
   - Note that Madrid APIs may change
   - Explain fallback/snapshot mechanism

2. **Set expectations** (likelihood reduction: 4→2)
   - Add clear disclaimer: "Personal project, no warranty"
   - No guarantee of maintenance or support
   - Contributions welcome but not required
   - Use issue template to manage expectations

3. **Generalize examples** (likelihood reduction: 3→2)
   - Make configuration examples more generic
   - Document how to adapt for different locations/data sources

**Mitigation Implementation:**

Add to README:
```markdown
## ⚠️ Project Status

This is a **personal project** with no warranty or support guarantees.

**Dependencies:**
- Madrid open data APIs (datos.madrid.es, esmadrid.com)
- FreeBSD/Linux hosting with Go runtime
- Static site hosting (any provider)

**Limitations:**
- APIs may change without notice
- No SLA or uptime guarantee
- Maintained on best-effort basis
- Contributions welcome but optional

**Fallback Mechanism:**
The site includes snapshot-based fallback. If upstream APIs fail, the last
successful data fetch is served with a "stale data" indicator.
```

**Post-Mitigation Actions:**
- [ ] Add project status section to README
- [ ] Create issue template setting support expectations
- [ ] Document upstream API dependencies

---

### 7. Reputational and Social Risks

| Risk | Pre-Mitigation L | Pre-Mitigation S | Pre-Score | Post-Mitigation L | Post-Mitigation S | Post-Score |
|------|------------------|------------------|-----------|-------------------|-------------------|------------|
| Code quality criticism | 3 | 1 | 3 | 2 | 1 | 2 |
| Security researcher attention | 2 | 2 | 4 | 2 | 2 | 4 |
| Unwanted contributions | 2 | 2 | 4 | 1 | 1 | 1 |
| Trademark issues (Plaza de España) | 1 | 3 | 3 | 1 | 2 | 2 |

**Current Status:** ✅ **LOW RISK**

**Findings:**
- Code quality appears good (tests, documentation, clean architecture)
- No controversial content or problematic naming
- "Plaza de España" is a geographic location (public domain)
- Project is non-commercial

**Mitigations (Optional):**

1. **Add CONTRIBUTING.md** (likelihood reduction: 2→1)
   - Set clear contribution guidelines
   - Reserve right to decline PRs
   - State project scope and goals

2. **Add code of conduct** (confidence building)
   - Use standard Contributor Covenant
   - Establishes community norms

3. **Trademark disclaimer** (severity reduction: 3→2)
   ```markdown
   "Plaza de España" refers to the geographic location in Madrid, Spain.
   This project is not affiliated with any government entity or tourism board.
   ```

**Post-Mitigation Actions:**
- [ ] Consider adding CONTRIBUTING.md (optional)
- [ ] Add geographic disclaimer to README (optional)

---

## Summary Risk Matrix

### Risk Heat Map

| Category | Pre-Mitigation<br>Score | Pre-Mitigation<br>Level | Post-Mitigation<br>Score | Post-Mitigation<br>Level | Priority |
|----------|-------------------------|-------------------------|--------------------------|--------------------------|----------|
| **Licensing & IP** | **15** | 🔴 **HIGH** | **1** | 🟢 **LOW** | ❌ **CRITICAL** |
| **Deployment Exposure** | 10 | 🟡 **MODERATE** | 5 | 🟢 **LOW** | ⚠️ High |
| **Security Vulnerabilities** | 9 | 🟡 **MODERATE** | 4 | 🟢 **LOW** | ⚠️ High |
| **Operational Risks** | 8 | 🟡 **MODERATE** | 6 | 🟡 **MODERATE** | ⚠️ Medium |
| **Secrets & Credentials** | 5 | 🟢 **LOW** | 5 | 🟢 **LOW** | ✅ Monitor |
| **Personal Information** | 5 | 🟢 **LOW** | 5 | 🟢 **LOW** | ✅ Accept |
| **Reputational Risks** | 3 | 🟢 **LOW** | 2 | 🟢 **LOW** | ✅ Accept |

**Legend:**
- 🔴 HIGH RISK (10-25): Must resolve before release
- 🟡 MODERATE RISK (5-9): Should mitigate if feasible
- 🟢 LOW RISK (1-4): Acceptable, monitor

### Summary

**Current State (Pre-Mitigation):**
- 1 High Risk (Licensing)
- 3 Moderate Risks (Deployment, Security, Operational)
- 3 Low Risks (Secrets, Privacy, Reputation)

**Target State (Post-Mitigation):**
- 0 High Risks
- 1 Moderate Risk (Operational - acceptable with documentation)
- 6 Low Risks

**Overall Assessment:** Currently **MODERATE to HIGH RISK** due to missing license. With mitigations applied: **LOW RISK** and safe to open source.

---

## Pre-Release Checklist

### Risk Mitigation Roadmap

This roadmap shows the recommended sequence for addressing risks. Complete each phase before moving to the next.

#### 🔴 **Phase 1: Critical Blockers** (Required - 1 hour total)

Must complete before any public release:

- [ ] **Add LICENSE file** (30 min)
  - Use MIT License (see [Section 5](#5-intellectual-property-and-licensing) for full text)
  - Commit to repository root as `LICENSE`
  - **Blocks:** All legal downstream use

- [ ] **Scan git history for secrets** (15 min)
  ```bash
  gitleaks detect --source . --verbose
  ```
  - If secrets found: rotate immediately, purge with `git filter-repo`
  - **Blocks:** Risk of credential compromise

- [ ] **Verify AWStats privacy** (5 min)
  ```bash
  grep -r "BEGIN_VISITOR" awstats-data/
  # Must return empty (no IPs stored)
  ```
  - **Blocks:** Potential privacy violation

- [ ] **Add deployment security disclaimer** (10 min)
  - Add warning to top of `docs/deployment.md`
  - See template in [Section 2](#2-deployment-and-infrastructure-exposure)
  - **Blocks:** Risk of infrastructure exposure

**Phase 1 Exit Criteria:** Repository is legally and technically safe for public release.

#### 🟡 **Phase 2: High Priority** (Strongly Recommended - 50 min total)

Complete within first week after open sourcing:

- [ ] **Create SECURITY.md** (15 min)
  - Document security model and vulnerability reporting
  - See template in [Section 4](#4-security-vulnerabilities)

- [ ] **Create ATTRIBUTION.md** (10 min)
  - Document Madrid open data sources
  - See template in [Section 5](#5-intellectual-property-and-licensing)

- [ ] **Update README with project status** (10 min)
  - Add disclaimer: personal project, no warranty
  - Document dependencies and limitations
  - See template in [Section 6](#6-operational-and-availability-risks)

- [ ] **Add `govulncheck` to CI** (5 min)
  ```bash
  # Add to .github/workflows/ci.yml
  go install golang.org/x/vuln/cmd/govulncheck@latest
  govulncheck ./...
  ```

- [ ] **Create issue template** (10 min)
  - Set support expectations
  - Link to SECURITY.md for vulnerabilities

**Phase 2 Exit Criteria:** Repository has clear security documentation and support boundaries.

#### 🟢 **Phase 3: Medium Priority** (Recommended - 40 min total)

Complete within first month:

- [ ] **Add copyright notices** (15 min)
  - Add to `cmd/buildsite/main.go`
  - Format: `// Copyright 2025 Eric Hanson // SPDX-License-Identifier: MIT`

- [ ] **Review deployment documentation** (15 min)
  - Generalize infrastructure-specific paths where possible
  - Replace `/home/public/` with `${DOCUMENT_ROOT}/` in examples

- [ ] **Add CONTRIBUTING.md** (10 min)
  - Set contribution guidelines
  - Reserve right to decline PRs
  - State project scope

**Phase 3 Exit Criteria:** Repository follows open source best practices.

#### ⚪ **Phase 4: Optional Enhancements**

Nice-to-have improvements:

- [ ] Add code of conduct (Contributor Covenant)
- [ ] Create separate private deployment repo for production configs
- [ ] Add geographic disclaimer to README
- [ ] Anonymize examples in documentation (trade-off: reduces clarity)

### Quick Checklist Summary

**CRITICAL (Required):**
- [ ] Add LICENSE file (MIT recommended)
- [ ] Scan git history for secrets (`gitleaks detect`)
- [ ] Verify AWStats data contains no IPs
- [ ] Add security disclaimer to deployment docs

**HIGH PRIORITY (Strongly Recommended):**
- [ ] Create SECURITY.md
- [ ] Create ATTRIBUTION.md
- [ ] Update README with project status
- [ ] Add `govulncheck` to CI
- [ ] Create issue templates

**MEDIUM PRIORITY (Recommended):**
- [ ] Add copyright notices to main package files
- [ ] Generalize infrastructure paths in documentation
- [ ] Add CONTRIBUTING.md

**LOW PRIORITY (Optional):**
- [ ] Add code of conduct
- [ ] Create separate deployment repo
- [ ] Add geographic disclaimer

---

## Post-Release Monitoring

After open sourcing, monitor for:

1. **GitHub Security Advisories**
   - Enable Dependabot alerts
   - Review and respond to CVE reports

2. **Unusual Activity**
   - Watch for suspicious forks
   - Monitor issues for security disclosures
   - Track who's cloning/starring the repo

3. **Upstream API Changes**
   - Monitor Madrid data portal for API changes
   - Update docs if APIs are deprecated

4. **Community Expectations**
   - Be clear about maintenance capacity
   - Welcome contributions but set boundaries
   - Close stale issues proactively

---

## Conclusion

The plaza-espana-calendar repository is **safe to open source** with the mitigations outlined above. The primary blocker is the **missing LICENSE file** (critical priority). Once licensing is resolved and deployment documentation is generalized, the repository presents minimal risk.

**Strengths:**
- No hardcoded secrets
- Good security practices (CSP, privacy filtering, no user input)
- Well-documented architecture
- Zero external dependencies
- Privacy-focused analytics

**Weaknesses (Mitigatable):**
- Missing license (CRITICAL)
- Extensive deployment documentation exposes infrastructure
- No SECURITY.md documenting security model
- No explicit support expectations set

**Final Recommendation:** **PROCEED WITH OPEN SOURCING** after completing the critical checklist items.

---

**Last Updated:** 2025-10-23
**Next Review:** After implementing critical mitigations
