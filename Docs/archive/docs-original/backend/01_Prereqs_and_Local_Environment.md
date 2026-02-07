# 01 — Prereqs & Local Environment

**Goal:** get a stable local dev environment so Codex can build and run tests consistently.

## Required tools
- Git
- Docker + Docker Compose
- Go (latest stable)
- Node.js (LTS) — only if you are building a UI in this repo
- Make (optional but recommended)

## Repo conventions
- One service per folder in `/services`
- Shared libs in `/libs`
- All docs in `/Docs`
- Test naming: `*_test.go`
- Use `go test ./...` as the baseline test command

## Baseline commands
- `make test` -> runs unit tests
- `make lint` -> runs lint
- `make up` -> starts docker compose
- `make down` -> stops docker compose

## Testing rules (start now)
- If Codex adds a new package, it must include at least:
  - one unit test
  - one table-driven test case
- Avoid tests that require live internet calls.

