# Phase 06 Requirement — LLM Context Budgets and Incremental Research

## Status

Cross-cutting requirement to be incorporated into Phase 06 work-package planning. This file does not add or authorize a new numbered work package.

## Objective

Ensure the research/recommendation engine constructs bounded, provenance-preserving evidence contexts rather than allowing prompts to grow with the size of Jax's database or research history.

## Required capabilities

Phase 06 research contracts should support:

- task-specific evidence packets;
- explicit evidence/source IDs and fingerprints;
- source freshness/vintage metadata;
- duplicate/near-duplicate suppression before inference;
- retrieval of only decision-relevant evidence;
- deterministic metrics as compact typed fields;
- explicit contradiction/unknown representation;
- context-budget accounting before provider invocation;
- incremental research based on changed/new evidence;
- provenance-preserving summaries where raw evidence is not required;
- references back to retained raw evidence;
- explicit oversize behaviour instead of silent truncation.

## Recommended context shape

```text
task identity + contract
        ↓
current objective
        ↓
small canonical entity/portfolio/event state
        ↓
selected evidence packet
        ↓
known contradictions / unknowns
        ↓
required output
```

Do not place arbitrary historical transcript or whole-source corpora into the default context.

## Token budget fields

A future research task should be able to declare target/max input tokens, max output/reasoning tokens, max evidence items/chunks, max retries, maximum model tier and maximum estimated cost.

## Incremental research

If prior research state exists, identify evidence unchanged, added, superseded or invalidated; conclusions dependent on changed evidence; and unresolved questions. Reconstruct only the minimum sufficient state plus changed evidence.

## Summary safety

A compact summary must retain source/fingerprint references, preserve uncertainty/contradiction, not alter material numbers silently, not replace exact legal/financial wording when material, record its generation/version, and leave raw evidence available for replay.

## Acceptance conditions for Phase 06 planning

Future Phase 06 packages must define and test context-size/token-budget behaviour, evidence-selection provenance, duplicate suppression, oversize handling, incremental evidence, preservation of material contradictions and replay from exact evidence identities.
