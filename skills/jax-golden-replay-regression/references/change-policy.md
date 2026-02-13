# Golden Change Policy

Use this policy when `tests/golden/*` or `tests/replay/*` output changes.

## Accept Change

- The change is intentional and tied to a requirement or bug fix.
- The new output preserves safety constraints and expected invariants.
- The team can explain the behavior delta in plain language.

## Reject Change

- The source of change is unknown.
- Output differs only intermittently across repeated runs.
- Risk-sensitive fields shift unexpectedly.

## Minimal Evidence to Keep with a Refresh

- What changed.
- Why it changed.
- Which command generated the new baseline.
- Which replay/golden checks passed after refresh.
