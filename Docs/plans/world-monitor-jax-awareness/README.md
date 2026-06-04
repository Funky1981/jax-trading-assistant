# World Monitor → Jax Awareness Integration Plan

## Purpose

This folder defines how to run World Monitor separately as a personal situational-awareness dashboard and feed Jax with research triggers.

World Monitor is **not** allowed to place trades, recommend execution, or bypass Jax guardrails.

Its role is:

```text
World Monitor = radar / event discovery
Jax = research, evidence, priced-in checks, candidate generation
Human = approval
Broker = execution only after approval
```

## Non-Negotiables

- World Monitor sends **research triggers**, not trade instructions.
- Jax must verify every event before creating a candidate trade.
- No live trading path is allowed from World Monitor.
- All events must be stored with source, timestamp, and audit metadata.
- Jax must run ETF mapping, historical comparison, priced-in scoring, and confounder checks before offering any paper trade.
- Human approval remains mandatory.
- Start local-only and zero-cost before considering hosting.

## Recommended Folder Order

1. `01-local-runtime-setup.md`
2. `02-signal-contract.md`
3. `03-jax-ingestion-flow.md`
4. `04-guardrails-and-safety.md`
5. `05-codex-implementation-prompts.md`
6. `06-test-plan.md`

## Target Flow

```text
World Monitor spots event
  ↓
Adapter converts event into Jax research trigger
  ↓
Jax stores raw + normalised event
  ↓
Jax verifies sources and timestamps
  ↓
Jax maps event to ETFs
  ↓
Jax checks historical reactions
  ↓
Jax checks priced-in risk
  ↓
Jax checks confounders
  ↓
Jax may create paper-trade candidate
  ↓
Human approves/rejects
```

## Success Criteria

The integration is successful only when:

- World Monitor can run independently.
- The adapter can send a test research trigger to Jax.
- Jax stores the trigger without treating it as a trade instruction.
- Jax can reject bad/stale/low-evidence triggers.
- Jax can produce an evidence-backed candidate only after its own research process.

## Final Separate-System Architecture

World Monitor and Jax must remain separate systems:

```text
World Monitor = awareness radar
Jax = research, evidence, candidate-trade engine
Adapter = controlled bridge between them
```

World Monitor is allowed to say: "this looks important".
Jax decides whether the event is worth researching.
Only after Jax evidence gates pass may Jax create a paper-trade candidate for human review.

Additional design details are in `07-separate-systems-control-layer.md`.

