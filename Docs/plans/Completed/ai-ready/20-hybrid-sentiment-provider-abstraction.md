# Add Hybrid Sentiment Provider Abstraction

## Summary

- Priority: P2
- Phase: Phase 3
- Estimate: 3-5d
- Outcome: infra flexibility

## Objective

Formalize a hybrid sentiment architecture so Jax can use third-party enrichment, self-hosted models, or both without changing product-facing contracts.

## In-scope touchpoints

- Backend adapter layer
- Sentiment scoring pipeline from Phase 2

## Implementation notes

- Define a provider interface that accepts normalized source documents and returns normalized sentiment score, confidence, drivers, source metadata, and limitations.
- Add provider configuration for disabled, local, external, and hybrid/fallback modes.
- Keep aggregation, route resolution, explainability, approval policy, and audit snapshots internal to Jax.
- Add timeout, retry, and degraded-state behavior for external providers.
- Ensure provider payloads are not leaked directly to UI contracts.

## Acceptance criteria

- Sentiment scoring can run through a provider abstraction.
- Local and external modes can be configured without changing Opportunity API contracts.
- Provider failures produce explicit degraded states and do not block unrelated trading workflows.
- Internal aggregation and policy logic remain owned by Jax.

## Suggested validation

- Run backend unit tests for provider interface, local fake provider, external failure behavior, and fallback behavior.
- Run integration tests with provider disabled to ensure existing workflows still load.
- Run replay/golden checks before enabling provider output in scanner decisions.

## Dependencies

- Builds on `10-sentiment-ingest-scoring-aggregates.md`.
