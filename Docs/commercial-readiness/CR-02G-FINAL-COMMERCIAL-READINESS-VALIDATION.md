# Jax Commercial-Readiness CR-02G Final Validation

Status: **IMPLEMENTATION COMPLETE — EXTERNAL REVIEW REQUIRED**

This is a technical architecture and release-readiness record. It is not a
licence grant, legal opinion, data-rights clearance, or live-trading approval.

## Starting state

- Repository: `C:\Projects\Jax\jax-trading-assistant`
- Branch: `capability-reset`
- Starting HEAD: `6014fd068696353745a2a1255464a6ab64ba93d8`
- Starting worktree: clean
- Starting `origin/capability-reset`: `6014fd068696353745a2a1255464a6ab64ba93d8`
- Starting divergence: `0 ahead / 0 behind`
- No CR-02G push is authorised or performed.

The prior CR-02F exact-SHA workflow was green: run `34710966205`, Go job
`103599571371` SUCCESS, frontend job `103599571267` SUCCESS, and the manual-only
integration job `103599572119` SKIPPED. CR-02G changes are local and have no
remote CI result.

## Final architecture

```text
Frontend (React/nginx)
        |
        v
Jax trader (cmd/trader) ----> Jax research (cmd/research)
        |                              |
        v                              v
  paper/risk/HITL                 evidence/memory/planner
        |                              |
        +---- disabled execution ----> Jax-owned provider boundary
                                       |-- deterministic/offline
                                       |-- OpenAI adapter (explicit)
                                       |-- Ollama adapter (explicit/local)

PostgreSQL/pgvector + golang-migrate are the core persistence boundary.
IB bridge, observability, World Monitor and tools are separate or optional.
```

The current supported tree is a Go-led modular monolith with a React frontend,
an isolated Python IB bridge, Postgres/pgvector and explicit Compose profiles.
Research, recommendation, risk, approval and paper domains do not grant live
broker authority. The planner and all model adapters return advisory data only.

## Commercial distribution boundary

| Classification | Components | Rationale |
| --- | --- | --- |
| SHIP | `cmd/trader`, `cmd/research`, `cmd/db-migrate`, shared supported Go modules, frontend/nginx, required config/migrations | Core Jax application boundary |
| SHIP / separate | `services/ib-bridge`, PostgreSQL/pgvector contract | Supported integration/infrastructure; separate contract and rights review |
| OPTIONAL SEPARATE | Prometheus, Grafana, World Monitor, Qdrant/tools, shadow validator, local diagnostic tooling | Not required for core startup; profile- or operator-selected |
| DEVELOPMENT ONLY | Playwright, Storybook, fixtures, SBOM/security tools, reports, private datasets/raw evidence, local caches | Not a customer distribution input |
| ARCHIVE/HISTORY | `archive/**`, completed plans and historical evidence | Reproducibility/history, not built or shipped |

The entire Git repository is not the commercial product. Private HYP-EVENT data,
secrets and ignored local state are excluded.

## Removed/legacy dependency verification

Repo-wide active-tree inspection found no supported runtime/configuration seam
for Dexter, Agent0, LiteLLM, Finnhub, NewsAPI, Massive aliases, Redis, old
Financial Datasets runtime wiring, Anthropic, Gemini, Tavily, old AI gateway
names or old service URLs. Remaining hits are classified historical, archive,
test fixture, or compatibility evidence. The ignored local remnants of removed
projects are not tracked and are absent from a tracked-only checkout.

**UNEXPLAINED ACTIVE LEGACY DEPENDENCIES = 0**

## Runtime/service inventory

| Component | Owner/runtime | Distribution posture | Failure/safety posture |
| --- | --- | --- | --- |
| Trader | Jax / Go | Core | Requires explicit safe runtime policy; execution disabled |
| Research | Jax / Go | Core | Research mode; deterministic provider default |
| DB migration | Jax / Go + golang-migrate | Core one-shot | Unique immutable migration chain |
| PostgreSQL/pgvector | External image/runtime infrastructure | Required persistence | Development defaults are not production secrets |
| Frontend/nginx | Jax / React + nginx | Core UI | Browser receives endpoints only, never secrets |
| IB bridge | Separate Jax / Python | Separate supported bridge | Paper/development safety; no live execution authority |
| Prometheus/Grafana | Third-party images | Optional observability | No core startup dependency |
| World Monitor | Separate user-owned repository | Optional separate profile | Core starts without sibling checkout |
| Tools/Qdrant/shadow | Jax/third-party development | Development or optional | No core or execution authority |

## World Monitor boundary

Final posture: **EXCLUDED FROM CORE COMMERCIAL DISTRIBUTION**. It remains a
separate user-owned repository behind the `world-monitor` Compose profile. Jax
core does not require its sibling checkout, and the HTTP/page/cursor contract is
explicit. Missing or disabled World Monitor is represented as optional/degraded,
not healthy active data. Any separate distribution requires independent source,
licence and upstream-data-rights review.

## Observability boundary

Prometheus = **OPTIONAL EXTERNAL**. Grafana = **OPTIONAL EXTERNAL**. Both are
profile-gated and not application dependencies. Excluding them from a narrow
commercial bundle avoids implying that their images or dashboards are shipped;
image-layer notices and Grafana terms still require review if redistributed.

## AI provider architecture

CR-02C remains technically coherent: one Jax-owned inference boundary with a
deterministic/offline default, explicit OpenAI and local/Ollama adapters, and a
strict Jax planner contract. LiteLLM is not an active dependency. Hosted
providers are never selected implicitly. Embeddings remain a distinct
configuration/use case. Provider identity, usage and cost metadata are retained
where the transport exposes them; no hosted inference was called for CR-02G.

## Market/data provider architecture

| Provider | Final role | Status |
| --- | --- | --- |
| Interactive Brokers | Isolated account/market-data bridge; future paper/development compatibility | Retained, execution disabled |
| Alpaca | Approved HYP historical dataset/research source and optional fallback | Retained required approved role |
| Polygon | Canonical optional current market/earnings/news adapter | Retained optional role |
| Massive | Alias path | Removed active alias |
| Finnhub / NewsAPI | Duplicate fallback paths | Removed |
| Financial Datasets | Explicit research compatibility only | Not default runtime |
| SEC/EDGAR | Primary filing/evidence source | Retained |
| FRED/ALFRED, BLS, Treasury, CBOE | Official macro/evidence sources | Retained |
| EIA/CFTC | No current supported integration | Not present/deferred |
| Telegram | Optional operator notification | Retained optional |
| World Monitor | Separate user-owned event component | Separate profile |

Provider output preserves source identity, data/event timestamps, retrieval
metadata, adjustment semantics, raw/evidence hashes and dataset identity where
applicable. No silent fallback between incompatible bar, session, calendar,
adjustment or timestamp semantics is authorized.

## Credential and secret boundary

The root `.env` is local operator input and is ignored. `.env.example` contains
placeholders/defaults only. Compose passes a credential only to the service that
owns the capability: database credentials to database/Jax DSNs, IB settings to
the bridge, market-provider credentials to relevant acquisition paths, model
credentials to relevant Jax AI paths, and Telegram credentials to notification
delivery. The frontend receives only non-secret endpoint values.

Before/after reduction is recorded as: obsolete Dexter, Agent0, LiteLLM,
Finnhub, NewsAPI, Massive-alias, Redis and old gateway credential names removed;
only currently supported provider and service credentials remain in active
configuration. Exact values were not inspected, printed, committed or copied.

## Production secret-management contract

**PRODUCTION SECRET MANAGEMENT CONTRACT = PASS (contract defined); deployment
secret-manager selection = OWNER/DEPLOYMENT DECISION.** Production must inject
secrets at runtime, prohibit repository/build-layer/frontend exposure, use
least-privilege ownership, support rotation/revocation and environment
separation, redact logs/errors, and fail closed on unsafe placeholders. Root
Compose development defaults are explicitly not a production secret manager.

## Secret scan

Pinned Gitleaks `v8.30.1` was run against a tracked-only tree with redacted
output. No real credential was found. One current documentation match was a
false positive for the phrase “API-key presence”; two matches were historical
archived curl examples with redacted placeholders. No secret incident was
created and no value appears in this record.

## Third-party licence review

| Component | Shipped? | Distribution form | Licence/review status |
| --- | --- | --- | --- |
| Supported Go graph | Yes | Go binaries, module dependencies | CycloneDX metadata generated; notice review required for release |
| `github.com/gofinance/ib` | Yes if IB bridge/Go path shipped | Go linkage | LGPLv3/static-linking exception: **FORMAL LEGAL REVIEW REQUIRED** |
| React/MUI/Radix/TanStack runtime graph | Yes with frontend | Bundled JS | Lockfile/BOM captured; notice review required |
| IB bridge dependencies | Separate supported component | Python image/requirements | Pinned requirements and PyPI metadata captured; contractual IB review separate |
| Postgres/pgvector, Alpine, nginx | Infrastructure/runtime | Container images | Digests recorded; image notices/release review required |
| Prometheus/Grafana/Qdrant | No core bundle | Optional external | Excluded unless separately redistributed; review if shipped |
| World Monitor | No core bundle | Separate repository/image | Separate licence/source-rights review required |
| Jax root application | Yes | Application source/binaries | **OWNER DECISION REQUIRED**; no licence was invented |

## Root application licence decision

`LICENSE`, `COPYING` and root `NOTICE` are absent. The owner must choose among
proprietary/all-rights-reserved distribution, an explicit open-source licence,
or a dual-licensing/commercial model after legal advice. This is an owner/legal
decision, not a technical inference. Public repository visibility does not
itself grant reuse rights.

## Data/API commercial-rights review

| Provider/service | Actual flow/retention | Status |
| --- | --- | --- |
| IB | Credentialed bridge account/market data; account and contractual terms apply | ACCOUNT-SPECIFIC TERMS / FORMAL REVIEW REQUIRED |
| Alpaca | Local/private historical and current market data; HYP raw data excluded from distribution | ACCOUNT/CONTRACT REVIEW REQUIRED; BYO-account model is an owner decision |
| Polygon | Optional current market/event data and derived storage | ACCOUNT/CONTRACT REVIEW REQUIRED |
| SEC/EDGAR | Public filing evidence retained with provenance | Terms/automated-access and redistribution review required; not assumed unrestricted |
| FRED/ALFRED/BLS/Treasury/CBOE | Official-source macro/evidence adapters | Source-specific terms/attribution review required |
| OpenAI | Optional provider receives only explicitly submitted Jax input | ACCOUNT/CONTRACT REVIEW REQUIRED; no CR-02G calls |
| Telegram | Optional notification payloads | ACCOUNT/CONTRACT REVIEW REQUIRED |
| World Monitor/upstream sources | Separate component and source graph | EXCLUDED FROM CORE; separate review required |

Jax code can process data locally for a credential holder, but code paths also
persist/cache normalized data and expose selected derived/read-model results.
That distinction does not establish redistribution rights. A customer-supplied
credential/BYO-provider deployment can reduce Jax-held licensed data, but it is
an owner/business decision and does not automatically clear provider terms.

## Data-flow and redistribution review

The supported flow is `provider -> Jax acquisition/raw or evidence identity ->
normalized domain/read model -> research/API/frontend/report` with provenance
retained. Raw private HYP data and the sealed 2025 holdout are local-only and
not uploaded to AI providers or committed. Browser/API output is not a blanket
right to redistribute upstream raw data. Commercial release must define which
derived fields may be displayed/exported for each provider.

## Dependency audit

- Go tests: PASS; Go vet: PASS.
- `go.work` modules and root graph use the recorded Go 1.25.4 toolchain policy;
  no runtime dependency changed in CR-02G.
- Frontend runtime audit: 0 high / 0 critical, 4 moderate.
- Frontend full audit: 1 critical, 8 high, 12 moderate, 2 low in the current
  development graph; these are explicitly dispositioned as release/security
  follow-up, not silently ignored.
- Python bridge requirements remained unchanged and prior CR-02F pip-audit
  evidence remains applicable.
- `govulncheck`/`pip-audit` host binaries are not installed; prior pinned
  ephemeral results remain recorded in CR-02F. No new runtime dependency was
  introduced.

## SBOM coverage

CycloneDX JSON generation passed using the pinned Syft image
`anchore/syft:v1.51.1` with its recorded digest. The local generated reports
are ignored and were not committed. Coverage includes supported Go manifests,
frontend runtime lockfile and IB bridge requirements. Runtime dependency change
in CR-02G = **NONE**. Optional/external World Monitor is not represented as Jax
core SBOM content.

## Container/image vulnerability audit

Docker Scout `v1.24.0` was available. The scan results below are against local
images; the Go/research/frontend/db-migrate images were cached images rather
than newly reproducible final release builds because Docker module download
stalled in this environment. The IB bridge image was rebuilt from the current
tree. Results are release blockers to resolve or formally accept, not waived:

| Image | Identity/status | Critical | High | Disposition |
| --- | --- | ---: | ---: | --- |
| Jax trader | local cached image `sha256:af9e1ab...` | not completed | not completed | Release scan required |
| Jax research | local cached image `sha256:047b727...` | 11 | 14 | Base/application findings require remediation or release disposition |
| db-migrate | local cached image `sha256:7404b4...` | 3 | 6 | Findings require remediation or release disposition |
| frontend/nginx | local cached image `sha256:3b9dd3...` | 8 | 37 | Findings require remediation or release disposition |
| IB bridge | rebuilt local image `sha256:98ed2f...` | 5 | 12 | Findings require remediation or release disposition |
| Prometheus/Grafana/Qdrant/World Monitor | Excluded/optional | not release-scanned | not release-scanned | Separate review only if shipped |

This means container security is **BLOCKED for commercial release** and is not
represented as clean. No broad base-image upgrade was performed in CR-02G.

## Frontend advisory disposition

`FRONTEND RUNTIME HIGH/CRITICAL = 0`. The full development graph contains 1
critical and 8 high findings in tooling/transitive packages. They affect the
developer/build/test graph and require CR-02G/owner release disposition before
commercial release; no major framework upgrade was undertaken without a
separate bounded change.

## Supply-chain review

GitHub Actions are pinned by commit SHA; package lockfiles are committed; npm
CI uses `npm ci`; Go sums are committed; Python requirements are pinned; SBOM
tooling is pinned; active controlled image references use digests; Docker ignore
rules exclude `.env`, private data and reports. Hosted CI runner images and the
local World Monitor build tag remain explicit mutable/external assumptions.

## Toolchain consistency

Go directives, CI and controlled Docker builders use the accepted 1.25.4
toolchain policy. Frontend CI uses Node 20.x while the pinned frontend builder
uses Node 22-alpine; the split is operationally evidenced by both successful
local build/tests and the prior exact-SHA CI run, but should be unified before a
strict release reproducibility claim. The operator prerequisite remains Node
20+, and package-lock v3 is compatible with the tested environments. This is a
non-blocking cleanup follow-up, not a reason to change runtime dependencies in
this validation.

## Database/migration validation

Root Compose and migration configuration are consistent; the supported
golang-migrate stream remains unique and historical migrations are unchanged.
Core, observability, World Monitor, shadow overlay (combined with root) and
tools/vector Compose configurations parse successfully. No CR-02G migration was
added or altered.

## API/frontend contract validation

Frontend route/proxy tests and E2E fixtures exercise current Jax routes. No
active Agent0/Dexter/LiteLLM routes or stale gateway aliases were found. The
frontend cannot directly grant broker execution authority; approval/paper paths
remain bounded by backend safety policy.

## Clean-checkout reproducibility

The CR-02F exact pushed SHA proved the tracked clean-checkout Go/frontend CI
path. In this CR-02G run, `npm ci`, lint, typecheck, 155 Vitest tests, build and
44 E2E tests passed from the current tracked source without a pre-running
developer server; 7 runtime-only cases were intentionally skipped by the
existing fixture contract. Go test/vet passed from the current working tree.
A tracked-only Windows clone could not complete because one historical document
path exceeds the host filename-length limit; a tar archive checkout also exposed
platform line-ending sensitivity in frozen byte-hash tests. These are
reproducibility findings, not reasons to weaken evidence hashes. A complete
fresh Docker rebuild was attempted but Docker's Go module-download step stalled;
therefore **CLEAN-CHECKOUT BUILD = CONDITIONAL**, pending a path-length-safe,
network-capable release environment. Private ignored evidence is intentionally
required by some historical forensic tests and is not a commercial build input.

## Safety verification

Safety remains preserved: `ALLOW_LIVE_TRADING=false`,
`BROKER_EXECUTION_ALLOWED=false`, `EXECUTION_ENABLED=false`, execution worker
disabled, maximum leverage `1x`, IB paper/development posture, explicit HITL,
isolated paper domain and no broker/hosted-inference/paid calls. The 2025 HYP
holdout and accepted scientific artifacts were not modified.

## Files changed

CR-02G implementation changes are documentation/governance only:

- `Docs/commercial-readiness/CR-02G-FINAL-COMMERCIAL-READINESS-VALIDATION.md`
- `Docs/commercial-readiness/README.md`
- `Docs/ROADMAP.md`
- `Docs/Jax-Roadmap-v2/ROADMAP-DECISION-LOG.md`
- `Docs/SETUP/QUICKSTART.md` (only if the final local diff records the toolchain
  prerequisite correction)

No runtime provider, market, AI, database, frontend source, migration or
scientific-data file was changed by CR-02G.

## Owner/legal/deployment decisions required

1. Root Jax application licence and notice model — OWNER DECISION / LEGAL.
2. `gofinance/ib` LGPL/static-linking exception — FORMAL LEGAL REVIEW.
3. Provider contracts/data retention/redistribution and BYO credentials —
   ACCOUNT/CONTRACT REVIEW.
4. World Monitor separate distribution and upstream-source terms — OWNER/LEGAL.
5. Production secret-manager selection, rotation and deployment model —
   DEPLOYMENT DECISION.
6. Container HIGH/CRITICAL findings and pinned-base remediation — TECHNICAL /
   RELEASE DECISION.
7. Frontend development advisory upgrade/disposition — TECHNICAL / RELEASE
   DECISION.

These decisions block legal/operational commercial release even when the
technical cleanup architecture is coherent.

## Cleanup / architecture gate

**CONDITIONAL**. The post-cleanup architecture is coherent, bounded and has no
unexplained active legacy dependency. The gate remains conditional on external
review of the explicit container/build reproducibility findings and the
owner/legal boundary decisions; no hidden dependency was found.

## Commercial release gate

**BLOCKED**. Root application licensing, LGPL interpretation, provider/data
rights, production secret deployment, container vulnerability disposition and
frontend development advisories are not all cleared.

## Adversarial review

The final review attacked legacy references, optional-service coupling, browser
secret exposure, provider fallback identity, data rights, image findings, SBOM
scope, clean-checkout assumptions, AI/execution authority and Phase-13 creep.

- UNEXPLAINED ACTIVE LEGACY DEPENDENCIES = 0
- UNEXPLAINED SHIPPED COMPONENTS = 0
- UNEXPLAINED THIRD-PARTY LICENCES = 0 (review-required items are explicit)
- UNEXPLAINED DATA/API RIGHTS = 0 (review-required items are explicit)
- UNEXPLAINED HIGH/CRITICAL RUNTIME VULNERABILITIES = 0 (all observed findings are recorded; release disposition remains open)
- CLEAN-CHECKOUT BUILD = CONDITIONAL
- SAFETY BOUNDARIES = PRESERVED
- PHASE 13 IMPLEMENTATION = NOT STARTED
- **Adversarial CR-02G review = PASS**

## Exact CR-02G status

- CR-02A = GO
- CR-02B = GO
- CR-02C = GO
- CR-02D = GO
- CR-02E = GO
- CR-02F = GO
- CR-02G = **IMPLEMENTATION COMPLETE — EXTERNAL REVIEW REQUIRED**
- Phase 13 = **NOT STARTED / BLOCKED BY EXTERNAL CR-02G REVIEW**
- Scientific status: **TRADING EDGE NOT DEMONSTRATED / INSUFFICIENT SAMPLE**
- HYP-EVENT-001A: **NOT VALIDATED**
- Actual forward-paper evidence: **0 DAYS / 0 ORDERS**

## Recommended external decision

**CONDITIONAL GO CR-02G**

This is a recommendation for external review, not a self-awarded GO. Phase 13
must remain blocked until the external technical lead reviews the conditional
architecture/release evidence and separately resolves the owner/legal/deployment
items.
