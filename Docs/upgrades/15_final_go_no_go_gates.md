# Final Go/No‑Go Gates

Before flipping the switch on a live trading system, you need to define and adhere to a set of go/no‑go criteria. These gates ensure that all components meet functional, performance, and compliance requirements and that you can halt trading quickly if something goes wrong.

## Why it matters

Missing a critical check can lead to severe financial loss or regulatory issues. A disciplined sign‑off process gives stakeholders confidence and protects the business.

## Tasks

1. **Define acceptance criteria**
   - All unit, integration and end‑to‑end tests must pass on the main branch.
   - Audit logging must capture every trade decision, risk calculation, research call and execution with a correlation ID.
   - The risk engine must enforce portfolio and position constraints under simulated load.
   - Market data ingestion must be live, accurate and up to date for all instruments traded.
   - Paper trading results should show consistent positive expectation over a significant sample size (weeks or months).

2. **Conduct a pre‑launch review**
   - Assemble stakeholders (developers, quants, compliance, operations) to review system readiness.
   - Walk through recent trades and confirm that the rationale, risk calculations and execution details match expectations.
   - Review the run books for handling failures, including kill switches and manual overrides.

3. **Establish monitoring thresholds and kill switches**
   - Define error and latency thresholds that will trigger an automatic halt of trading (e.g. >5% of orders failing, market data stale for >30 minutes, risk engine errors >0.1% of calls).
   - Implement a kill switch to disable order placement and signal generation. Ensure this can be invoked manually by authorised personnel and automatically by alerting systems.

4. **Plan a staged rollout**
   - Start with a very small subset of symbols or a limited account balance. Monitor behaviour closely.
   - Gradually increase exposure as confidence grows and performance metrics are met.

5. **Post‑launch monitoring and continuous improvement**
   - After go‑live, review performance and audit logs daily. Identify and fix issues quickly.
   - Solicit feedback from users and adjust strategies, risk parameters, or system architecture as needed.
   - Continuously refine monitoring, risk models and strategies based on real‑world results.