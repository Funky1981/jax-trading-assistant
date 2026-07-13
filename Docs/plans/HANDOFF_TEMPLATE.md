# Handoff Template

Use this copy-block format at the end of every completed implementation phase.

```text
HANDOVER SUMMARY

Phase:
<phase name>

Commit:
<short sha> <commit subject>

Branch:
<branch name>

Files changed:
- <path>
- <path>

What changed:
- <concise implementation summary>
- <safety boundary summary>

Migrations:
<none, or list migration files and rollback notes>

Tests/verification run:
- <command>
- <command>

Known issue:
- <known verification blocker or "None">

Safety notes:
- <no live trading / no broker execution / no approval flow changes, as applicable>

Remaining risks:
- <known risk>

Recommended next phase:
<next phase>

What's Left:
- <remaining required work, skipped checks, optional broader validation, deployment/migration steps, and known risks>
```
