# Jax Commercial-Readiness CR-02D — Data Vendor Consolidation

Status: implementation complete, external review required

This document is the current CR-02D inventory and cleanup record. It is a
technical dependency assessment, not a legal opinion and not authorization for
live execution or commercial redistribution.

## Scope and safety

The audit covers the active post-CR-02C tree on `capability-reset`. Historical
and archived documents are retained as history and are not executable provider
dependencies. The HYP-EVENT-001A datasets and the sealed 2025 holdout were not
modified.

The following remain unchanged:

- `ALLOW_LIVE_TRADING=false`.
- `BROKER_EXECUTION_ALLOWED=false`.
- Live execution worker disabled.
- Maximum leverage is 1x.
- No broker calls, orders, trades, fills, approvals or portfolio mutations
  were performed by this cleanup.

## Provider inventory and dependency proof

| Provider/component | Active paths and callers | Capability | Classification | CR-02D decision |
| --- | --- | --- | --- | --- |
| Interactive Brokers / IB Gateway | `services/ib-bridge/`; `libs/marketdata/provider_ib.go`; `libs/marketdata/provider_ib_bridge.go`; `internal/modules/execution/ib_adapter.go`; `cmd/trader` | Isolated broker account/market-data bridge, delayed/current quotes and candles, future adapter contract | ACTIVE REQUIRED, isolated | KEEP |
| Alpaca | `libs/marketdata/provider_alpaca.go`; hardened raw/provenance path; `cmd/trader`; HYP-EVENT commands and dataset evidence | Approved historical/daily market data, benchmark data and optional trader fallback | ACTIVE REQUIRED | KEEP |
| Polygon.io | `libs/marketdata/provider_polygon.go`; `cmd/trader/market_tools.go`; research backfill and optional market fallback | Recent quotes, daily/intraday bars, earnings and company news | ACTIVE REQUIRED, optional by configuration | KEEP as canonical identity |
| Massive | Previously accepted through `MASSIVE_*` aliases in trader/provider configuration | Same logical REST role as the Polygon adapter; no separately proven capability | DUPLICATE ALIAS | REMOVE active alias/config surface |
| Finnhub | Previously `cmd/trader/market_tools.go` earnings/news fallback and readiness credential | Earnings/news fallback behind Polygon | DEAD/DUPLICATE | REMOVE |
| NewsAPI | Previously `cmd/trader/market_tools.go` news fallback and Compose credential | General news fallback behind Polygon/World Monitor/SEC evidence | DEAD/DUPLICATE | REMOVE |
| Financial Datasets | `libs/marketdata/financialdatasets.go` and hardened normalizer/tests | Explicit research-only historical EOD adapter; no supported trader-runtime caller after cleanup | RESEARCH-ONLY / COMPATIBILITY | Remove runtime wiring; retain library for explicit research compatibility |
| World Monitor | `cmd/trader/world_monitor_*`; `../Jax-World-News-Monitor`; Compose `worldmonitor-events` and separate DB | User-owned event ingestion and durable cursor/replay contract | ACTIVE REQUIRED, separate component | KEEP SEPARATE |
| Telegram | `internal/modules/approvals/notification_dispatcher.go`; mobile notification worker | Optional operator approval notifications | ACTIVE OPTIONAL | KEEP optional; not a market-data source |
| SEC / EDGAR | `libs/sec`; `cmd/hyp-event-evidence`; evidence-quality and Phase-03 paths | Filing/accession evidence, XBRL and event-time provenance | ACTIVE REQUIRED / PRIMARY | KEEP |
| FRED / ALFRED | `libs/fred` | Macro observations, initial-release and vintage/revision-aware context | ACTIVE IMPLEMENTED / PRIMARY | KEEP |
| BLS | `libs/bls` | Official release-calendar evidence | ACTIVE IMPLEMENTED / PRIMARY for BLS releases | KEEP |
| Treasury | `libs/treasury` | Official rates/auction context | ACTIVE IMPLEMENTED / PRIMARY for Treasury data | KEEP |
| CBOE | `libs/cboe` | VIX historical/public volatility context | ACTIVE IMPLEMENTED / PRIMARY for CBOE data | KEEP |
| EIA | No active adapter found; roadmap/reference mentions only | Energy data | PLANNED / NOT PRESENT | DEFER |
| CFTC | No active adapter found; roadmap/reference mentions only | Commitment-of-traders data | PLANNED / NOT PRESENT | DEFER |

Other adjacent infrastructure is not a data vendor: Redis is a cache library
with no active root Compose service, and Prometheus/Grafana are optional
observability components. Neither was removed as part of this data-vendor
cleanup.

## Capability matrix

| Capability | Primary | Fallback | Current policy |
| --- | --- | --- | --- |
| Operator quote/candle access | IB bridge when configured | Alpaca, then Polygon in the configured read-only chain | Provider identity and timestamp remain observable; no silent semantic equivalence is claimed |
| HYP-EVENT historical daily bars | Alpaca SIP dataset `hyp-event-001a-sec-8k-alpaca-sip-2016-2025-v1` | None for the accepted experiment | Immutable private dataset; no raw data rewrite or redistribution |
| Historical intraday bars | Polygon or isolated IB bridge where explicitly configured | None | Not a Phase-12 research requirement |
| SPY benchmark data | Alpaca HYP dataset | None in the accepted experiment | Benchmark and adjustment semantics are dataset-bound |
| Corporate-action-adjusted research data | Alpaca all-adjusted HYP artifact | None | Preserve raw/all-adjusted identities; no new provider substitution |
| Recent equity earnings | Polygon | None | Finnhub fallback removed |
| Recent company news | Polygon | Calendar macro events / World Monitor where semantically applicable | NewsAPI and Finnhub fallbacks removed |
| SEC filings and event timestamps | SEC / EDGAR | None | Acceptance/publication time and accession identity are authoritative |
| World/news event ingestion | Separate user-owned World Monitor | None | Explicit HTTP/page contract and separate persistence boundary |
| Macro observations and vintages | FRED / ALFRED | None | Revision/vintage state is explicit |
| BLS releases | BLS | None | Official calendar evidence only |
| Treasury rates/auctions | Treasury | None | Official-source provenance required |
| Volatility/VIX context | CBOE | None | Public-source provenance required |
| Options data | None | None | No current requirement |
| Broker account state | IB bridge, isolated | None | Execution remains disabled |
| Order execution | None in supported runtime | None | No live or paper broker execution is enabled by CR-02D |
| Paper execution | Jax-owned Phase-11 paper domain | None | Separate from observed/real provider state |
| Economic releases | BLS/FRED and local calendar contracts | None | No EIA/CFTC adapter is present |

## Provider-specific decisions

### Interactive Brokers

Verdict: `INTERACTIVE BROKERS = KEEP`.

IB is not retained as an unbounded live-trading dependency. The Python bridge
and Go adapter provide an isolated account/market-data boundary used by the
trader runtime and future broker compatibility. Compose defaults remain paper
and delayed-data oriented; `ALLOW_LIVE_TRADING`,
`BROKER_EXECUTION_ALLOWED`, execution-worker flags and 1x leverage were not
changed. The bridge's order endpoints are not called by this cleanup.

### Alpaca

`ALPACA CURRENT REQUIRED ROLE = accepted HYP-EVENT-001A historical daily bars,
SPY benchmark evidence, and an optional read-only trader market-data fallback`.

Verdict: `ALPACA = KEEP`. The accepted private HYP dataset remains the primary
scientific artifact and its provenance is unchanged. Alpaca's feed, adjustment,
timestamp and dataset identities remain part of the research contract.

### Polygon/Massive

`POLYGON/MASSIVE PROVIDES UNIQUE REQUIRED CAPABILITY = NO` as two separate
integrations. The source tree had one Polygon Go adapter with Massive key/base
URL aliases, not two independently justified implementations. The canonical
active identity is now `polygon`; `MASSIVE_API_KEY` and `MASSIVE_BASE_URL` are
no longer loaded by trader/provider code or Compose. `POLYGON_API_KEY`,
`POLYGON_BASE_URL`, `POLYGON_AUTH_MODE` and `POLYGON_TIER` are the single
canonical configuration surface.

Verdict: `POLYGON/MASSIVE = CONSOLIDATE`; Polygon is retained and Massive's
duplicate active alias is removed.

### Finnhub and NewsAPI

`FINNHUB = REMOVE` and `NEWSAPI PROVIDES UNIQUE REQUIRED CAPABILITY = NO`.
Both were fallback HTTP implementations behind Polygon and had no unique
accepted role. Their runtime clients, fallback branches, readiness checks,
Compose variables and example configuration were removed. Historical CR-01
and archived records remain unchanged.

### Financial Datasets

`FINANCIAL DATASETS PROVIDES UNIQUE REQUIRED CAPABILITY = NO` for the current
supported trader/runtime estate. The dedicated package and hardened tests are
retained for explicit historical/research compatibility because they implement
canonical raw-first normalization contracts. Default trader composition,
research backfill selection, Compose configuration and Phase-03 fallback
readiness no longer activate or require it.

### World Monitor

Verdict: `WORLD MONITOR = KEEP SEPARATE`. It is user-owned software in
`C:\Projects\Jax-World-News-Monitor`, not an interchangeable commercial data
vendor. Jax consumes the versioned HTTP/page contract, stores the source page
and cursor provenance, and remains operable as a separate component. CR-02D
does not absorb or rewrite the sibling repository.

### Telegram

Verdict: `TELEGRAM = KEEP` as an optional operator-notification integration.
It is not a data vendor and is disabled when its token/chat configuration is
absent. It does not approve, execute, or create trading artifacts.

## Primary/fallback and semantic policy

Fallback is limited to the explicit chain constructed by
`cmd/trader/market_tools.go`: IB bridge, Alpaca, and canonical Polygon for
market reads. Event reads use Polygon when configured and otherwise degrade to
the local calendar or an explicit fail-closed error in production. Removed
vendors are not dormant fallbacks.

Provider substitutions must preserve or expose differences in:

- raw versus adjusted bars, split/dividend treatment and feed coverage;
- market session, holiday, timezone and timestamp semantics;
- missing bars, stale values, symbol changes and inactive symbols;
- event publication time versus retrieval time;
- source identity, endpoint family, retrieval timestamp and raw payload hash.

No provider is silently asserted to cover delisted symbols or to provide
survivorship-controlled history merely because its adapter exists.

## Provenance, scientific evidence and history

The HYP-EVENT-001A parent dataset, evidence extension, scientific results and
sealed 2025 holdout are immutable and untouched. Provider cleanup changes only
active runtime selection/configuration. Existing canonical raw payload and
normalization contracts retain provider identity, source identity, event/data
timestamps, adjustment policy, dataset identity and content hashes.

The current scientific status remains:

`TRADING EDGE NOT DEMONSTRATED / INSUFFICIENT SAMPLE`

This cleanup creates no new experiment result and makes no profitability or
data-quality claim.

## Credential reduction

Active market/news/data credential types before cleanup included:

- `ALPACA_API_KEY`, `ALPACA_API_SECRET`;
- `POLYGON_API_KEY` and Polygon configuration;
- `MASSIVE_API_KEY` and Massive configuration;
- `FINNHUB_API_KEY`;
- `NEWSAPI_KEY`;
- `FINANCIAL_DATASETS_API_KEY`;
- `SEC_USER_AGENT`, `SEC_CONTACT` and `FRED_API_KEY` where their retained
  official adapters require them.

After cleanup, active supported runtime configuration retains only the
credentials required by retained paths: Alpaca, canonical Polygon, SEC and
FRED. BLS, Treasury and CBOE use their existing public/official paths where
configured. Massive, Finnhub, NewsAPI and Financial Datasets variables were
removed from active Compose/example/runtime wiring. No secret values were
read into logs, source, tests or commits.

## Data-rights review flags

This is not a legal determination. Formal commercial data-rights review is
required before redistribution, resale, customer display, commercial caching
or external publication of data from Alpaca, Polygon, IB, World Monitor, SEC
or any other retained source where the source terms impose restrictions.

The private HYP-EVENT Alpaca market data remains local/private and is not
redistributed. SEC filing evidence retains source/provenance identity. The
repository does not assert that public availability removes all downstream
usage obligations.

## Runtime and configuration reduction

The cleanup removed the Finnhub and NewsAPI fallback branches and their
credential-bearing fields from the trader event aggregator; removed Financial
Datasets from default trader and backfill provider construction; and removed
Massive alias handling from the Polygon adapter, trader configuration and
Compose. `config/providers.json` contains only provider-neutral Jax capability
entries (`market-data`, `broker`, `storage`, `memory`, `risk`, `backtest`) and
no stale vendor aliases.

## Historical compatibility and deferred work

Archived implementation plans and CR-01's historical inventory are retained
for auditability. The Financial Datasets library remains an explicit
research-only compatibility adapter; it is not a supported default runtime
provider. EIA and CFTC remain deferred and were not implemented. CR-02E is the
next cleanup/review package and was not started.

Migration `000010_event_data_foundation` remains unchanged because it is part
of the immutable migration history and may be referenced by persisted events.
Forward migration `000068_retire_redundant_event_source` marks its historical
`finnhub` source row disabled and retired without deleting it. The migration
registry remains unique and ordered, and historical provider parsing remains
available only for compatibility/replay rather than active acquisition.

Current roadmap state:

- CR-02A = GO.
- CR-02B = GO.
- CR-02C = GO.
- CR-02D = IMPLEMENTATION COMPLETE — EXTERNAL REVIEW REQUIRED.
- CR-02E = NOT STARTED.
- Phase 13 = NOT STARTED / BLOCKED BY CLEANUP GATE.
