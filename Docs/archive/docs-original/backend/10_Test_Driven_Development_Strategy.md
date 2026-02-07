# 10 — TDD Strategy (How we keep this system sane)

**Goal:** enforce a test taxonomy that Codex can follow consistently.

## 10.1 — Test layers
- Unit tests (default):
  - fast
  - deterministic
  - no network, no docker
- Contract tests:
  - validate tool payloads and response shapes
- Integration tests (`-tags=integration`):
  - spin docker compose
  - prove service-to-service wiring
- Optional end-to-end tests:
  - run orchestrator against fixtures

## 10.2 — “Write the test first” rule

For each new capability:
1) Add failing unit test
2) Implement minimal code to pass
3) Refactor
4) Add edge cases

## 10.3 — Logging to memory is testable

For every decision pipeline test, assert:
- a MemoryItem was produced
- item validates schema
- item contains summary + tags + type + ts
- item does NOT contain secrets

## 10.4 — Fixtures

Put fixtures in:
- `libs/testing/fixtures/...`
Use:
- golden JSON files
- deterministic timestamps (inject clock)

## 10.5 — Definition of Done
- `go test ./...` green
- `go test -tags=integration ./...` green (if enabled)
- Coverage targets (start with 60–70%, raise later)
