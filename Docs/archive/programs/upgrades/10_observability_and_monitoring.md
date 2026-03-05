# Observability and Monitoring

Observability is the practice of making a system’s internal state visible through logs, metrics and traces. For the Jax trading assistant, observability is essential to detect and diagnose failures, understand performance bottlenecks and ensure compliance.

## Why it matters

A trading system operating without observability is a black box. You need to know how long each stage takes, where errors occur, and how data flows through the system to maintain reliability and trust.

## Tasks

1. **Structured logging**
   - Adopt a structured logging library (e.g. [zerolog](https://github.com/rs/zerolog) or [log/slog](https://pkg.go.dev/log/slog)) for all services.
   - Include correlation IDs, timestamps, and service names in every log entry.
   - Ensure sensitive information (account numbers, private keys) is redacted.

2. **Metrics collection**
   - Instrument code using Prometheus client libraries to expose metrics such as:
     - Request counts and latencies per endpoint.
     - Processing time per pipeline stage (event detection, generation, risk, research, storage, execution).
     - Error counts and types.
     - Data ingestion freshness and volume.
   - Expose a `/metrics` HTTP endpoint on each service and configure a Prometheus server to scrape it.

3. **Distributed tracing**
   - Integrate OpenTelemetry (OTel) to capture traces across services. Use the correlation ID as a trace or span identifier.
   - Export traces to a backend like **Jaeger** or **Tempo** and visualise them to understand call graphs and latency breakdowns.

4. **Alerting**
   - Define alert rules based on metric thresholds (e.g. high error rate, slow processing time, stale market data). Use Alertmanager or a similar tool to route alerts to email, Slack or PagerDuty.
   - Document run books for common alert scenarios and include steps for triage and remediation.

5. **Log aggregation**
   - Forward logs from all services to a central store (e.g. **Loki**, **ELK stack**, or **Fluent Bit**). Enable full‑text search and indexing by correlation ID.
   - Configure log rotation and retention to avoid disk exhaustion.

6. **Health and readiness probes**
   - Implement `/health` and `/ready` endpoints in each service that check dependencies (database, market data provider, Dexter, broker).
   - Configure Kubernetes or your orchestrator to use these probes to manage service restarts.
