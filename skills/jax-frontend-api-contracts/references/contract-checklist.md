# Frontend API Contract Checklist

Run this checklist whenever API-facing behavior changes.

## Update Points

- `frontend/src/data/*.ts`: request construction and response normalization.
- `frontend/src/hooks/*.ts`: query keys, loading/error handling, derived state.
- `frontend/src/types/*.ts`: shared local types if data contracts changed.

## Validation Commands

- `cd frontend && npm run test -- --runInBand`
- `cd frontend && npm run test -- src/tests/integration/...`
- `cd frontend && npm run e2e` for route-level behavior changes

## Breakage Signals

- hook tests fail due to missing fields
- UI renders placeholders due to renamed contract fields
- backend returns shape that bypasses adapter invariants
