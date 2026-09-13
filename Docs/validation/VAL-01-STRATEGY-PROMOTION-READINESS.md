# Jax VAL-01 Strategy Promotion & Forward-Paper Readiness

## Starting state

- Repository: `C:\Projects\Jax\jax-trading-assistant`
- Branch: `capability-reset`
- Starting HEAD: `036547d4d1ca4f464c9917cf9fcc60b8d562f7c9`
- Starting worktree: clean
- Starting `origin/capability-reset`: `036547d4d1ca4f464c9917cf9fcc60b8d562f7c9`
- Starting divergence: `0 ahead / 0 behind`
- No strategy optimisation, parameter search, model shopping, new inference,
  broker call or paper run was performed.

## CR-02G external GO recorded

The current governance record now records `CR-02G = COMPLETE / GO` and
`CR-01 → CR-02G TECHNICAL CLEANUP = COMPLETE / GO`. This is technical cleanup
acceptance, not commercial-sale or licensing clearance.

## Personal-platform objective

Jax is currently a personal trading platform. `PERSONAL TRADING VALIDATION =
ACTIVE OBJECTIVE`. `OPTIONAL FUTURE COMMERCIALISATION = DEFERRED / NOT A
CURRENT OBJECTIVE`. The prior commercial-readiness hardening remains intact,
but customer packaging and SaaS work are outside VAL-01.

## Current promotion pipeline

The evidence path is:

```text
implemented logic
  → frozen identity and point-in-time inputs
  → development evidence
  → validation selection
  → frozen candidate
  → unseen OOS / walk-forward evidence
  → falsification, cost, concentration and calibration review
  → formal admission manifest
  → real elapsed forward paper
```

A paper-capable runtime or a reviewable candidate is not itself a
`FORWARD_PAPER_ELIGIBLE` strategy.

## Candidate inventory

The inventory was derived from `libs/strategytypes`, `libs/strategies`, the
Swing Brain, `config/strategy-instances`, the Phase-06/07 research contracts,
the Phase-12 registry and the retained run artifacts.

### Candidate assessment matrix

| Candidate ID / version | Evidence source and current behaviour | Assessment | Promotion blocker |
| --- | --- | --- | --- |
| `rsi_momentum_v1` | `libs/strategies/rsi_momentum.go`; generic RSI signal and legacy backtester tests | `INSUFFICIENT_EVIDENCE` | No frozen dataset/config hash, OOS/walk-forward, cost/falsification bundle or forward manifest |
| `macd_crossover_v1` | `libs/strategies/macd_crossover.go`; generic MACD signal and tests | `INSUFFICIENT_EVIDENCE` | Same missing promotion evidence; test coverage is not performance evidence |
| `ma_crossover_v1` | `libs/strategies/ma_crossover.go`; generic moving-average signal and tests | `INSUFFICIENT_EVIDENCE` | Same missing promotion evidence |
| `same_day_earnings_drift_v1` | `libs/strategytypes/strategy_earnings_drift.go`; paper instance `config/strategy-instances/earnings-qqq-paper-v1.json` is disabled | `INSUFFICIENT_EVIDENCE` | No accepted historical evidence bundle or real forward evidence; 1m/5m and earnings timing require dedicated point-in-time validation |
| `same_day_news_repricing_v1` | `libs/strategytypes/strategy_news_repricing.go` | `INSUFFICIENT_EVIDENCE` | No frozen event dataset, OOS, cost and falsification record |
| `news_shock_momentum_v1` | `libs/strategytypes/strategy_news_shock_momentum.go` | `INSUFFICIENT_EVIDENCE` | No frozen evaluation/admission record |
| `opening_range_to_close_v1` | `libs/strategytypes/strategy_opening_range.go`; `or-spy-paper-v1` is disabled | `INSUFFICIENT_EVIDENCE` | No reproducible OOS/walk-forward evidence, realistic fills or paper admission |
| `event_gap_continuation_v1` | `libs/strategytypes/strategy_event_gap_continuation.go` | `INSUFFICIENT_EVIDENCE` | No evidence bundle and event-time treatment not demonstrated |
| `panic_reversion_v1` | `libs/strategytypes/strategy_panic_reversion.go` | `INSUFFICIENT_EVIDENCE` | No frozen evaluation, regime, cost or concentration evidence |
| `pairs_event_relative_v1` | `libs/strategytypes/strategy_pairs_event_relative.go`; marked research-first in source | `RESEARCH_CANDIDATE` | Peer selection, point-in-time universe, dependence and survivorship are not resolved |
| `index_flow_v1` | `libs/strategytypes/strategy_index_flow.go` | `INSUFFICIENT_EVIDENCE` | No frozen OOS/walk-forward or cost evidence |
| `etf_news_market_panic_reversal_v1` | `libs/strategytypes/strategy_etf_news_market_panic.go`; disabled paper instance | `INSUFFICIENT_EVIDENCE` | No complete historical event/price evidence bundle or forward evidence |
| `etf_news_sector_momentum_v1` | `libs/strategytypes/strategy_etf_news_sector_momentum.go`; disabled paper instance | `INSUFFICIENT_EVIDENCE` | No complete historical/OOS/falsification package |
| `etf_news_rates_bonds_rotation_v1` | `libs/strategytypes/strategy_etf_news_rates_rotation.go`; disabled paper instance | `INSUFFICIENT_EVIDENCE` | No complete historical/OOS/falsification package |
| `SWING_BRAIN_V1` | `internal/decisioning/brains/swing`, `Docs/STRATEGIES/SWING_TRADING`; deterministic decisions `NO_TRADE`, `WATCH`, `SETUP_FORMING`, `TRADE_CANDIDATE` | `INSUFFICIENT_EVIDENCE` | Decision/risk rules are implemented and tested, but no frozen strategy-level research bundle, OOS/walk-forward evidence or calibration evidence exists |
| `HYP-EVENT-001A` | Phase-12 private artifact and scientific closure; 2024 lineage is exploratory and promotion is closed | `CONTAMINATED_FOR_FORMAL_OOS` | Required development→validation→admission→freeze sequence was not durably recorded; 2025 remains sealed and survivorship is unresolved |

The documented `hyp_swing_001` and `hyp_commodity_dislocation` values are
templates/test fixtures, not registered forward candidates. The
`Docs/runs/real-candidate-proof` files demonstrate candidate-routing and
approval safety outcomes; they do not establish a strategy performance sample
or create paper orders. They are not treated as candidates.

## Promotion admission contract

Contract identity: `jax.val-01.forward-paper-admission/v1`.

Admission requires a complete immutable manifest binding all of the following:

1. Candidate ID, strategy version, exact code SHA, configuration and
   configuration hash.
2. Dataset identities, provider/data contracts, model and prompt identities
   where applicable, and the exact cost/risk/decision-policy versions.
3. Point-in-time knowability: publication/acceptance timestamps, session
   calendar, corporate-action and symbol-history semantics, and no future data
   in features or labels.
4. Explicit development, validation, OOS and final-holdout boundaries. A
   repeatedly inspected OOS or holdout period cannot be called pristine.
5. Genuine unseen-period or walk-forward evidence appropriate to signal rate,
   horizon and universe. One historical case or one issuer is insufficient.
6. Realistic fees, spread, slippage, latency, partial-fill, market-hours,
   liquidity and order-type assumptions bound to the frozen Phase-11 model
   where applicable.
7. Falsification including null/placebo, shuffled or randomized controls,
   alternative periods/regimes, unrelated assets where meaningful, parameter
   perturbation and benchmark comparisons.
8. Concentration and dependence analysis for issuer, sector, market-cap,
   regime, time period, outliers, survivorship, delistings and symbol changes.
9. Calibration/resolution evidence whenever a probability or confidence is
   emitted; confidence is not accepted as calibrated probability by assertion.
10. Explicit `ALLOW`, `REVIEW` or `ABSTAIN` behaviour for missing, stale,
    conflicting, low-quality or out-of-distribution evidence.
11. Reproducibility from the frozen inputs, code, configuration and model/prompt
    contract within the documented determinism boundary.
12. A promotion freeze: any material post-admission change creates a new
    strategy version and resets the evidence clock unless a separately frozen
    protocol says otherwise.

The only acceptable VAL-01 admission statuses are
`FORWARD_PAPER_ELIGIBLE`, `RESEARCH_CANDIDATE`, `INSUFFICIENT_EVIDENCE`,
`FAILED_VALIDATION`, `CONTAMINATED_FOR_FORMAL_OOS`, `HISTORICAL_ONLY` and
`NOT_A_TRADING_STRATEGY`.

## HYP-EVENT-001A disposition

`HYP-EVENT-001A FORWARD PAPER ELIGIBILITY = CONTAMINATED_FOR_FORMAL_OOS`.

This is a governance classification, not a claim that the underlying data was
corrupt. The final scientific closure records the 2024 values as exploratory
because the required progression/admission sequence was not durably recorded.
The 2025 final holdout remains sealed. Current-ticker survivorship and
delisted-asset coverage remain unresolved. The promotion gate remains closed,
recommendation logic is unchanged, and no paper run was started for the
hypothesis.

The scientific status remains exactly:

```text
HYP-EVENT-001A = NOT VALIDATED
TRADING EDGE = NOT DEMONSTRATED / INSUFFICIENT SAMPLE
```

## OOS / walk-forward evidence

Phase 07 provides reusable replay, backtest, cost and walk-forward contracts,
but the inventory found no candidate with a complete strategy-specific,
frozen, concentration-aware walk-forward admission package. HYP-EVENT-001A's
2024 values are not formal pristine OOS for this decision. No final-holdout
outcomes were opened or calculated.

## Cost and slippage evidence

The Phase-11 cost model and paper venue are implemented and versioned. The
candidate inventory does not show a complete candidate-level admission record
binding each strategy's execution assumptions, liquidity, order type, partial
fills, fees, spread, slippage and latency to an unseen evaluation. Therefore
cost infrastructure exists, but no candidate passes the cost-evidence gate.

## Falsification evidence

The Phase-12 HYP-EVENT line retains its registered falsification plan, but its
formal OOS status is not validated. The other inventoried strategies have no
complete retained placebo/null, shuffled-label, alternative-period/regime and
concentration evidence package. No new falsification run or strategy search was
performed in VAL-01.

## Calibration and resolution evidence

No inventoried candidate has a retained candidate-level calibration and
resolution report sufficient for admission. Confidence fields exist in Jax
contracts and signals, but their presence is not calibration evidence. The
Swing Brain explicitly supports uncertainty through blocking/watch outcomes;
that is a safety behaviour, not a demonstrated probability calibration result.

## Uncertainty and abstention evidence

The current architecture supports `NO_TRADE`, `WATCH`, blocked/unknown
evidence states, risk vetoes and paper-only boundaries. This is sufficient to
preserve safe non-action, but the inventory does not show a frozen strategy
manifest demonstrating abstention rates, distribution shift handling and
outcome-conditioned calibration for any candidate.

## Contamination / leakage review

- No 2025 semantic or outcome data was accessed.
- No 2024 result was rerun, retuned or relabelled.
- No labels were generated from future returns in this audit.
- No parameter, threshold, feature, asset or model search was performed.
- HYP-EVENT-001A remains exploratory/not validated because historical OOS
  admission was missing, not because the audit silently repairs it.
- Legacy generic backtest code is not promoted by the existence of unit tests;
  point-in-time and execution semantics require candidate-specific evidence.

## Concentration review

No candidate passed the required concentration review. The HYP-EVENT dataset's
historical issuer concentration facts remain in its private evidence record,
but survivorship limitations and the missing formal OOS progression prevent
promotion. The disabled ETF paper instances do not supply a historical sample.

## Strategy freeze readiness

No strategy version has a complete VAL-01 admission manifest. Consequently no
strategy is frozen for real forward paper and no evidence clock has started.

## Forward-paper admission

`FORWARD PAPER ADMISSION = NO-GO`.

No existing candidate satisfies the admission contract. A real forward-paper
programme must not start merely because Phase 11 capability is implemented or
because an approval-gated candidate route exists.

## Selected candidate, if any

None. No candidate is selected and no admission manifest is created.

## Strategy freeze manifest, if applicable

Not applicable. Creating a manifest would falsely imply admission.

## Forward-soak infrastructure

`FORWARD SOAK INFRASTRUCTURE = NOT READY` for a strategy-specific six-month
evidence programme, although the Phase-11 paper capability is implemented and
accepted.

`internal/modules/papertrading/soak.go` and the WP-11.08 evidence distinguish
`SOAK_INFRASTRUCTURE_DEMONSTRATED`, `ACCELERATED_SYNTHETIC_SOAK` and
`REAL_TIME_FORWARD_SOAK`. The accelerated exit harness is correctly not real
elapsed evidence. The current soak observation contract captures core venue
and safety counters, but a final candidate-specific prospective programme
still needs durable bindings for strategy/configuration identity, elapsed
market days, recommendation decisions/abstentions, benchmark-relative net
outcomes, calibration, concentration, provider/data incidents and complete
operational continuity. These are readiness gaps, not a reason to start a
real-time soak during VAL-01.

Preserved Phase-11 facts:

```text
PAPER-TRADING CAPABILITY = IMPLEMENTED / ACCEPTED
SOAK INFRASTRUCTURE DEMONSTRATED = YES
REAL FORWARD-PAPER EVIDENCE = 0 DAYS / 0 ORDERS
```

## Six-month real forward-evidence contract

The eventual programme must use real elapsed time, not accelerated simulation,
and bind an immutable admitted strategy version. It must record start/end
timestamps, market days, recommendations, abstentions, candidate decisions,
risk vetoes, approvals/rejections, paper intents/orders/fills, partial fills,
costs, slippage, latency, positions, gross/net/benchmark-relative returns,
drawdown, volatility, meaningful risk metrics, hit/payoff context, tails,
predicted versus observed probabilities, calibration/resolution, regime and
issuer/sector slices, data/provider incidents, crashes/restarts,
reconciliation, audit completeness and uptime.

Success cannot be defined as simply making money. The report must compare the
frozen strategy with its registered benchmark/null and explain missing or
abstained observations.

## Forward-paper invalidation rules

The programme must invalidate or pause consideration if the strategy/model/
prompt/configuration changes without a new version; thresholds are tuned from
forward outcomes; leakage is discovered; elapsed duration or sample is
insufficient; reconciliation/provenance/audit gaps are material; the live path
activates; drawdown exceeds the frozen policy; apparent performance is driven
by a few extreme winners, issuers, sectors or regimes; costs remove the
effect; calibration is materially poor; confidence has no resolution; data or
provider failures invalidate observations; continuity is broken by repeated
crashes; distribution shift does not trigger review/abstention; null/placebo
behaviour is indistinguishable; or human intervention changes the rules.

## Phase-13 personal-live gate

Phase 13 remains `NOT STARTED / BLOCKED BY STRATEGY VALIDATION AND REAL
FORWARD-PAPER EVIDENCE`. Before even considering it, Jax requires a frozen
`FORWARD_PAPER_ELIGIBLE` strategy, substantial real elapsed paper evidence,
demonstrated cost-adjusted benchmark-relative edge, calibration/uncertainty
evidence where applicable, reliable operations, deterministic risk gates,
HITL approval, no unresolved safety blockers, external technical-lead GO and a
separate owner decision to risk personal capital. Any initial live mode would
remain human-approved; this audit does not authorize it.

## Optional future commercialisation

`OPTIONAL FUTURE COMMERCIALISATION = DEFERRED / NOT A CURRENT OBJECTIVE`.
Commercial licensing, customer data rights and deployment decisions remain
separate from personal strategy validation.

## Next package

The bounded follow-on definition is in
`Docs/validation/VAL-01-NEXT-RESEARCH-PACKAGE.md`. It addresses the evidence
gaps exposed by this audit without starting execution, reopening the sealed
holdout or shopping for a new strategy.

## VAL-01 control statements

```text
STRATEGY INVENTORY COMPLETE = YES
PROMOTION CONTRACT FROZEN = YES
HYP-EVENT-001A STATUS PRESERVED = YES
NO HOLDOUT LEAKAGE INTRODUCED = YES
NO STRATEGY OPTIMISATION PERFORMED = YES
FORWARD PAPER ADMISSION = NO-GO
REAL FORWARD PAPER STARTED = NO
PHASE 13 STARTED = NO
OPTIONAL FUTURE COMMERCIALISATION = DEFERRED / NOT A CURRENT OBJECTIVE
SAFETY BOUNDARIES PRESERVED = YES
```
