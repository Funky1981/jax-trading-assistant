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

The data-only `config/ai-shadow-issuer-diagnostic-manifest-v1.json` freezes 48 independently adjudicated issuer-recognition cases, with six cases in each of eight review categories. It remains separate from the UUID/FK-backed historical benchmark. `config/ai-shadow-issuer-diagnostic-input-fingerprints-v1.json` freezes the canonical model-visible `EventInput` fingerprint for every symbolic diagnostic ID.

The diagnostic-specific command performs a file-backed, append-only audit beneath `.runtime/diagnostics/ai-shadow-issuer/`; it does not connect to the database or reuse operational benchmark rows. Preflight verifies the exact manifest bytes and semantic fingerprint, every event-input fingerprint, prompt and output-contract versions, policy version, 48-case order, three-repetition shape, model configuration, and paper-mode safety. It deliberately exits without contacting Ollama:

This separation is intentional. The historical benchmark command accepts UUID event references that must already exist in `world_monitor_research_inbox`, then joins qualified outcomes and persists through UUID foreign keys. The issuer diagnostic instead freezes complete receipt-time inputs under symbolic case IDs and has no outcome dependency. Rewriting those IDs, inserting synthetic inbox rows, weakening the operational manifest loader, or changing the v54 foreign keys would alter the frozen diagnostic or contaminate operational benchmark storage, so those alternatives are rejected.

```powershell
$env:JAX_AI_SHADOW_ENABLED = "true"
$env:JAX_AI_PROVIDER = "ollama"
$env:JAX_AI_MODEL = "ministral-3:8b"
$env:JAX_AI_BASE_URL = "http://localhost:11434"
$env:JAX_AI_TIMEOUT_SECONDS = "120"
$env:JAX_AI_TEMPERATURE = "0"
$env:JAX_AI_SEED = "20260803"
$env:JAX_AI_MAX_EVENTS = "48"
$env:JAX_RUNTIME_MODE = "paper"
$env:ALLOW_LIVE_TRADING = "false"
$env:EXECUTION_ENABLED = "false"
$env:EXECUTION_INSTRUCTION_WORKER_ENABLED = "false"
$env:BROKER_EXECUTION_ALLOWED = "false"
$env:MAX_LEVERAGE = "1"
go run ./cmd/ai-shadow-issuer-diagnostic --preflight
```

`--execute` is a separate, fail-closed action that always runs exactly three complete repetitions in frozen event order and allows only the committed single corrective retry per case. It requires separate inference authorization; never substitute it for preflight during implementation or review.

## Predeclared verdicts

The thresholds are fixed in code before any model batch: structural validity at least 98% after at most one retry; fabricated/invalid ticker rate at most 2%; direct mapping agreement at least 85%; AI HIGH 1-hour median movement at least 0.05 percentage points and 10% above LOW/UNCERTAIN; and at least a 0.05 percentage-point separation improvement over the deterministic baseline at either 1 hour or 1 day.

- `AI VALUE DEMONSTRATED` requires all thresholds.
- `MIXED` means operational validity and mapping thresholds pass but improvement is inconsistent or too small. Any run below 60 events is explicitly smoke-only and receives this non-final verdict.
- `NOT DEMONSTRATED` means useful improvement is absent or validity/mapping error thresholds fail.
- `INVALID` is a non-zero execution failure caused by leakage-invariant failure, a corrupt/non-reproducible manifest, missing qualified outcomes, unsafe state or mutation, or unavailable required persistence/dependencies.
