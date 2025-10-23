# Documentation Structure

**Philosophy**: Nearly all documentation should be point-in-time archives, not living documents. Avoid creating new maintained docs.

## Maintained Documentation (Rare)

Actively maintained docs - should be minimal:

- **workflows/** - Active workflows (e.g., screenshot workflow)
- **deployment.md** - Deployment instructions

Root-level maintained docs:
- **../README.md** - Project overview
- **../CLAUDE.md** - Development guidelines
- **../justfile** - Command reference

## Archived Documentation (Default)

Point-in-time records - where most docs belong:

- **plans/** - Implementation plans (dated: `YYYY-MM-DD-description.md`)
- **logs/** - Implementation logs, investigations, retrospectives (dated)
- **artifacts/** - Large data files from investigations (dated)

## Guidelines

**When to archive** (almost always):
- Completed implementations
- Point-in-time decisions or investigations
- Historical context that doesn't need updates

**When to keep maintained** (rarely):
- Operational procedures that change with the system
- Frequently referenced workflows

**Naming**: All archived docs use `YYYY-MM-DD-short-description.md`

**Default**: When in doubt, archive it. Maintained docs require ongoing maintenance burden - only create them if absolutely necessary.
