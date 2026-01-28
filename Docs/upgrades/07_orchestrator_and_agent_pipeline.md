# Orchestrator and Agent Pipeline

The orchestrator coordinates the core flow: detect events → generate trades → calculate risk → enrich with research → persist results. It is the heart of the trading assistant, and its current implementation is skeletal.

## Why it matters

In production, the orchestrator must be reliable, fault‑tolerant and extensible. It should ensure that one component’s failure does not halt the entire pipeline and that correlation IDs and audit logs persist across service boundaries.

## Tasks

1. **Complete the `jax-orchestrator` service**
   - Implement the gRPC or HTTP endpoints for orchestrator operations (e.g. `/process`, `/status`).
   - Leverage concurrency to process multiple symbols or events in parallel while respecting rate limits and system capacity.
   - Handle retries and partial failures gracefully: if trade generation fails for one event, continue processing other events.

2. **Integrate memory service**
   - Use the `jax-memory` service (Hindsight facade) to recall previous trades, notes or reflections relevant to the current symbol.
   - When processing a symbol, store key observations (trade rationale, risk notes) via `memory.retain` and retrieve them via `memory.recall` for context.
   - Schedule periodic `memory.reflect` calls to summarise past performance and feed insights back into strategy selection.

3. **Correlation IDs and audit propagation**
   - Ensure each request context carries a correlation ID (via `EnsureCorrelationID`) from the API gateway through the orchestrator into downstream services.
   - Propagate the audit logger across service boundaries so that events within `jax-memory`, `jax-ingest`, or Dexter are logged under the same ID.

4. **Error isolation and timeouts**
   - Set sensible timeouts for each pipeline stage (e.g. market data fetch, strategy evaluation, risk calculation, research). Abort or skip stages that exceed the limit and log the reason.
   - Circuit‑breakers should prevent repeated calls to failing components and enable graceful degradation.

5. **Extensibility hooks**
   - Expose a plugin interface for adding new stages (e.g. sentiment analysis, alternative data ingestion). The orchestrator should call them in order and handle their results uniformly.
   - Document how to add a new stage, including expected input/output and error behaviour.

6. **Monitoring and metrics**
   - Capture per‑stage latency, success/failure counts, and queue depths as Prometheus metrics.
   - Add tracing (e.g. OpenTelemetry) to visualise the end‑to‑end flow across services.