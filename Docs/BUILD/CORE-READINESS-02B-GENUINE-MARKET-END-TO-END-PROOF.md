# CORE-READINESS-02B Genuine-Market End-to-End Proof

## Result

**Status: `BLOCKED_NO_ELIGIBLE_GENUINE_EVENT`**

The isolated proof reached the genuine World Monitor intake, deterministic event decision and asset-resolution boundary, and acquired genuine Alpaca quote/trade and candle observations. The current bounded event naturally remained `WATCH`; the existing deterministic promoter created no candidate. The proof stopped there. No portfolio-risk decision, canonical PREPARE, human approval, PAPER order/fill, lifecycle, or outcome was created.

This is a technical/provider result only. It is not strategy evidence, hypothesis selection or validation, profitability evidence, a new pilot, FORMAL_FORWARD_PAPER, broker execution, or live trading.

## Identity and isolation

- Starting reviewed SHA: `7966b92a5c6840a0cd3f37deec0babd06939ae94`.
- `PROOF_CODE_SHA`: `e818fa2ad3b31a72a535d150de1237cdaf4a2078`.
- Proof code commit: `Add guarded CORE-READINESS-02B proof runtime`.
- Proof database: `jax_paper02r_test_core02b_20260930`.
- Database container: `jax-core02b-proof-20260930`, container ID `1db909e5382060fd349480c7df25a15e18e1e43ae98358c13f14d42bef976135`, image `pgvector/pgvector:pg16` (digest `sha256:ccc6e83d6e35e931dc7c5def2022729d5a6c370318d099181995567ff1fb4d6b`). PostgreSQL data used a 512 MiB container tmpfs; the complete migration chain reached version 81.
- Dedicated PAPER account: `b33647ee-2e29-4aa0-8a79-f642a9c8c220` (`environment=PAPER`).
- One technical routing fixture: `6f707cf0-1f89-48aa-919c-7c97226d1d87`, strategy type `etf_news_sector_momentum_v1`. It was explicitly labeled `CORE_READINESS_02B_TECHNICAL_ROUTING_ONLY`, `NOT_STRATEGY_EVIDENCE`, `NOT_HYPOTHESIS_SELECTION`, and `DISPOSABLE`; both evidence/selection flags were false. It was not a selected strategy or a pilot.
- No normal Jax PostgreSQL connection was made. No normal Jax cursor was read or advanced. The normal World Monitor source database was not modified.

## Proof clock and market session

- Proof start UTC: `2026-09-30T18:27:30.0924154Z`.
- Before provider sampling, Windows Time was `Running` / `Automatic`; `w32tm /query /status` reported Leap Indicator `0 (no warning)`, Stratum `5`, source `time.windows.com` (`51.145.123.29`), with last successful sync `2026-09-30 17:03:51` local time.
- Live `w32tm /stripchart` samples during the proof were between `+58.8 ms` and `+61.8 ms`. The host clock gate passed; no clock was changed.
- The bounded sample was acquired on Wednesday, September 30, during the regular US session (14:27–14:36 US/Eastern during the recorded sampling steps).

## World Monitor evidence

The existing World Monitor events service was healthy. Its read-only health response at `2026-09-30T18:32:33.6841903Z` reported 910 retained events and maximum persistence sequence `72795`. This health request did not advance a World Monitor source cursor. The proof used its own disposable-consumer window: cursor position `72770` (max sequence minus the bounded page size 25), then the production pull worker fetched and committed the page through `72795`.

- Service: Compose service `worldmonitor-events`, accessed locally at `http://localhost:8082/api/v1/jax/events` because the Compose-only hostname in the existing `.env` is not resolvable from a host process.
- Endpoint identity digest: `sha256:347159f398747c2b1ab3d3a9059c09274b715e54f29a173cf9991eba423a42a3`.
- Provider schema: `world-monitor-events/v1`.
- Current-window page: page ID `2`; after `72770`; next cursor `72795`; one event; raw payload retained (1,867 bytes); page digest `e0b174046fbdaebc69431ea41adf7375e9d25e206cbac84b39dcb7cb80c762dc`; acquired `2026-09-30T18:34:06.180927Z`.
- Event identity: source event ID `wm_ce04220e03761d18cdf4f54c61dbdcb4456ac1ee324ab8ff4b82fa556694c9b3`; provider `world-monitor`; source `https://www.cnbc.com/2026/09/30/iran-war-strait-hormuz-gulf-oil-fuel.html`; source count 1; event type `energy_oil`; publication/event time `2026-09-30T18:17:24Z`; received `2026-09-30T18:34:06.178744Z`.
- Genuine-source checks: normalized record `data_source_type=real`, `source_provider=world-monitor`, `is_synthetic=false`; the retained provider page and its digest are in the disposable database. The current event decision was ID `973193d4-a444-4cd8-86e7-7dd5910e69a7`, origin `live_origin`, ruleset `genuine-event-decision-v2`, result `WATCH`. Reasons: `material_event_requires_continued_observation` and `persisted_asset_mapping_exists_but_no_complete_candidate_contract_exists`.
- Asset resolution was `resolved` to `XLE` by the configured `event_category_proxy` rule. The normalized inbox's `possible_affected_etfs` remained empty; the decision did not manufacture a candidate contract.
- The production World Monitor promoter endpoint was invoked with the dedicated authenticated disposable operator. It returned promoted `0`, skipped `0`, outcomes `0`, promoted items `0`. The inbox confidence was `0.50`, below the unchanged production promoter minimum `0.55`; no candidate was produced.
- Independent disposable cursor for this current-window endpoint advanced `72770 → 72795`. The normal Jax cursor was not accessed.

An initial bounded pull from position zero returned 25 genuine but stale events published July 29–31 (page ID `1`, cursor `0 → 27`, digest `ecb38b1cd58f90bb8c7d2b8fc973548a29a2889bf2b6d7568b4005bfdac0095c`). That page was excluded from the current proof window: all 25 asset resolutions were `unresolved`; its decisions were 23 `NO_TRADE` for confidence below the watch threshold and 2 `WATCH` records requiring continued observation without truthful asset mapping. The endpoint was then independently bounded to its current tail using the read-only health maximum above. These older rows remain preserved only in the disposable proof database and were not used to create candidates.

## Alpaca market evidence

- Provider configuration passed the proof guard with exactly one enabled market-data provider: `alpaca`. `IB_BRIDGE_URL` was empty; no Polygon provider was configured. API credentials were read from the existing supported local `.env` configuration and are intentionally not recorded here.
- Eleven latest quote rows were persisted in the disposable database, all with provider `alpaca` and non-null quote, latest-trade and local receipt timestamps. The candidate-economic policy symbol allowlist was unchanged: `SPY, QQQ, DIA, IWM, XLK, XLF, XLE, SMH, SOXX, TLT, GLD`.
- Relevant `XLE` quote: provider quote timestamp `2026-09-30T18:34:07.801358Z`; latest-trade timestamp `2026-09-30T18:34:07.118658Z`; local response receipt `2026-09-30T18:34:07.819668Z`; bid/ask `61.93 / 61.94`; displayed bid/ask sizes `900 / 8600`.
- `AvailableAt = max(quote, trade, receipt) = 2026-09-30T18:34:07.819668Z`. Quote-to-local-receipt difference was about `18.31 ms`, within the existing 250 ms allowance. The provider quote and trade timestamps were kept distinct and unchanged.
- The authenticated production genuine-candle collection route requested 7 days of `XLE` `1h` Alpaca candles from `2026-09-23T18:35:59Z`, receiving and persisting 37. Provider timestamp semantics were `interval_start`; ingestion timestamps were persisted separately. At collection as-of `2026-09-30T18:35:59Z`, 36 bars had completed; latest complete interval began at `17:00Z` and ended at `18:00Z`. The `18:00Z` interval-start bar was not yet complete at that as-of and is excluded by the production no-lookahead reader. That latest completed bar was ingested at `18:36:07.386123Z`. No position existed, so no production position-review observation ran. Market-data classification and adjustment treatment remained honestly recorded as `unknown` by the existing adapter.
- The existing startup ingester also persisted 22 genuine Alpaca daily bars (two per configured symbol); these were not used as substitutes for the required hourly chart history or as candidate evidence.

## Furthest proven boundary and stop

- Current sample event: one genuine event; deterministic result `WATCH`; asset resolved to XLE; candidate promoter result empty.
- Candidate outcome: none. No candidate/evidence item or candidate economic input was created. The sample did not cross the unchanged confidence/candidate-contract boundary.
- Portfolio risk: not run because no canonical candidate existed.
- Human entry decision: none requested or made. No approval was attempted.
- Queue/order/fill/lifecycle/outcome: zero. Disposable DB counts: candidate trades 0; candidate economic inputs 0; account-scoped entry queue 0; PAPER orders 0; PAPER fills 0; exploratory lifecycles 0; outcomes 0.
- The proof runtime ran only in `PAPER`; `ExecutionAuthority=NONE`; `EXECUTION_ENABLED=false`; `BROKER_EXECUTION_ALLOWED=false`; `ALLOW_LIVE_TRADING=false`; maximum leverage was 1x. IB Bridge was not used. Broker/order API calls: 0.
- No event/candidate threshold, chart requirement, risk parameter, market freshness bound, or strategy setting was changed. No genuine candidate, quote, event, liquidity, or human decision was fabricated.

## Validation and exact-SHA workflows

Local validation on the proof-code SHA used a fresh disposable validation database migrated to version 81. `go test -count=1 ./...`, `golangci-lint run ./...` (0 issues), gofmt verification, `git diff --check`, `go list -mod=readonly -m all`, and `docker compose config --quiet` passed. The optional raw-payload integration variable was left unset to keep that test isolated; required PostgreSQL tests ran with `PAPER02R_REQUIRED_INTEGRATION=true`.

For `PROOF_CODE_SHA=e818fa2ad3b31a72a535d150de1237cdaf4a2078`, all required exact-SHA workflows passed:

- CI run `36757414662` — passed, including PostgreSQL integration, Linux race jobs and frontend E2E.
- Golden Tests run `36757414403` — passed.
- Import Boundary Enforcement run `36757415264` — passed.
