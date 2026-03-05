# Decision Audit Model (verify every AI decision)

## What “verify” means here
You cannot prove an AI output is “true” in general.
You can prove:
- exactly what it saw
- exactly what it produced
- what rules accepted/rejected it
- what the system did because of it
- that it can be replayed later

That is how you make it trustworthy.

---

## Decision Graph (end-to-end trace)
Every run (live/paper/backtest/research) gets:
- `correlation_id` (one per top-level action)
- `run_id` (backtest run, research run, or live session run)
- `instance_id` (strategy instance)
- `symbol` (optional)

Each step appends nodes:
1) Data fetch
2) Event detection
3) AI classification/extraction (optional)
4) Strategy evaluation (deterministic)
5) Risk calc (deterministic)
6) Signal persisted
7) Approval decision (human or policy)
8) Order intent created
9) Order submitted to broker
10) Fills received
11) P/L and reconciliation

You verify decisions by inspecting this chain.

---

## Minimum audit record per AI decision
Store one row per AI call:
- `id` (uuid)
- `timestamp`
- `correlation_id`
- `run_id`
- `instance_id`
- `symbol`
- `purpose` (e.g. `classify_news`, `extract_guidance`, `summarise_run`)
- `model_provider` (ollama/openai/etc.)
- `model_name`
- `prompt_template_id` + `prompt_version`
- `inputs` (JSONB)
  - raw text (news/earnings)
  - derived context (recent candles summary, etc.)
- `tool_calls` (JSONB) if any
- `output_raw` (JSONB or text)
- `output_structured` (JSONB) (your parsed schema)
- `confidence` (float 0..1) if produced
- `latency_ms`
- `token_usage` (if available)
- `error` (nullable)

### Output must be schema validated
Do not accept “free text”.
Parse into a strict schema:
- `event_type`
- `severity`
- `direction_bias`
- `tickers[]`
- `rationale[]`
- `unknowns[]`

If parsing fails -> mark decision as `invalid` and do not use it.

---

## Deterministic acceptance record
Every place AI output could influence anything, record:
- `ai_decision_id`
- `accepted` boolean
- `rejection_reason` if false
- `rules_applied[]`
- `final_inputs_used` (what deterministic layer used after sanitisation)

This is how you prevent silent drift.

---

## Replay requirement (critical)
You must be able to replay:
- the exact AI prompt version
- the exact inputs
- the same deterministic pipeline version

Even if model outputs differ later, you will retain the original for audit.

