# Jax Commercial-Readiness CR-02B — Jax-Native Planner + Complete Agent0 Removal

Date: 2026-09-09
Repository: `C:\Projects\Jax\jax-trading-assistant`
Branch: `capability-reset`
Starting HEAD: `8ddce7e8c6e08d1e1fad539298453fa621645551`
Scope: replace the supported Agent0 planning capability, then remove Agent0
from the supported Jax runtime. CR-02C/CR-02D and Phase 13 were not started.

## Decision and resulting invariant

External technical-lead authorization was `GO PHASE 12`, `GO CR-01`,
`GO CR-02A`, followed by authorization for CR-02B. The resulting supported
runtime invariant is:

**AGENT0 IS NOT PART OF THE SUPPORTED JAX RUNTIME.**

Jax now owns its planning contract and orchestration. The planner is an
advisory boundary only. It cannot approve, execute, invoke a broker, create an
order, mutate portfolio state, or invoke unrestricted tools.

## Pre-removal dependency forensic

The fresh post-Dexter search classified Agent0 references as follows:

| Class | Findings | Disposition |
| --- | --- | --- |
| `ACTIVE_RUNTIME` | `cmd/research` and `internal/modules/orchestration` used `libs/agent0` planning client; Compose started `agent0-service` | Replaced with `internal/modules/planner`; service dependency removed |
| `ACTIVE_CONFIG` | `AGENT0_SERVICE_URL`, Agent0 provider/memory/IB variables, port 8093 Compose wiring | Removed from supported Compose, example configuration and startup scripts |
| `ACTIVE_TEST` | Agent0 client/mock, orchestration fakes, frontend suggestion tests, Agent0 CI/import checks | Replaced or removed; native planner tests added |
| `ACTIVE_DOC` | README, architecture, operations, setup, testing and roadmap references | Updated; historical cleanup/audit records retain explicit supersession notes |
| `EMBEDDED_EXTERNAL_SOURCE` | `Agent0/` (1,016 tracked files), `services/agent0-service` (12 tracked files) | Removed under this authorized cleanup |
| `ARCHIVE/HISTORICAL` | `Docs/archive/`, legacy `archive/`, historical evidence/plans and prior cleanup bodies | Preserved; not part of supported runtime |

The tracked Agent0 roots removed by this package were:

- `Agent0/` — 1,016 tracked files;
- `services/agent0-service/` — 12 tracked files;
- `libs/agent0/` — 4 tracked files;
- `scripts/bootstrap-agent0.ps1` — retired bootstrap script.

The supported capability map showed that only the planning operation was
consumed by the current `cmd/research` composition. Agent0 Execute and the
Python `/suggest`, `/chat`, `/v1/execute`, `/ready` and provider-specific
surfaces were not required by supported Go orchestration. They were not
reimplemented. The frontend Agent0 suggestion path was removed because it was
an Agent0-only route that could promote directly into candidate/approval flows;
Jax's existing deterministic scanner/opportunity APIs remain separate.

## Native planner contract

`internal/modules/planner` defines the Jax-owned `Planner` interface:

```go
type Planner interface {
    Plan(context.Context, Request) (Result, error)
}
```

`Provider` is a small replaceable boundary behind the service. `Request` carries
symbol, bounded context, constraints, recalled memory and strategy signals.
`Result` is structured advisory output. Boundary validation rejects missing or
unbounded fields, non-finite confidence, and authority-bearing action labels
such as `APPROVE`, `EXECUTE`, `ORDER`, `BROKER`, `SUBMIT` and `TRADE`.

The current default provider is `DeterministicProvider`, a zero-cost local
provider whose bounded output is `HOLD`. A selectable `ModelProvider` now
adapts the existing Jax `llmcontext.LiteLLMClient` transport to the same
Planner contract. It sends only bounded planner context and a strict output
schema; it receives no tool catalog or execution API. No hosted inference or
paid API call is part of CR-02B. The Jax ToolRunner remains a separate bounded
boundary and does not execute arbitrary planner steps.

`NewConfiguredPlanner` selects deterministic/offline mode when
`JAX_PLANNER_PROVIDER` is absent or explicitly `deterministic`/`offline`. The
model-backed route is opt-in with `JAX_PLANNER_PROVIDER=litellm` and requires
the existing `AI_GATEWAY_BASE_URL` and `AI_GATEWAY_API_KEY` configuration plus
an optional `JAX_PLANNER_MODEL`/`AI_DEFAULT_MODEL`. Missing or unknown
configuration fails closed; there is no silent fallback from a failed model
request to a deterministic result.

## CR-02B native-planner capability closure

The final capability closure is:

- `JAX-NATIVE AI PLANNING CAPABILITY PRESERVED`: YES;
- `REAL LLM-BACKED PLANNING CAPABILITY PRESERVED`: YES, through the existing
  Jax LiteLLM transport and the opt-in `ModelProvider`;
- `DETERMINISTIC ZERO-COST TEST PROVIDER RETAINED`: YES;
- structured JSON is decoded with unknown-field rejection and then passed
  through the existing planner validation, including forbidden authority
  actions and bounded fields;
- provider failures, malformed JSON, invalid schema results, unsafe actions,
  invalid confidence, and missing model configuration fail closed;
- provider/model identity and available usage metadata are retained in planner
  results and orchestration audit records;
- all capability tests use fakes/local fixtures; hosted inference spend was
  `$0.00`;
- Agent0 remains removed and `AGENT0 IS NOT PART OF THE SUPPORTED JAX RUNTIME`.

For the deterministic default, orchestration audit identity is
`Provider=jax-planner` and `Model=jax-planner/v1`; model-backed runs retain the
truthful selected provider/model identity. Observability uses `planner_plan`.
No Agent0 Execute interface remains in supported orchestration.

## Safety and data boundaries

The following remain unchanged and are regression-checked:

- `ALLOW_LIVE_TRADING=false`;
- `BROKER_EXECUTION_ALLOWED=false`;
- live execution worker disabled;
- maximum leverage `1x`;
- no recommendation/candidate/approval/ticket/order/trade/fill/portfolio
  mutation caused by this cleanup;
- no credentials, private market data or hosted-inference payloads added.

Historical Agent0 and Dexter references in archived/audit evidence are not
active runtime dependencies. The prior Agent0 source/service were removed only
after the native planner focused tests passed in commit
`fec0bdb` (`Introduce Jax-native advisory planner boundary`).

## Verification record

The implementation was verified before the bounded cleanup commit, with no
code changes after those checks:

- `go test ./... -count=1`: PASS;
- `go vet ./...`: PASS;
- native planner/orchestration/observability focused tests: PASS;
- frontend lint: PASS;
- frontend typecheck: PASS;
- frontend Vitest: 56 files and 155 tests passed;
- `docker compose config --quiet`: PASS;
- PowerShell parse checks for `start.ps1` and the modified platform scripts:
  PASS;
- active Agent0/configuration/deployment reference audit: PASS for the
  supported tree; archived and historical records remain preserved;
- `git diff --cached --check` and `git diff --check`: PASS;
- race detector: not rerun for this cleanup; the accepted Phase 10/11 result
  remains `CGO_ENABLED=1 go test -race ./internal/modules/workflow
  ./internal/modules/papertrading -count=1` under the existing Docker toolchain;
- no paid spend and no new credentials.

## Remaining review position

This is an implementation record, not a self-awarded external GO. External
technical-lead review must decide the CR-02B result. CR-02C/CR-02D and Phase 13
remain not started; the scientific status remains
`TRADING EDGE NOT DEMONSTRATED / INSUFFICIENT SAMPLE`.
