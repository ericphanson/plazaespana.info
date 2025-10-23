# Documentation Structure

This directory contains the project's documentation, organized by purpose and maintenance status.

## Maintained Documentation

These files are actively maintained and should be kept up to date:

- **workflows/** - Active workflows and processes (e.g., screenshot workflow, deployment procedures)
- **deployment.md** - Deployment instructions and operational guidance

For general project information, see the root-level documentation:
- **../README.md** - Quick start guide and project overview
- **../CLAUDE.md** - Development guidelines for Claude Code
- **../justfile** - Task automation reference (also serves as command documentation)

## Archived Documentation

These directories contain historical records that are not meant to be maintained:

- **plans/** - Implementation plans with date prefixes (YYYY-MM-DD format)
  - Design documents and feature planning
  - Implementation roadmaps
  - Architecture decisions

- **logs/** - Implementation logs, investigation notes, and retrospectives with date prefixes
  - Build logs and implementation records
  - Investigation findings
  - Retrospectives and post-mortems
  - Historical design documents

## Naming Convention

All archived documentation follows a consistent naming pattern:

```
YYYY-MM-DD-short-description.md
```

Examples:
- `plans/2025-10-19-madrid-events-site-generator.md`
- `logs/2025-10-20-distrito-filtering.md`
- `logs/2025-10-23-icons.md`

The date represents when the document was created or the work was completed, not necessarily when the file was committed.

## When to Archive

Documentation should be archived (moved to plans/ or logs/) when:

1. **It describes a completed implementation** - The work is done and the document serves as a historical record
2. **It captures point-in-time decisions** - Design rationale, trade-offs, or investigations specific to a moment
3. **It's not needed for day-to-day development** - Information that's useful for understanding history but not current work

Documentation should remain active when:

1. **It describes ongoing processes** - Workflows that are regularly used
2. **It needs frequent updates** - Operational procedures that change with the system
3. **It's essential reference material** - Information developers need regularly

## Finding Information

- **For current practices**: Check maintained documentation (workflows/, deployment.md, README.md, CLAUDE.md, justfile)
- **For implementation history**: Search logs/ by date or keyword
- **For design decisions**: Search plans/ and logs/ for relevant documents
- **For commands**: Check justfile first, then workflows/

When in doubt, maintained documentation takes precedence over archived documentation.
