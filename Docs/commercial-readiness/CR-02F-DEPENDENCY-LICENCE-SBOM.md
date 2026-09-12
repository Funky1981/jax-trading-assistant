# Jax Commercial-Readiness CR-02F — Dependency, Licence & SBOM

Status: **FINAL CI CLOSURE COMPLETE — EXTERNAL REVIEW REQUIRED**

This is a technical inventory and reproducibility record. It is not a legal
opinion, a data-rights clearance, or approval for live trading.

## Starting state

- Branch: `capability-reset`.
- Starting HEAD: `5a3dd89264eed873e49846928e90f26476cd9b53`.
- Starting worktree and `origin/capability-reset`: clean and equal.
- Prior accepted cleanup: CR-02A through CR-02E.
- Safety remains `ALLOW_LIVE_TRADING=false`,
  `BROKER_EXECUTION_ALLOWED=false`, live worker disabled and maximum leverage
  `1x`.
- Scientific status remains `TRADING EDGE NOT DEMONSTRATED / INSUFFICIENT
  SAMPLE`; forward-paper evidence remains `0 DAYS / 0 ORDERS`.

## Distribution boundary

The supported product boundary is deliberately separated:

| Boundary | Contents | Status |
| --- | --- | --- |
| Runtime application | `cmd/trader`, `cmd/research`, `cmd/db-migrate`, shared Go modules and versioned config | Supported Jax runtime |
| Frontend | React/Vite application and nginx runtime image | Supported UI |
| IB bridge | `services/ib-bridge`, pinned Python requirements and Python image | Separate supported bridge; broker execution remains disabled |
| Database | PostgreSQL/pgvector image and migrations | Runtime infrastructure |
| Observability | Prometheus/Grafana profile | Optional |
| World Monitor | Sibling user-owned repository and its image | Separate component; not part of the Jax core SBOM |
| Development/test | Go workspace tooling, Playwright, Storybook, test dependencies and shadow validator | Not shipped as runtime |
| Archive | `archive/**` and historical documents | Historical/reproducibility record, not supported runtime |

Private HYP-EVENT datasets, secrets and ignored local state are excluded from
distribution and SBOM inputs.

## Dependency ecosystem inventory

- Go workspace: root plus the modules declared in `go.work`; archive modules
  are historical and excluded from the supported inventory.
- Frontend: `frontend/package.json` and the committed
  `frontend/package-lock.json`.
- Python: `services/ib-bridge/requirements.txt`; there is no separate runtime
  requirements file for development dependencies.
- Containers: active Dockerfiles, root Compose images, optional profiles and
  specialized shadow/tools Compose files.
- CI/build: pinned GitHub Action SHAs, Go 1.25.4, npm lockfile installation,
  PowerShell scripts, migration tooling and the pinned Syft SBOM tool.
- No tracked `vendor/` directory, copied Agent0/Dexter source, or unknown
  generated SDK source remains in the supported tree.

## Go dependency inventory

The root graph and supported workspace modules are locked by `go.mod`,
module-specific `go.mod`/`go.sum` files and `go.work.sum`. Runtime ownership
includes PostgreSQL/pgx, golang-migrate, Alpaca market data, Polygon client
types, SEC/HTTP support, JWT authentication, decimal arithmetic, validation,
resilience and observability. Direct and transitive components are enumerated
by the generated CycloneDX files under the ignored `reports/sbom/` directory.

The supported module set is reproducible under Go language target `1.25.0`
and the pinned build/CI toolchain `1.25.4`. Every supported module was checked
with `go mod tidy -diff`; the standalone ingest module now carries its local
replacements for all local module imports.

## Redis verdict

`REDIS GO DEPENDENCY = REMOVE`.

The active Redis cache implementation and `github.com/redis/go-redis/v9` were
removed in the prior bounded CR-02F commit. No supported source, registry,
Compose service or runtime configuration requires Redis. Historical references
remain only where needed to explain the superseded architecture.

## Go toolchain decision

| Concern | Decision |
| --- | --- |
| Go language compatibility target | `1.25.0` for modules whose fixed security dependency graph requires it; older compatible leaf modules retain their lower language directive |
| Build toolchain | Go `1.25.4` |
| CI toolchain | Go `1.25.4` |
| Workspace directive | `go.work` uses `1.25.4` |
| Reproducibility | `go.sum`, `go.work.sum`, exact toolchain and tidy checks |

The former 1.24/1.25 drift was resolved deliberately because the fixed
security versions of the reachable Go graph require the newer language target.

## Frontend dependency inventory

The committed lockfile is the reproducibility source and CI uses `npm ci`.
Runtime and development dependencies remain separated in `package.json`.
`npm sbom --package-lock-only --omit=dev --sbom-format cyclonedx` produced a
CycloneDX runtime BOM with 204 components and license metadata. No broad
framework upgrade was performed.

`npm audit --omit=dev` reported four moderate advisories and no high or critical
advisories in the current runtime graph. They are recorded for the next bounded
frontend upgrade/security review; no production-breaking major upgrade was
introduced during CR-02F.

## Python / IB bridge dependency inventory

The runtime requirements are pinned:

| Package | Version | Role / licence evidence |
| --- | --- | --- |
| `ib-insync` | `0.9.86` | IB client; BSD metadata |
| `fastapi` | `0.141.1` | HTTP API; MIT metadata |
| `uvicorn[standard]` | `0.52.4` | ASGI server; BSD metadata |
| `pydantic` | `2.13.5` | Validation; MIT metadata |
| `pydantic-settings` | `2.15.0` | Configuration; MIT metadata |
| `websockets` | `17.1` | WebSocket support; BSD-3-Clause metadata |
| `python-dotenv` | `1.2.3` | Local configuration loading; BSD-3-Clause metadata |
| `tzdata` | `2026.4` | Time-zone data; Apache-2.0 metadata |

All listed packages are imported by the bridge or required by its pinned
runtime. A pinned Python 3.11 image installed the requirements and passed all
13 bridge unit tests. `pip-audit` 2.10.1 reported no known vulnerabilities for
the resolved requirements.

The IB API/account contractual and commercial terms remain separate from the
open-source package licences and require the CR-02D data-rights review.

## Container/image inventory

Active images and base images are pinned to immutable digests wherever the
repository controls the reference:

| Image family | Use | Identity |
| --- | --- | --- |
| `golang` | Go builders | `1.25.4-alpine` plus verified digest |
| `alpine` | Go runtime | `3.19` plus verified digest |
| `python` | IB bridge | `3.11-slim` plus verified digest |
| `node` | Frontend builder | `22-alpine` plus verified digest |
| `nginx` | Frontend runtime | `1.27-alpine` plus verified digest |
| `pgvector/pgvector` | Core database | `pg16` plus verified digest |
| `postgres` | Shadow/tools/World Monitor databases | `16`/`16-alpine` plus verified digests |
| `prom/prometheus` | Optional observability | Verified digest |
| `grafana/grafana` | Optional observability | Verified digest; licence review remains required |
| `qdrant/qdrant` | Optional tools profile | Verified digest |
| World Monitor image | Separate profile | Local sibling build tag; outside Jax core SBOM |

The active image references contain no unsupported floating `latest` tag except
the intentionally local-built World Monitor integration image. Archive
Dockerfiles retain historical tags and are not supported deployment inputs.
Registry image SBOM enrichment was attempted with the pinned Syft tool; the
large Grafana registry scan stalled on this host before completion. Therefore
the source-dependency SBOM set is complete for the supported manifests, while
complete per-layer image SBOM coverage remains an external review/deployment
follow-up.

## CI/build dependency inventory

- CI action references are pinned to verified commit SHAs for checkout,
  setup-go, setup-node, cache, artifact upload and golangci-lint.
- CI installs the exact Go 1.25.4 toolchain and uses `npm ci`.
- `capability-reset` push coverage is present for the active CI workflows.
- Dockerfiles use pinned base digests, multi-stage builds and non-root runtime
  users where applicable. No secret is copied as a build input.
- `Dockerfile.shadow-validator` contains a historical shell-generation step and
  remains specialized validation, not a product runtime.

## Embedded source audit

No supported `vendor/` tree or unexplained copied third-party source was found.
`frontend/node_modules`, local Python environments, Go caches, reports, logs,
private datasets and build outputs are ignored and excluded from Docker build
contexts. Archived repositories and old provider source under `archive/**` are
historical, not shipped runtime dependencies.

## Application licence posture

`ROOT APPLICATION LICENCE = OWNER DECISION REQUIRED`.

No root `LICENSE`, `COPYING` or `NOTICE` was invented or added. The owner/legal
release process must choose the Jax application licence and decide the exact
distribution notice package before external distribution.

## Third-party licence matrix

| Area | Representative shipped components | Licence posture |
| --- | --- | --- |
| Go runtime | pgx, golang-migrate, JWT, Alpaca SDK, Polygon client, validator, resilience/telemetry libraries | Metadata captured by Syft where available; permissive licences recorded in the machine BOM |
| IB Go client | `github.com/gofinance/ib` | LGPLv3 with the upstream static-linking exception; formal licence review required before distribution |
| Frontend | React, MUI, Emotion, Radix, TanStack, Vite runtime graph | CycloneDX lockfile BOM contains license metadata; review copyleft/custom entries before resale |
| Python bridge | FastAPI, Uvicorn, Pydantic, ib-insync and pinned transitive packages | PyPI metadata and Python BOM captured; IB contractual terms remain separate |
| Containers | PostgreSQL/pgvector, Alpine, nginx, Prometheus, Grafana, Qdrant | Base image/source identities pinned; image-layer notice and Grafana terms require release review |

## Unknown/review-required licences

There are no intentionally hidden unknown third-party licences in the recorded
runtime inventory. Review-required items are explicit rather than guessed:

- Jax root application licence: owner decision required.
- `gofinance/ib` LGPL/static-linking exception: formal legal review required.
- Optional Grafana image and bundled image-layer notices: formal licence review
  required before commercial redistribution.
- Any World Monitor distribution: separate component owner/licence review.

## Vulnerability audit

| Ecosystem | Tool/scope | Result |
| --- | --- | --- |
| Go | pinned `govulncheck` 1.1.4; supported production packages | PASS: 0 reachable vulnerabilities after updates; 18 module-only non-reachable advisories remain outside called code |
| Python | pinned `pip-audit` 2.10.1 against resolved bridge requirements | PASS: no known vulnerabilities |
| Frontend | `npm audit --omit=dev` | 0 high, 0 critical; 4 moderate advisories retained for bounded follow-up |
| Containers | Syft SBOM attempted; no container vulnerability scanner was installed on the host | NOT FULLY ASSESSED: image vulnerability scan remains deployment review work |

The Go scan tool and Python audit tool were used ephemerally and are not
runtime dependencies. Current official project references are [Syft
releases](https://github.com/anchore/syft/releases) and [Go vulnerability
tooling](https://github.com/golang/vuln). No paid service was used.

## SBOM format/tool/coverage

Preferred format is CycloneDX JSON. The reproducible SBOM set uses the pinned
Syft image `anchore/syft:v1.51.1` with its recorded digest plus npm's native
CycloneDX generator. It covers:

- each supported Go module manifest with Go enrichment;
- frontend runtime lockfile dependencies;
- IB bridge Python requirements;
- optional pinned image references when registry scanning is available.

Separate BOMs avoid pretending that the external World Monitor component is
part of the Jax core. Generated BOMs are ignored local reports and are not
committed with private data or credentials.

## SBOM regeneration

From the repository root:

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/generate-sbom.ps1
```

This writes standard CycloneDX JSON files to ignored `reports/sbom/`, validates
their `bomFormat`, scans every supported Go module plus the Python manifest and
frontend lockfile, and records the pinned Syft image in the command source.
Use `-IncludeImages` only in an environment with bounded registry access; image
scans are optional because the external World Monitor image is locally built.

## Third-party notices

`THIRD_PARTY_NOTICES.md` records direct runtime components, upstream links,
licence evidence and explicit review flags. The complete transitive inventory
is generated from the committed lock/manifests rather than hand-maintained.
Licences that require text/notice redistribution must be packaged by the
future release process; this phase does not claim legal sufficiency.

## Data/API terms distinction

Software licences do not clear provider data/API terms. CR-02D flags remain in
force for Alpaca, Polygon, Interactive Brokers, SEC/EDGAR, World Monitor and
any retained or derived data. Raw private HYP-EVENT data and its sealed 2025
holdout were not modified.

## Secret-management assessment

`.gitignore` and `.dockerignore` exclude `.env`, private datasets, reports and
local state. The frontend receives endpoints only; no secret is compiled into
browser assets. Dependency/SBOM/security commands were run without Jax
credentials, provider secrets or proprietary research data. Exact secret
values were never printed or committed.

`PRODUCTION SECRET MANAGEMENT = DEPLOYMENT REQUIREMENT`: a deployment-specific
secret manager/rotation policy must be selected before external production
distribution. No vault or cloud secret service was introduced here.

## Dependencies removed

- Redis cache implementation and `github.com/redis/go-redis/v9`.
- Tracked generated binaries and logs with no supported consumer.
- No historical migration, scientific dataset, archive record or live-execution
  path was deleted.

## Dependencies updated

- Fixed reachable Go advisories by updating JWT, pgx, x/net and x/text, with
  compatible x/crypto/x/sys/x/sync graph updates.
- Updated pinned IB bridge framework/runtime packages to versions with no
  known pip-audit findings.
- Updated build reproducibility to Go 1.25.4, exact action SHAs and digest
  pinned images.

## Outstanding legal/owner review

- Select and record the Jax application licence.
- Review LGPL/static-linking terms for `gofinance/ib`.
- Review Grafana/image-layer notices and optional observability distribution.
- Review CR-02D commercial data/API terms and World Monitor separation.
- Run a container vulnerability scanner in the release/deployment environment.
- Review the four moderate frontend advisories and determine an upgrade window.

## Remaining CR-02G items

CR-02G must perform final commercial-readiness validation, including owner/legal
licence decisions, complete release-artifact SBOM/image scanning, data/API
rights decisions, production secret management, and final distribution review.
Phase 13 remains `NOT STARTED / BLOCKED BY CLEANUP GATE`.

## Final CI closure

The original authoritative push run for `e1431482eee9e5fe2e88f293db21c4ffa6b0f47b`
was red in Go job `103572257922` and frontend job `103572258042` (workflow run
`34700848786`). The Go failure was the Go-1.25-incompatible `golangci-lint`
v1.64.8 configuration; the frontend failure was stale E2E assumptions and
missing deterministic shell fixtures, not Vite readiness. The corrective
commits are `3457edcbd8902d135d699ddf824b0f79e0f3ab37` (pinned
golangci-lint-action v7.0.1 with golangci-lint v2.13.2 and explicit workspace
coverage) and `40deb188aeb87231000988f5dd603290cc964e32` (frontend E2E
fixtures and current route/UI contracts). The final exact-SHA workflow result
is recorded in the CR-02F closure handover.

The linter is a development/CI tool only: `RUNTIME SBOM IMPACT = NONE`.
No runtime dependency, lockfile, research artifact or safety boundary was
changed by this closure.

### Clean-checkout CI closure

The first post-remediation exact-SHA run (`34710136479` on
`18b8669c2405c5905c3f2ab203d45b0b8297009d`) reached the corrected linter and
frontend E2E successfully, but exposed that several legacy forensic tests
assumed private, ignored `.runtime` and `data/datasets` artifacts were present
in every checkout. Those artifacts are intentionally local and are not part
of the public repository. Commit `b1bac5416e028f587b36f0fea043587a874a4e40`
added narrow prerequisite guards: the tests execute their original assertions
when the private evidence is mounted, and explicitly skip only when that
private evidence is absent from a clean checkout. Other errors remain fatal.

The exact final push run for `b1bac5416e028f587b36f0fea043587a874a4e40` is
`34710567041`: Go job `103598475052` **SUCCESS**, frontend job
`103598475151` **SUCCESS**, and the intentionally manual-only integration job
`103598475620` **SKIPPED**. The run is a push on `capability-reset` for that
exact SHA. No hosted inference, broker call, paid service, runtime dependency,
lockfile, research artifact, or safety boundary changed.

## Roadmap state

- CR-02A = `GO`
- CR-02B = `GO`
- CR-02C = `GO`
- CR-02D = `GO`
- CR-02E = `GO`
- CR-02F = `FINAL CI CLOSURE COMPLETE — EXTERNAL REVIEW REQUIRED`
- CR-02G = `NOT STARTED`
- Phase 13 = `NOT STARTED / BLOCKED BY CLEANUP GATE`
