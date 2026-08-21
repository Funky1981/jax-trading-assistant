# Fincept Capability Map for Jax

This map is intentionally selective: it captures capabilities that can materially improve Jax, not every Fincept screen.

| Fincept capability | Evidence in repository | Jax value | Jax phase | Treatment |
|---|---|---:|---:|---|
| Modular bounded contexts | Architecture doc separates Markets, News, Economics, Geopolitics, Trading, Portfolio, Crypto, Derivatives, Predictions, Agents, Workflow | Very high | 01 | Adopt concept, preserve Jax topology |
| DataHub pub/sub | topic subscriptions, TTL, freshness, last-known-good, producer errors, rate control | Very high | 02 | Independently design Jax data plane/health semantics |
| Data normalization | raw + normalized records, schema validation/transforms | Very high | 02 | Adopt concept |
| Large connector catalogue | Python source integrations plus auto-generated manifest containing 190 dormant/unwired connectors | High as catalogue | 02–04 | Audit source-by-source; never assume production maturity |
| Market data | Yahoo/Alpha Vantage/CBOE/CFTC/FMP/etc.; Polygon connector exists in current tree | High | 03 | Choose providers independently |
| SEC/EDGAR | filings/XBRL tooling | Very high | 03 | Prefer official SEC APIs where possible |
| Macro/economics | FRED, World Bank, IMF, OECD, BLS, BEA, ECB, DBnomics | Very high | 03 | Curate small high-trust set |
| News monitoring/NLP/cluster/correlation | dedicated services and DataHub migration docs | Very high | 04 | Recreate concepts around World Monitor |
| Geopolitics/maritime | geopolitics and maritime services; ACLED/AIS connectors | High | 04 | Add only after source licensing/reliability check |
| Prediction markets | Polymarket service/scripts/WebSocket | Medium-high | 04 | Read-only evidence source candidate |
| Analytics library | 80+ documented analytics modules; NumPy/SciPy/scikit-learn and portfolio wrappers | Very high | 05 | Prefer mature libraries; validate formulas independently |
| Portfolio optimization | PyPortfolioOpt/Riskfolio/skfolio wrappers | High later | 05/09 | Evaluate against Jax needs |
| Alpha Arena | deterministic context, risk verdicts, append-only replay, HITL, kill/crash recovery | Very high | 01/07/10/11 | Adopt replay/safety concepts, not crypto/perp assumptions |
| Backtesting provider abstraction | multiple engines dynamically loaded from Python | High | 07 | Use provider abstraction; select engines based on task |
| MCP tool system | schemas, validation, async, timeout/cancel, auth/destructive confirmation | Very high | 08/10 | Strong reference for controlled AI tools |
| Agentic research | durable SQLite checkpointing; target reflection, replanning, budgets, HITL, memory/evals | High | 08 | Recreate minimal durable research runner |
| Workflow engine | executor, node registry, expression engine, audit logger, risk manager, confirmation | High | 10 | Recreate only needed state-machine/workflow pieces |
| Portfolio service/metrics | holdings, summaries, metrics, broker linkage | High | 09 | Map to canonical Jax portfolio state |
| Broker abstraction | wide broker interface and 16 broker implementations | Medium until late | 11/13 | Study capability contract; do not inherit breadth |
| Report builder | structured reporting components/tooling | Medium | 06/08 | Prefer simpler Jax research-report view first |
| Screeners | equity/F&O screener surfaces and provider scripts | Medium | 06/12 | Candidate-discovery later, not core |
| Surface/derivatives analytics | dedicated screen/MCP tooling | Low now | Later | Defer |
| TTS/STT/forum/voice | implemented services | Low | — | Reject for roadmap |
| HFT/market making/RL claims | AI Quant Lab/Qlib wrappers | Low until evidence demands | 12 | Explicitly deferred; high overfitting/complexity risk |

## Fincept verification rule

Before using any Fincept idea:
1. confirm whether it is **current**, **target**, **dormant/unwired**, or **documentation-only**;
2. inspect tests/callsites, not just README claims;
3. identify the underlying external library/provider;
4. compare the external library directly;
5. check licence/terms;
6. implement independently inside Jax unless an external dependency is deliberately adopted.

## Important maturity caveats

- Fincept's architecture document explicitly labels portions as current vs target.
- The current connector manifest identifies a large set of connectors not invoked by a normal source callsite.
- The public repository's maintenance notice says public development cadence reduced in June 2026 while effort moved toward a private edition/Quantcept.
- Therefore Fincept is a rich reference/capability catalogue, not a production-quality oracle.
