# Read-only AI shadow benchmark

The benchmark sends a fixed set of genuine historical receipt-time event records to local Ollama, validates the structured responses, and writes only to isolated `ai_shadow_benchmark_*` tables. It does not change deterministic decisions or create candidates, approvals, tickets, execution instructions, orders, trades, or fills.

Apply migration `000054_ai_shadow_benchmark` before the first run. Ollama must remain locally bound and the configured model must already be installed.

Host run (60-event fixed benchmark):

```powershell
.\scripts\run-ai-shadow-benchmark.ps1
```

The script shows the exact manifest, event count, model, configuration, worst-case runtime, and command. It performs a no-inference preflight and requires `RUN` confirmation. For an explicitly automated repeat run, add `-NonInteractive`. For the bounded three-event smoke, add `-MaxEvents 3`.

Host Ollama defaults to `http://localhost:11434`. For a command executed inside a container, set `JAX_AI_BASE_URL=http://host.docker.internal:11434`. All `JAX_AI_*` values are configurable and the Go command fails closed if any required value is absent or invalid.

Reports are written beneath `.runtime/ai-shadow/<run-id>/` as Markdown, JSON, CSV, and a copied manifest.

## v4 issuer-resolution contract

`ai-shadow-output-v4-issuer-resolution` makes the model classify issuer or exposure identity without producing any ticker. A `DIRECT` output carries `direct_issuer`; a `PROXY` output carries one bounded `proxy_exposure`; and `UNRESOLVED` carries neither. Jax then applies `event-asset-resolution-v1` independently. The persisted JSONB envelope and JSON/CSV/Markdown reports keep the model classification, deterministic resolution provenance, and frozen reference comparison separate.

Direct issuer matching reuses the existing canonical issuer and alias rules. Matching is exact after documented normalization; unknown issuers remain valid but unresolved, collisions remain ambiguous, and share-class-specific rules never silently select a ticker. No v2 asset policy or database migration is required for this contract.

The data-only `config/ai-shadow-issuer-diagnostic-manifest-v1.json` freezes 48 independently adjudicated issuer-recognition cases, with six cases in each of eight review categories. It is deliberately not wired to the benchmark command in this implementation phase and must not be run without separate authorization.

## Predeclared verdicts

The thresholds are fixed in code before any model batch: structural validity at least 98% after at most one retry; fabricated/invalid ticker rate at most 2%; direct mapping agreement at least 85%; AI HIGH 1-hour median movement at least 0.05 percentage points and 10% above LOW/UNCERTAIN; and at least a 0.05 percentage-point separation improvement over the deterministic baseline at either 1 hour or 1 day.

- `AI VALUE DEMONSTRATED` requires all thresholds.
- `MIXED` means operational validity and mapping thresholds pass but improvement is inconsistent or too small. Any run below 60 events is explicitly smoke-only and receives this non-final verdict.
- `NOT DEMONSTRATED` means useful improvement is absent or validity/mapping error thresholds fail.
- `INVALID` is a non-zero execution failure caused by leakage-invariant failure, a corrupt/non-reproducible manifest, missing qualified outcomes, unsafe state or mutation, or unavailable required persistence/dependencies.
