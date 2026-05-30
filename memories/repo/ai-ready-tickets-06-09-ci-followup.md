# AI-ready tickets 06-09 CI follow-up

Date: 2026-05-30

## Findings

- PR #11 was open from `redesign` to `main`; local `redesign` matched `origin/redesign`.
- Latest CI on commit `51b9fda` failed in:
  - `go`: golangci-lint v1.61.0 was built with Go 1.23 while the repo targets Go 1.24.
  - `frontend`: two Playwright strict-mode locator collisions in AI Trading and Notifications specs.
  - `golden-tests`: Docker frontend build failed because npm 10.9.8 required `frontend/package-lock.json` to include `tailwindcss` peer/optional `yaml@2.9.0`.
- After bumping golangci-lint to v1.64.8, lint surfaced three existing findings in chat/trader packages; these were corrected with minimal lint-only edits.

## Validation evidence

- `npx npm@10.9.8 ci --prefer-offline` passes in `frontend/`.
- `go run github.com/golangci/golangci-lint/cmd/golangci-lint@v1.64.8 run` passes.
- `go test ./...` passes.
- `npm run lint`, `npm run typecheck`, `npm run test`, and `npm run build` pass in `frontend/`.
- `npm run test:e2e` exits successfully; one module guide spec retried once and then passed.

## Residual risk

- Local Docker verification could not run because Docker Desktop Linux engine was unavailable. The lockfile failure was reproduced and fixed locally with npm 10.9.8, matching the CI/Docker npm version.
