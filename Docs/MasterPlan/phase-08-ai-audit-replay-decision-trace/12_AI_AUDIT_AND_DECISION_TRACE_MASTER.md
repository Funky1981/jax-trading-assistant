# AI Audit and Decision Trace Master Plan

## AI role
Allowed: classify/extract/summarize.
Not allowed: place trades, size positions, override risk, bypass controls.

## Required AI audit capture
- ai_decision_id, run_id, correlation_id, instance_id, symbol
- purpose, provider/model
- prompt template/version
- input payload reference
- raw output + structured output
- schema validation result
- latency/tokens/errors
- acceptance/rejection record with rule reasons

## Replayability
Must replay prompt version + inputs + deterministic acceptance rules + strategy/risk versions.
