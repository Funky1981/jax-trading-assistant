# CORE-READINESS-02B1 — Proof Policy and Market Temporal Prerequisites

Status: **IMPLEMENTED / VALIDATION IN PROGRESS**. CORE-READINESS-02B1 closes technical policy and market-time semantics prerequisites only. The isolated genuine-market economic end-to-end proof, CORE-READINESS-02B, remains **NOT YET EXECUTED**.

## Boundary

The policy values below are operator-selected parameters for the technical proof harness only. They are not a strategy, economic hypothesis, profitability assumption, formal-paper policy, production sizing recommendation, or evidence of an edge. Do not carry them forward automatically into hypothesis selection.

The normal Jax database remains stopped/untouched for this package. No normal strategy instance is enabled or changed. No strategy or economic hypothesis is selected. No new pilot, broker execution, or live trading is authorized. The recovered World Monitor service is an external healthy dependency and is not modified or restarted here.

## Candidate economic technical-proof policy

`config/core-readiness-02b-candidate-economic-policy.json` is versioned as `core-readiness-02b-technical-proof-v1`, with identity policy `jax-us-etf-identity-v1` and source `operator-reviewed-official-fund-identity-mapping`.

| Parameter | Technical-proof value | Meaning |
| --- | ---: | --- |
| Risk allocation | `0.005` | 0.5% of PAPER equity; lower bound already present in the reviewed risk-constraints range |
| Requested leverage | `1.0` | Cash-only; no borrowing beyond equity |
| Slippage allowance | `0.50 USD/share` | Explicit sizing/revalidation allowance, not a forecast of realized slippage |
| Sizing identity | `core-readiness-02b-technical-sizing`, `v1` | Technical-proof request identity |

Actual simulated spread, slippage, and commission accounting remains governed by the existing PaperVenue cost model. The technical allowance does not replace or predict those costs.

## Instrument and legal issuer identities

Instrument IDs use the stable Jax namespace `jax.instrument.us.etf.<ticker>`. Issuer IDs identify the verified legal fund/trust issuer (not a ticker-derived string). Where a sponsor is distinct from the trust issuer, the issuer ID denotes the legal trust and this table records the sponsor/adviser separately rather than conflating their roles.

| Symbol | Internal issuer ID | Legal fund/trust identity | Sponsor/adviser provenance |
| --- | --- | --- | --- |
| SPY | `jax.issuer.us.trust.spdr-s-and-p-500-etf-trust` | State Street SPDR S&P 500 ETF Trust | PDR Services LLC is identified as sponsor in official SSGA fund details |
| QQQ | `jax.issuer.us.trust.invesco-qqq-trust-series-1` | Invesco QQQ Trust, Series 1 | Invesco identifies Invesco Capital Management LLC as sponsor |
| DIA | `jax.issuer.us.trust.spdr-dow-jones-industrial-average-etf-trust` | State Street SPDR Dow Jones Industrial Average ETF Trust | PDR Services LLC is identified as sponsor in the 2026 prospectus |
| IWM | `jax.issuer.us.trust.ishares-trust` | iShares Russell 2000 ETF, a series of iShares Trust | BlackRock Fund Advisors is identified as investment adviser |
| XLK | `jax.issuer.us.trust.select-sector-spdr-trust` | Technology Select Sector SPDR Fund | Identified as a Select Sector SPDR Trust fund |
| XLF | `jax.issuer.us.trust.select-sector-spdr-trust` | Financial Select Sector SPDR Fund | Identified as a Select Sector SPDR Trust fund |
| XLE | `jax.issuer.us.trust.select-sector-spdr-trust` | Energy Select Sector SPDR Fund | Identified as a Select Sector SPDR Trust fund |
| SMH | `jax.issuer.us.trust.vaneck-etf-trust` | VanEck Semiconductor ETF, a series of VanEck ETF Trust | VanEck identifies Van Eck Associates Corporation as adviser |
| SOXX | `jax.issuer.us.trust.ishares-trust` | iShares Semiconductor ETF, a series of iShares Trust | BlackRock Fund Advisors is identified as investment adviser |
| TLT | `jax.issuer.us.trust.ishares-trust` | iShares 20+ Year Treasury Bond ETF, a series of iShares Trust | iShares identifies the fund and its prospectus |
| GLD | `jax.issuer.us.trust.spdr-gold-trust` | SPDR Gold Trust | World Gold Trust Services, LLC is identified as sponsor |

Official source identities (provider pages and/or filed fund documents):

- SPY: <https://www.ssga.com/us/en/intermediary/etfs/state-street-spdr-sp-500-etf-trust-spy>
- QQQ: <https://www.invesco.com/qqq-etf/en/about.html>
- DIA: <https://www.sec.gov/Archives/edgar/data/1041130/000119312526067237/d35140d485bpos.htm>
- IWM: <https://www.ishares.com/us/products/239710/iwm-ishares-russell-2000-etf> and <https://www.ishares.com/us/literature/prospectus/p-ishares-trust-russell-market-cap-3-31.pdf>
- XLK: <https://www.ssga.com/mainfund/XLK>
- XLF: <https://www.ssga.com/us/en/intermediary/etfs/state-street-financial-select-sector-spdr-etf-xlf> and <https://www.sec.gov/Archives/edgar/data/1064641/000119312526027312/d15107d485bpos.htm>
- XLE: <https://www.sec.gov/Archives/edgar/data/1064641/000119312525017256/d913710d497k.htm>
- SMH: <https://www.sec.gov/Archives/edgar/data/1137360/000113736026000117/0001137360-26-000117-index.htm> and <https://www.vaneck.com/us/en/investments/semiconductor-etf-smh/>
- SOXX: <https://www.ishares.com/us/literature/summary-prospectus/sp-ishares-phlx-semiconductor-etf-3-31.pdf>
- TLT: <https://www.ishares.com/us/products/239454/ishares-20-year-treasury-bond-etf>
- GLD: <https://www.ssga.com/us/en/individual/etfs/spdr-gold-shares-gld>

The policy contains only the approved catalog symbols above. TQQQ, SQQQ, UVXY, and VXX are absent. No issuer mapping was guessed from a ticker.

## Portfolio risk policy

`config/core-readiness-02b-portfolio-risk-policy.json` maps only directly supported `portfoliorisk.RiskPolicy` fields: contract `jax.portfolio.risk_policy/v1`; version `core-readiness-02b-technical-proof-v1`; currency `USD`; maximum position value `50000`; maximum risk allocation `0.02`; maximum leverage `1.0`. Unsupported optional dimensions remain unset. `BuildRiskPolicy` calculates the policy identity. Technical-proof runtime freshness is `JAX_PORTFOLIO_STATE_MAX_AGE=60s`.

## Market-data temporal contract

The former generic `JAX_MARKET_DATA_MAX_AGE` is not consulted as authority by the canonical economic market-data path. All split values are required and fail closed when absent or invalid:

| Setting | 02B technical proof |
| --- | --- |
| `JAX_MARKET_DATA_ALLOWED_SOURCES` | `alpaca` |
| `JAX_MARKET_DATA_TIMEFRAME` | `1h` |
| `JAX_MARKET_DATA_QUOTE_MAX_AGE` | `60s` |
| `JAX_MARKET_DATA_LATEST_CANDLE_MAX_AGE` | `90m` |
| `JAX_MARKET_DATA_CANDLE_LOOKBACK` | `7d` |

Quotes use quote freshness only. Candle selection enforces configured symbol, timeframe, provider allowlist, `ingested_at <= asOf`, valid timestamp semantics, non-fixture classification, and lookback; it selects the requested count in deterministic newest-first order. Older required history may exceed 90 minutes, but the newest completed candle must be at most 90 minutes old. A stale newest completed candle rejects the chart input.

For `interval_start` candles, availability is derived as `CompletedAt = timestamp + timeframe`. For 1h this is exactly one hour; the candle is rejected until `CompletedAt <= asOf` and `ingested_at >= CompletedAt`. Thus a 10:00–11:00 interval cannot be consumed at 10:30, and a record ingested before 11:00 cannot pass even when examined later. Provider timestamps remain unchanged; `CompletedAt` is additional derived availability semantics. For 1d, the conservative completion bound is 24 hours after the stored interval start; no regular-session close is invented. Unsupported timeframe or timestamp semantics fail closed.

## Disposable technical routing fixture

The current promoter requires an enabled ETF `strategy_instances` identity even though no economic hypothesis has been selected. The only permitted routing fixture for the isolated 02B harness is the existing compatible `etf_news_sector_momentum_v1` contract, with labels `CORE_READINESS_02B_TECHNICAL_ROUTING_ONLY`, `NOT_STRATEGY_EVIDENCE`, `NOT_HYPOTHESIS_SELECTION`, and `DISPOSABLE`.

The PostgreSQL integration test creates this single enabled fixture only after the shared test guard validates the database name as `jax_paper02r_test` or `jax_paper02r_test_<suffix>`, verifies promoter routing, then deletes it. The CI integration service provisions an isolated PostgreSQL service and sets `PAPER02R_REQUIRED_INTEGRATION=true`; the new temporal and routing proof test is a required step. The fixture is not created in the normal Jax database, strategy validation, PAPER-02, formal evidence, or profitability statistics. No thresholds, confidence gates, chart confirmation, or entry/stop/target calculations were changed to force promotion.

## Corrected 02B proof criterion and remaining boundary

The future CORE-READINESS-02B proof must establish, in an isolated disposable database: genuine World Monitor intake and provenance; genuine Alpaca market observations; existing deterministic event/candidate logic; the explicitly labelled disposable routing identity; persisted candidate/evidence/economic input; canonical portfolio risk; JWT human approval; the account-scoped queue; PAPER entry/review/exit/outcome; and accounting/restart/replay integrity. Every resulting candidate remains technical-proof data, not strategy evidence. No candidate reaching the existing logic during the bounded genuine event sample is a valid `BLOCKED_NO_ELIGIBLE_GENUINE_EVENT` outcome; thresholds must not be tuned to force one.

This package does not execute that end-to-end proof. CORE-READINESS-02B remains **NOT YET EXECUTED**. CORE-READINESS-01 remains `BLOCKED_BEFORE_HYPOTHESIS_DESIGN` until the separately reviewed genuine-market proof is completed. No new strategy, pilot, formal-forward paper, broker execution, or live trading is authorized.
