# Jax CR-02G Final Technical Closure

Status: **TECHNICAL CLOSURE COMPLETE — EXTERNAL REVIEW REQUIRED**

This record closes the bounded technical remediation requested after the prior
CR-02G NO-GO. It is not a legal opinion, licence grant, commercial-release
approval, or live-trading authorization.

## Repository

- Repository: `C:\Projects\Jax\jax-trading-assistant`
- Branch: `capability-reset`
- Closure base: `58fc30debd19ff2359fa8578269c78913d37f569`
- Remote before closure: `origin/capability-reset` at the same SHA
- No scientific evidence or sealed 2025 holdout was changed.

## Closure root causes and remediations

The prior technical closure was not release-clean for two concrete reasons:

1. Clean Windows checkouts inherited CRLF through global `core.autocrlf`,
   changing frozen-byte fixtures. Long repository paths also exceeded the
   default Windows checkout limit.
2. The pinned Go/Alpine/frontend/bridge images had stale or oversized runtime
   contents, and the frontend lockfile was not consumable by the Node 22/npm 10
   image used by Docker.

The canonical release-build environment is now:

- Git checkout: `core.longpaths=true`, `core.autocrlf=false`, short checkout
  path or Linux/WSL equivalent.
- Go: 1.26.8, pinned in CI and builder images.
- Node: 22, pinned in CI and the frontend builder; `npm ci` is required.
- Python bridge: pinned Python 3.11 image and requirements.
- Runtime images: digest-pinned Alpine 3.23/nginx 1.29 and current package
  upgrades during image construction.

## Clean-checkout proof

A tracked-only clone using the canonical Git settings completed Go tests/vet,
frontend `npm ci`, lint, typecheck, 155 Vitest tests, build and Playwright E2E.
The E2E suite starts its own Vite server and uses deterministic fixtures; it
does not require a pre-existing developer process or backend service.

The default Windows checkout failure was reproduced as a path-length/line-ending
configuration issue, then cleared with long-path support and LF checkout. The
canonical clone contained zero CRLF pairs in the tracked source/configuration
set. This establishes `CLEAN-CHECKOUT BUILD = PASS` under the documented
environment.

## Container rebuild and scans

All five retained local images were rebuilt with `--no-cache` from current
source. Final local image identities were:

| Image | Digest | Critical | High |
| --- | --- | ---: | ---: |
| trader | `sha256:72934298411ef4c1ed6f71f97cbf75ff5b49e14fa7aa3d22051bb013e28478ff` | 0 | 0 |
| research | `sha256:796c901654485fd5d270ac9465a43ccb4ef60fbbda1203491cd24b95812a08ba` | 0 | 0 |
| db-migrate | `sha256:0f6b3f009ff38265cc8357b6a6ec9b4cd8cb86bcdad827dcd386146eefc9384c` | 0 | 0 |
| frontend | `sha256:e1ab8c89ef8d15aec3344c528a3673e613c550edbe340b06d027d803198c200c` | 0 | 0 |
| IB bridge | `sha256:c178d094af72c827d5c32572a538381947e584302653de640bbafa2b753d8dac` | 0 | 1 |

Scanner: Docker Scout v1.24.0, commit `b1c9331b2166aef7ec690aa16fd655b8798ea4c6`.
The single IB-bridge high finding is Debian trixie `zlib` CVE-2026-85091;
Docker Scout reports no fixed version. The image was upgraded with current
Debian security packages and installer-only pip/setuptools/wheel were removed.
This is an explained upstream/no-fix release disposition, not an ignored
finding. Optional Prometheus, Grafana, Qdrant and World Monitor images are
outside the commercial image boundary and were not treated as shipped images.

The trader image no longer bundles the Go toolchain, module cache or source
tree for runtime trust-gate execution; those are development/test concerns.
This reduced the rebuilt image from approximately 932 MB to approximately
19 MB without changing execution authority.

## Frontend advisories

Runtime high/critical findings are zero. The full development graph remains at
one high (`vite`, major upgrade required) and one critical (`vitest`, major
upgrade required), with additional moderate findings. No safe bounded major
ecosystem upgrade was performed. This is explicitly dispositioned as a
development-tooling release follow-up; it is not a runtime vulnerability claim.

## Go/toolchain closure

The CI linter remains enabled at `golangci-lint` v2.13.2 via the existing pinned
GitHub Action. The exact v2.13.2 image digest used for local verification was
`sha256:ba07dffad130794ae79ebaa0056809d18c0168f3f846480ffd3eb6c04578b83d` and
reported `0 issues`. CI/workspace/toolchain references are aligned to Go 1.26.8.
The dependency update to `golang.org/x/crypto` v0.56.0 and `golang.org/x/net`
v0.57.0 removes the scanned application findings; Go module tidy is clean.

## SBOM and supply chain

`scripts/generate-sbom.ps1` now lists only the final supported image/base scope:
pgvector, the IB Python base, Go builder, Alpine runtime, Node builder and
nginx runtime. Excluded observability/database duplicates/Qdrant entries are
not represented as core shipped images. CycloneDX JSON remains the required
format and the pinned Syft image is unchanged. Runtime dependency changes are
limited to the security/toolchain updates recorded above; generated SBOMs and
scan reports remain local/ignored.

Actions remain SHA-pinned, lockfiles remain committed, Dockerfiles use digest
pins, and Docker build contexts exclude frontend local state. Mutable package
repository snapshots and hosted CI runner images remain explicit supply-chain
assumptions.

## Safety and commercial boundary

`ALLOW_LIVE_TRADING=false`, `BROKER_EXECUTION_ALLOWED=false`, execution workers
remain disabled and leverage remains 1x. No broker, paid provider, hosted AI or
live-account call was made. Planner/model providers remain advisory only; paper
and real portfolio domains remain isolated.

The commercial boundary remains core Jax trader/research/migration, frontend,
required configuration/migrations and the Postgres/pgvector contract. IB is a
separate supported bridge. World Monitor, observability, Qdrant, shadow tools,
private datasets, raw evidence, fixtures and archives remain optional/separate
or development-only. World Monitor is excluded from core distribution.

## Outstanding non-technical release decisions

- Root application licence: owner decision required.
- `gofinance/ib` LGPL/static-linking interpretation: formal legal review
  required.
- Provider data/API terms, Alpaca/IB/Polygon account terms and source-specific
  redistribution/retention rights: account/contract/legal review required.
- World Monitor distribution and upstream-source terms: separate review.
- Production secret-manager selection: deployment decision required.
- Frontend development advisories and the unfixed bridge zlib advisory require
  release-owner disposition.

These do not get silently converted into technical clearance.

## Verification summary

- Go tests: PASS.
- Go vet: PASS.
- Exact CI linter container: PASS, 0 issues.
- Frontend npm 22 clean install, lint, typecheck, 155 tests and build: PASS.
- Frontend E2E: PASS, deterministic local server/fixtures.
- Docker Compose config: PASS.
- No CRLF pairs under canonical checkout settings: PASS.
- Rebuilt retained image scans: four clean; one explained no-fix base finding.
- Pinned Gitleaks v8.30.1 current tracked-tree scan: no credential material;
  three document-only false positives were classified (one generic API-key
  phrase in `Docs/AI_SHADOW_HOSTED_A1.md` and two historical curl examples in
  `Docs/archive/runtime-history/phases/PHASE_4_COMPLETE.md`).
- Docker-based race verification remains previously accepted; native Windows
  cgo is not required for that accepted path.

## Remote CI proof

The exact final pushed SHA `8b5d6a33589a416cd4cd5fa9a5bb9717873267fb` was
validated by push workflow `34727057218` on branch `capability-reset`.
The Go job `103642921851` and Frontend job `103642921757` both succeeded;
Golden Tests `34727057249` and Import Boundary Enforcement `34727057264` also
succeeded. The integration job `103642922264` was intentionally skipped by
the workflow's manual-only policy. This closure record itself is the subject
of external review; no external GO is self-awarded.

## Exact status

- `UNEXPLAINED ACTIVE LEGACY DEPENDENCIES = 0`
- `UNEXPLAINED SHIPPED COMPONENTS = 0`
- `UNEXPLAINED THIRD-PARTY LICENCES = 0`
- `UNEXPLAINED DATA/API RIGHTS = 0`
- `UNEXPLAINED HIGH/CRITICAL RUNTIME VULNERABILITIES = 0`
- `CLEAN-CHECKOUT BUILD = PASS`
- `SAFETY BOUNDARIES = PRESERVED`
- `PHASE 13 IMPLEMENTATION = NOT STARTED`
- `Blocking CR-02G technical closure findings remaining: 0`
- `Adversarial CR-02G technical closure review: PASS`

CR-02G technical closure is complete locally, pending external review and the
owner/legal/deployment decisions above. Phase 13 remains
`NOT STARTED / BLOCKED BY EXTERNAL CR-02G REVIEW`.
