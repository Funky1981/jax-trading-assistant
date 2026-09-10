# Jax Commercial-Readiness CR-02E — Configuration & Compose Consolidation

Status: **IMPLEMENTATION COMPLETE — EXTERNAL REVIEW REQUIRED**

This document records the bounded CR-02E cleanup. It is an operational
configuration contract, not a legal, licensing or live-trading approval.

## Starting state

- Branch: `capability-reset`
- Starting HEAD: `8521dd8f4dbfbe1dfcab422f8d08a4346e00e6bf`
- Starting worktree: clean; starting local and `origin/capability-reset` equal.
- Prior accepted cleanup: CR-02A/02B/02C/02D.
- `ALLOW_LIVE_TRADING=false`, `BROKER_EXECUTION_ALLOWED=false`, live worker
  disabled and maximum leverage `1x` remain mandatory.
- Scientific status remains **TRADING EDGE NOT DEMONSTRATED / INSUFFICIENT
  SAMPLE**. Actual forward-paper evidence remains `0 DAYS / 0 ORDERS`.

## Configuration inventory

The supported configuration surfaces are now:

| Surface | Ownership | Status |
| --- | --- | --- |
| Root `.env` | Local operator secrets and Compose interpolation; ignored and never committed | Authoritative local input |
| `.env.example` | Safe, non-secret operator contract | Canonical example |
| Root `docker-compose.yml` | Container wiring, development defaults and service profiles | Supported Compose |
| `cmd/trader` / `cmd/research` environment readers | Process-level runtime contract | Active |
| `frontend/.env.example` and Vite `VITE_*` readers | Browser build-time endpoints only | Active, non-secret |
| `config/providers.json` | Jax provider capability registry | Active |
| `config/providers.local.json` | Ignored local override | Optional local/test compatibility; no Dexter entry |
| `config/*.json` strategy/risk/evidence files | Versioned domain/research artifacts | Active or reproducibility-scoped; not a second env system |
| `docker-compose.shadow.yml` | ADR-0012 shadow validation | Specialized test |
| `tools/docker-compose.yml` | Knowledge-ingest development tools; Qdrant remains profile-gated | Specialized optional |
| archived Compose/docs | Historical evidence only | Not supported runtime |

The root `.env` remains user-managed. Its contents were not copied or
rewritten. Unsupported legacy names in that ignored file are not read by the
supported runtime; active example/Compose/configuration seams were removed.

## Canonical configuration ownership

For containers, Docker Compose loads the root `.env` and interpolates the
canonical variables into service environments. For direct process execution,
the same variable names are supplied by the shell or an established script.
There is no second gateway or secret-management system.

Service-specific Compose selectors `JAX_TRADER_RUNTIME_MODE` and
`JAX_RESEARCH_RUNTIME_MODE` are intentionally distinct inputs. Compose maps
each to the process-owned `JAX_RUNTIME_MODE`. A direct process uses
`JAX_RUNTIME_MODE` directly. This avoids silently sharing trader and research
mode while preserving the runtime policy's fail-closed semantics.

## Runtime modes

| Setting | Owner | Safe default / rule |
| --- | --- | --- |
| `JAX_TRADER_RUNTIME_MODE` | Compose trader selector | `paper` |
| `JAX_RESEARCH_RUNTIME_MODE` | Compose research selector | `research` |
| `JAX_RUNTIME_MODE` | Runtime safety policy | Explicitly set; direct-process default is not a production posture |
| `JAX_REQUIRE_EXPLICIT_RUNTIME_MODE` | Both runtimes | `true` in supported Compose |
| `ALLOW_LIVE_TRADING` | Trader live boundary | `false`; never enabled by cleanup |
| `BROKER_EXECUTION_ALLOWED` | Broker execution boundary | `false` |
| `EXECUTION_ENABLED` | Execution service construction | `false` |
| `EXECUTION_INSTRUCTION_WORKER_ENABLED` | Execution worker | `false` |
| `MAX_LEVERAGE` | Safety projection/risk policy | `1` |
| `IB_PAPER_TRADING` | IB bridge/execution guard | `true` |

Future execution would require an independently authorized phase and explicit
coherent safety configuration; CR-02E cannot enable it.

## Compose service inventory

| Service | Classification | Reason |
| --- | --- | --- |
| `postgres` | CORE REQUIRED | Jax application database and vector-capable Postgres |
| `db-migrate` | CORE ONE-SHOT | Applies the authoritative golang-migrate chain |
| `ib-bridge` | CORE REQUIRED for current trader stack | Isolated IB account/market-data boundary; execution remains disabled |
| `jax-trader` | CORE REQUIRED | Trader runtime and frontend API |
| `jax-research` | CORE REQUIRED | Research/orchestration, memory and Jax-native planner |
| `frontend` | CORE UI | Supported dashboard; can also be run independently in development |
| `prometheus` | OPTIONAL OBSERVABILITY | Profile `observability`; not a startup dependency |
| `grafana` | OPTIONAL OBSERVABILITY | Profile `observability`; not a startup dependency |
| `worldmonitor-events` | OPTIONAL / EXTERNAL COMPONENT | Profile `world-monitor`; built from sibling user-owned checkout |
| `worldmonitor-postgres` | OPTIONAL / EXTERNAL COMPONENT | Profile `world-monitor`; storage for the sibling component |

Core `docker compose up` no longer selects the observability or World Monitor
profiles. The ordinary core topology therefore does not require the sibling
World Monitor checkout. `docker compose --profile world-monitor up` is the
explicit integration arrangement. World Monitor pulling is disabled by default
and requires both an explicit `WORLD_MONITOR_EVENTS_URL` and enabled policy.

## Optional services

Observability is enabled with `--profile observability`; World Monitor is
enabled with `--profile world-monitor`. Both combinations and the combined
profile arrangement pass Compose validation. The root startup script starts
core services only; `.\start.ps1 -Observability` opts into dashboards.

## World Monitor deployment model

World Monitor remains a separate user-owned project at the existing versioned
HTTP/page contract. Jax retains its pull worker and durable ingestion boundary,
but does not copy or build the sibling component in the default core topology.
Unavailable optional World Monitor is a disabled/degraded integration, not a
false healthy active capability. The separate database URL is owned by the
World Monitor profile and is not used by Jax's database.

## Observability deployment model

Prometheus and Grafana are operationally useful but not core application
dependencies. They are profile-gated. The stale `jax-api` scrape target was
removed; current targets are `jax-trader` API/internal metrics and
`jax-research` metrics. Floating image tags remain a CR-02F reproducibility and
SBOM review item.

## Database configuration

`DATABASE_URL` is the single Jax application/migration connection variable.
Root Compose uses it consistently for `db-migrate`, `jax-trader` and
`jax-research`, with a clearly development-only fallback derived from the
Postgres variables. `POSTGRES_USER`, `POSTGRES_PASSWORD` and `POSTGRES_DB`
configure the root Postgres service. Production deployments must supply their
own safe values; the development fallback is not a production secret policy.

The former standalone `db/postgres/docker-compose.yml` stack was removed
because it used the conflicting `jaxdb`/`jaxuser`/port-5432 contract. `Makefile`,
README and migration instructions now target root Compose on host port 5433.
`WORLD_MONITOR_DATABASE_URL` is a separate profile-local connection for the
external component. Shadow validation retains explicit production/shadow DSNs.

`KNOWLEDGE_DATABASE_URL` is not read by the supported Go runtime. Knowledge
tools use their explicit `KNOWLEDGE_DSN` make target and dedicated tools Compose,
so the former root env alias was removed. Redis configuration was removed from
the root examples/generator and legacy JSON config; the Redis-backed library
code remains for CR-02F dependency review, but no supported root runtime
requires Redis.

## Service/port matrix

| Service/capability | Internal port | Host port | Ownership |
| --- | ---: | ---: | --- |
| `jax-trader` runtime/metrics | 8100 | 8100 | Core Jax |
| `jax-trader` frontend API | 8081 | 8081 | Core Jax |
| `jax-research` | 8091 | 8091 | Core Jax |
| `ib-bridge` | 8092 | 8092 | Core isolated bridge |
| `postgres` | 5432 | 5433 | Core Jax development DB |
| `frontend` nginx | 80 | 3000 | Core UI |
| Prometheus | 9090 | 9090 | Optional observability profile |
| Grafana | 3000 | 3001 | Optional observability profile |
| World Monitor events | 8082 | 8082 | Optional separate profile |
| World Monitor Postgres | 5432 | 5434, loopback | Optional separate profile |
| Playwright agent | 9092 | host process | Test-only script, not Compose core |

Internal service URLs use Compose DNS names; browser-facing defaults use
localhost and are build-time non-secret values.

## AI configuration

`JAX_MODEL_PROVIDER` is owned by the Jax inference boundary and defaults to
`deterministic`; `JAX_MODEL_BASE_URL`, `JAX_MODEL_MODEL` and
`JAX_MODEL_API_KEY` are provider-specific optional inputs. `OPENAI_*` remains
the OpenAI adapter fallback/embedding contract. `JAX_PLANNER_*` selects the
planner contract. `EMBEDDING_PROVIDER`/`EMBEDDING_MODEL` are separate memory
configuration. `JAX_AI_*` is diagnostic/shadow-only configuration. No
LiteLLM/Agent0/Dexter gateway names remain in active configuration and no paid
provider is selected by default.

## Market/data configuration

| Capability | Primary | Fallback | Notes |
| --- | --- | --- | --- |
| Current market/quotes | IB bridge | None | Provider identity and session semantics remain visible |
| Historical daily research bars | Alpaca approved dataset | None | Parent HYP evidence is immutable/private |
| Polygon current event/market adapter | Polygon | None | Explicit optional key/base/auth settings |
| SEC filings and acceptance time | SEC/EDGAR | None | Official evidence/provenance boundary |
| Macro/economic evidence | FRED/ALFRED, BLS, Treasury, CBOE | None | Retained official-source adapters |
| World/news event ingestion | Separate World Monitor | Polygon/SEC only where semantically applicable | No silent provider substitution |
| Broker/account boundary | IB bridge | None | Execution authority remains disabled |

Adjusted/raw bars, splits/dividends, timestamps, sessions, calendars, missing
bars, symbol changes and inactive-symbol limitations are part of each dataset
or provider contract; a fallback cannot silently change those semantics.

## Frontend configuration

The canonical browser variables are `VITE_JAX_API_URL` and
`VITE_IB_BRIDGE_URL`. The former `VITE_API_URL`/`VITE_MEMORY_API_URL` aliases
were removed from the examples. Vite build arguments contain endpoints only;
no secret is compiled into browser assets. Nginx uses fixed same-origin proxy
routes to the core services and exposes no arbitrary URL forwarding.

## Dead configuration removed

- Root Redis and `KNOWLEDGE_DATABASE_URL` example settings.
- Unread `MARKET_DATA_PROVIDER`, `JAX_API_PORT`, `JAX_RESEARCH_PORT`,
  `JAX_INGEST_INTERVAL`, `JAX_SYMBOLS` and obsolete frontend aliases from the
  root example.
- Redis cache blocks from legacy unowned JSON configuration fixtures.
- Stale Redis/knowledge/API settings from `generate-credentials.ps1`; its path
  now correctly targets the root `.env`.
- The ignored local provider override's stale Dexter entry (local-only; it is
  not a tracked artifact).
- The conflicting standalone Postgres Compose file and its active Makefile
  entry points.
- Stale `jax-api` Prometheus scrape target.

Historical/archive documentation and completed phase fixtures retain old names
when needed to explain past architecture; those are not active runtime seams.

## Secret surface

| Secret/configuration | Recipient | Status |
| --- | --- | --- |
| `POSTGRES_PASSWORD` | Root Postgres; Jax DSN as required | Active |
| `JWT_SECRET`, auth bootstrap values | `jax-trader` | Active, optional/local or deployment supplied |
| `ALPACA_API_KEY`/`ALPACA_API_SECRET` | `jax-trader` when market capability needs them | Active optional |
| `POLYGON_API_KEY` | `jax-trader` when Polygon capability needs it | Active optional |
| `OPENAI_API_KEY` / `JAX_MODEL_API_KEY` | `jax-research` and only relevant adapter paths | Active optional |
| `TELEGRAM_BOT_TOKEN` | `jax-trader` notification dispatcher only | Active optional |
| `SEC_USER_AGENT`/`SEC_CONTACT` | Explicit SEC acquisition processes | Identity metadata, not a secret value claim |
| Legacy Finnhub/NewsAPI/Massive/Financial Datasets/Dexter/Agent0 values | None | Removed from supported config; local ignored legacy file was not rewritten |
| World Monitor DB credential | `worldmonitor-events` profile only | Separate component, development fallback only |

The root Compose file does not distribute secrets to services that do not own
the corresponding capability. Exact values are never committed or recorded.

## Safe defaults

- No live trading or broker execution.
- Execution instruction worker disabled and leverage capped at 1x.
- Explicit trader `paper` and research `research` modes.
- Deterministic/offline AI is the default.
- World Monitor and observability are opt-in profiles.
- Missing optional integrations do not appear as healthy active capabilities.
- Unknown/invalid provider or safety settings continue to fail closed in code.

## Alternative Compose files

| File | Classification | Reason |
| --- | --- | --- |
| `docker-compose.yml` | SUPPORTED | Canonical core plus explicit optional profiles |
| `docker-compose.shadow.yml` | SPECIALISED TEST | ADR-0012 shadow validation; not normal startup |
| `tools/docker-compose.yml` | OPTIONAL SPECIALISED | Knowledge tooling; Qdrant remains `vector` profile-gated |
| `archive/jax-memory/docker-compose.yml` | HISTORICAL | Archived component, not supported |
| `db/postgres/docker-compose.yml` | DEAD / REMOVED | Conflicting duplicate Postgres contract |

## Scripts/config files removed or retained

`start.ps1` now starts core services and accepts `-Observability`. The
credentials generator, migration script, Makefile and database docs use the
root stack. Phase-era scripts and archived evidence remain retained where they
are reproducibility/history records; they are not supported startup paths.
`config/providers.json` remains the active provider-neutral registry. The
ignored local override no longer contains a Dexter provider. Versioned strategy,
risk, evidence and migration artifacts were not rewritten.

## Remaining CR-02F concerns

- Dependency/license/SBOM inventory, including the retained Redis library.
- Floating container image tags and image provenance.
- Formal commercial data-rights review for Alpaca, Polygon, IB, SEC, World
  Monitor and any retained redistributed/derived data.
- Production secret storage/rotation and deployment-specific credential policy.
- Broader dependency minimization outside the configuration surfaces addressed
  here.

## Roadmap state

- CR-02A = `GO`
- CR-02B = `GO`
- CR-02C = `GO`
- CR-02D = `GO`
- CR-02E = `IMPLEMENTATION COMPLETE — EXTERNAL REVIEW REQUIRED`
- CR-02F = `NOT STARTED`
- CR-02G = `NOT STARTED`
- Phase 13 = `NOT STARTED / BLOCKED BY CLEANUP GATE`
