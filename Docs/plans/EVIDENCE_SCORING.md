# Evidence Scoring

> **HISTORICAL / SUPPORTING IMPLEMENTATION NOTE** — Current roadmap authority is
> `Docs/ROADMAP.md`. This document records an earlier evidence-scoring increment;
> its implementation status does not imply current-phase acceptance.

## Purpose

Evidence scoring answers one narrow question:

`Does this structured trade candidate have enough quality evidence to move toward trust-gate review?`

It does not decide whether to execute a trade.

Historical roadmap reference:

`Docs/plans/jax-trading-roadmap-pack`

## What Evidence Scoring Does

- Accepts a structurally complete trade candidate.
- Scores candidate-scoped evidence items deterministically.
- Separates supporting, contradictory, stale, and low-quality evidence.
- Produces an evidence score summary.
- Marks whether the candidate is evidence-ready for later trust-gate review.

## What Evidence Scoring Does Not Do

- Does not approve candidates.
- Does not create execution instructions.
- Does not place orders.
- Does not call broker or execution code.
- Does not size risk.
- Does not use leverage.
- Does not make BUY/SELL/HOLD decisions.
- Does not replace trust gates.

## Evidence Item Fields

- `evidence_id`
- `candidate_id`
- `source_type`
- `source_ref`
- `observed_at`
- `summary`
- `evidence_kind`
- `supports_candidate`
- `contradicts_candidate`
- `confidence`
- `impact_score`
- `quality_score`
- `freshness_status`
- `notes`

## Score Summary Fields

- `support_score`
- `contradiction_score`
- `quality_score`
- `freshness_score`
- `overall_evidence_score`
- `evidence_item_count`
- `supporting_item_count`
- `contradictory_item_count`
- `stale_item_count`
- `evidence_status`
- `evidence_ready`
- `evidence_gate_ready`
- `approval_granted`
- `broker_execution_allowed`
- `execution_instruction_created`

The last three fields are always false in this phase.

## Status Meanings

- `missing`: no candidate evidence exists.
- `weak`: evidence exists, but the score or quality is too low.
- `mixed`: both supporting and contradictory evidence exists.
- `sufficient`: fresh, high-quality supporting evidence exists with no contradiction.
- `stale`: at least one evidence item is stale.
- `blocked`: candidate structure is incomplete or evidence is only contradictory.

## Example Evidence Item JSON

```json
{
  "evidenceId": "8d4f3d1b-2c9d-4d7a-91d9-8aeb99a12d01",
  "candidateId": "b2c2fdc3-47b2-4781-9fb7-2c6e9c1190ef",
  "sourceType": "research",
  "sourceRef": "research:spx-cpi-reaction:2026-07-08",
  "observedAt": "2026-07-08T10:00:00Z",
  "summary": "Fresh CPI reaction supports broad-market continuation setup.",
  "evidenceKind": "catalyst",
  "supportsCandidate": true,
  "contradictsCandidate": false,
  "confidence": 0.9,
  "impactScore": 0.85,
  "qualityScore": 0.95,
  "freshnessStatus": "fresh",
  "notes": "Deterministic score input; not an execution signal."
}
```

## Example Evidence Score Summary JSON

```json
{
  "candidateId": "b2c2fdc3-47b2-4781-9fb7-2c6e9c1190ef",
  "supportScore": 0.727,
  "contradictionScore": 0,
  "qualityScore": 0.95,
  "freshnessScore": 1,
  "overallEvidenceScore": 0.691,
  "evidenceItemCount": 1,
  "supportingItemCount": 1,
  "contradictoryItemCount": 0,
  "staleItemCount": 0,
  "evidenceStatus": "sufficient",
  "evidenceReady": true,
  "evidenceGateReady": true,
  "approvalGranted": false,
  "brokerExecutionAllowed": false,
  "executionInstructionCreated": false
}
```

## Safety Rules

- Structurally incomplete candidates cannot become evidence-ready.
- Missing evidence returns `missing`.
- Stale evidence returns `stale`.
- Contradictory evidence reduces the overall score.
- Only contradictory evidence returns `blocked`.
- Mixed evidence can exist, but does not automatically pass.
- Evidence scoring never approves a candidate.
- Evidence scoring never creates execution instructions.
- Evidence scoring never changes broker or execution state.

## Persistence

Migration `000044_candidate_evidence_scoring` adds:

- `candidate_evidence_items`
- `candidate_evidence_scores`

Both tables reference `candidate_trades` and are additive.

## Deferred Work

- Evidence source adapters.
- Evidence ingestion from research/news/market data.
- Trust-gate review rules.
- Risk sizing and slippage-adjusted loss.
- Journal/review integration.
- Shadow-mode evidence comparisons.
- Any broker execution or live activation.
