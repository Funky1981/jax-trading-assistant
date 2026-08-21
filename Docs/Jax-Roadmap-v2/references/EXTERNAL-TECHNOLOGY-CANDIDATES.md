# External Technology Candidates

These are candidates, not pre-selected dependencies.

## Backtesting and research

### LEAN
Open-source institutional-calibre algorithmic engine supporting research, backtesting and live trading, with modular data/brokerage integrations and Python/C# algorithms. Best candidate when Jax needs realistic event-driven execution simulation across multiple asset classes. It may be too heavy for simple recommendation replay.

Official: https://www.quantconnect.com/docs/v2/lean-engine

### VectorBT
Python/NumPy/pandas-oriented vectorized backtesting and analysis, accelerated with Numba and optional Rust kernels. Strong candidate for rapid parameter sweeps, factor/signal research and large comparative experiments. It is not a substitute for Jax's event/evidence replay engine.

Official: https://vectorbt.dev/

### Qlib
Microsoft's AI-oriented quant research platform covering data processing, models, backtesting, risk, portfolio optimization and automated research integration with RD-Agent. Strong Phase 12 research candidate, not an early Jax runtime dependency.

Official: https://github.com/microsoft/qlib

## Portfolio/risk

### skfolio
Scikit-learn-style portfolio optimization and risk framework with cross-validation/model-selection semantics and measures including Sharpe/CVaR/drawdowns. Attractive because leakage/overfitting controls fit Jax's evaluation philosophy.

Official: https://skfolio.org/

### Riskfolio-Lib
CVXPY-based portfolio optimization library with broad risk measures, risk parity, Black-Litterman, factor constraints and advanced objectives. Strong advanced-risk candidate; likely excessive for the first quant core.

Official: https://riskfolio-lib.readthedocs.io/

## Primary evidence sources

### SEC EDGAR
Official unauthenticated JSON APIs for company submissions and extracted XBRL company facts; bulk archives are available and feeds are updated in near-real time. Prefer official SEC data over third-party repackaging for canonical corporate evidence where possible.

Official: https://www.sec.gov/search-filings/edgar-application-programming-interfaces

### FRED / ALFRED
Official Federal Reserve Bank of St. Louis APIs for economic series, releases and historical vintages. Vintage/revision semantics matter for leakage-safe historical replay.

Official: https://fred.stlouisfed.org/docs/api/fred/

### ACLED
Structured conflict/event API suitable for geopolitical corroboration. Access/authentication/licensing must be checked before adopting into Jax.

Official: https://acleddata.com/acled-api-documentation

### AISStream
WebSocket AIS maritime data. Useful only as an alternative evidence source; service documentation currently describes it as beta with no SLA, so it must never be treated as a sole high-confidence source.

Official: https://aisstream.io/documentation.html

### Polymarket
Public Gamma/Data APIs and public CLOB market-data endpoints expose prediction-market discovery/prices/orderbooks. For Jax, treat as read-only market-implied evidence; do not connect authenticated trading operations.

Official: https://docs.polymarket.com/api-reference/introduction

## Selection rule

For each adoption decision record:
- exact requirement;
- candidate(s);
- maturity/maintenance;
- licence/terms;
- runtime/operational cost;
- reproducibility;
- failure modes;
- testability;
- why selected;
- explicit revisit trigger.
