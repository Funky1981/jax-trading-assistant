# 12 — CI Pipeline (Keep it green)

**Goal:** Codex can wire this into GitHub Actions (or similar).

Steps:
1) `go test ./...`
2) `go test -tags=integration ./...` (optional; can run nightly)
3) lint
4) build docker images (optional)

Artifacts:
- test reports
- coverage report
- integration logs (when failing)
