---
name: jax-frontend-api-contracts
description: Frontend-backend contract workflow for `frontend/src/data`, `frontend/src/hooks`, and API-facing UI modules. Use when adding or changing endpoints, response shapes, domain types, or integration behavior between React components and JAX backend services.
---

# Jax Frontend Api Contracts

Keep UI and API behavior aligned while preserving type safety and test coverage.

## Workflow

1. Start in data layer:
   - update `frontend/src/data/*` service calls and types first
2. Propagate through hooks:
   - update `frontend/src/hooks/*` selectors and state handling
3. Update consumers:
   - pages/components using the changed hook output
4. Run tests in this order:
   - affected `vitest` unit/integration tests
   - relevant `e2e` specs if route behavior changed

## Guardrails

- Preserve backward compatibility where possible in response adapters.
- Keep backend DTO drift isolated in `frontend/src/data/*` mappers.
- Avoid spreading raw HTTP shape assumptions through UI components.

## Required Validation

- Type check and lint in `frontend/`
- target test files for changed feature
- backend tests if endpoint contract changed

Use `references/contract-checklist.md` before final handoff.
