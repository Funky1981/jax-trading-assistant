# Research Notes — Why the roadmap changed

## Fincept lessons that materially changed Jax planning

1. **Data plane before more intelligence.** Fincept's DataHub treats freshness, errors, last-known-good state and producer ownership as first-class concerns. Jax should adopt the principle even if its implementation differs.
2. **Normalization + raw provenance is foundational.** Provider-specific fields should terminate at adapters, not leak through research logic.
3. **Connector count is not maturity.** Fincept exposes a huge catalogue, but part of the current 190-connector manifest is explicitly described as not invoked by real callsites. Jax needs a qualification registry, not a connector race.
4. **Replay is a product capability.** Alpha Arena's persisted contexts/decisions/risk/orders/events show that full reconstruction should be designed in, not added later.
5. **MCP/tool authorization matters.** Schema validation, cancellation/timeouts and destructive-tool confirmation are stronger design references than persona-style agents.
6. **Durable research agents are useful only after tools and evidence exist.** Checkpointing, budgets, reflection and evaluation are the useful parts.
7. **Established quant libraries remove low-value reinvention.** Fincept's broad analytics wrappers prompted direct evaluation of skfolio/Riskfolio/VectorBT/Qlib/LEAN rather than porting Fincept maths.
8. **Workflow and risk must be deterministic.** Fincept's separate risk/confirmation/audit components align with Jax's paper-safe architecture.
9. **Public Fincept is a reference, not a dependency.** Its public maintenance cadence reduced in June 2026 and its architecture docs include target-state sections.

## Direct external-source findings

- SEC's `data.sec.gov` APIs expose submissions and XBRL data without API keys and provide bulk archives; this is a strong canonical company-data source.
- FRED supports real-time/vintage semantics, which is especially important for avoiding hindsight leakage in historical economic replay.
- LEAN is strong for realistic event-driven backtesting/live-style simulation.
- VectorBT is strong for high-volume vectorized research and parameter sweeps.
- Qlib is strong for advanced ML/factor research and integrates with automated R&D concepts.
- skfolio/Riskfolio provide mature portfolio/risk optimization machinery; they should be evaluated rather than reimplemented.
- AISStream explicitly states beta/no SLA, so it belongs in corroborating evidence, not as a sole truth source.

## Research boundary

This pack does not assume that Fincept documentation proves runtime quality. Each phase explicitly requires callsite/test/maturity verification before borrowing a concept.
