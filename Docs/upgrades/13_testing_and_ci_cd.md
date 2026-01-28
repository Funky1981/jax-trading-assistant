# Testing and Continuous Integration/Continuous Deployment (CI/CD)

A comprehensive testing strategy and automated pipeline ensure that changes do not break existing functionality and that deployments are repeatable and safe.

## Why it matters

Relying on manual tests invites regressions. Automated testing and CI/CD catch issues early, enforce code quality and speed up delivery without compromising stability.

## Tasks

1. **Unit testing**
   - Add table‑driven tests for domain and application layers (`internal/domain`, `internal/app`). Cover edge cases and error paths.
   - Use mocks for external dependencies (market data provider, storage, Dexter, broker) to isolate components.
   - Integrate Go tools like `go test`, `sqlmock`, and `httptest` for thorough coverage.

2. **Integration and end‑to‑end testing**
   - Write tests that spin up services together via Docker Compose (e.g. `jax-api`, `jax-ingest`, `jax-memory`, Postgres). Use sample data to simulate real workflows.
   - For the front‑end, use Cypress or Playwright to run complete user journeys: processing a symbol, inspecting research, adjusting risk settings and approving a trade.

3. **Static analysis and linting**
   - Configure `golangci-lint` with a standard configuration to check for common mistakes (unused code, concurrency issues, error handling). Run it in CI.
   - Use `go vet` and `staticcheck` for additional checks.
   - For the front‑end, use ESLint and Prettier with TypeScript rules.

4. **CI pipeline**
   - Use GitHub Actions or another CI platform to run the following on every pull request:
     - `go test ./...` for all backend services.
     - `golangci-lint run` for static analysis.
     - Build and test the front‑end with `npm run test` or equivalent.
     - Run unit and integration tests inside Docker when changes affect services.
   - Fail the build on lint errors or test failures.

5. **CD pipeline**
   - On merge to `main`, build container images for each service and push them to a registry.
   - Deploy images to staging and run smoke tests. If tests pass, promote to production.
   - Use infrastructure‑as‑code (e.g. Terraform, Helm charts) to manage deployments.

6. **Quality gates**
   - Define thresholds for code coverage and enforce them in CI.
   - Require at least one peer review on each pull request. Automate checks for unreviewed changes or failing status checks.

7. **Documentation**
   - Document how to run tests locally (`scripts/test.ps1`, `make test`) and how to troubleshoot failing pipelines.
   - Provide guidelines for writing new tests and adding them to the suite.