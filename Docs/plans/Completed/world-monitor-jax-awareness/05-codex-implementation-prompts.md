# 05 — Codex Implementation Prompts

## Prompt 1 — Add Research Trigger Contract

Add a new World Monitor research-trigger contract to Jax.

Requirements:

- Create a typed request model for World Monitor research triggers.
- Include required fields: source, source_event_id, event_type, headline, source_urls, source_count, timestamp_utc, possible_affected_etfs, confidence, reason, raw_payload.
- Add validation that rejects missing timestamp, missing source URLs, insufficient source count, stale events, and trade-execution language.
- Add unit tests for valid and invalid payloads.
- Do not create trades, candidates, orders, or broker instructions in this prompt.

## Prompt 2 — Add Ingestion Endpoint

Add a Jax research ingestion endpoint for World Monitor.

Suggested route:

```http
POST /research/events/world-monitor
```

Requirements:

- Accept the research-trigger contract.
- Validate request.
- Deduplicate events.
- Persist raw event metadata.
- Return a receipt containing event id, status, and rejection reason if rejected.
- Add tests for accepted, duplicate, and rejected triggers.

## Prompt 3 — Map To Existing Event Tables

Wire accepted World Monitor triggers into Jax event storage.

Requirements:

- Store raw payload in the existing raw event path.
- Store normalised headline, summary, timestamp, source, and event type.
- Store ETF mappings in the existing event-symbol mapping flow.
- Store source URLs and confidence in attributes/evidence metadata.
- Do not bypass existing ETF allowlist policy.

## Prompt 4 — Add Research Review Queue

After ingestion, queue or mark the event for Jax research review.

Requirements:

- Add event status transitions: received, validated, rejected, researching, candidate_created, candidate_rejected, archived.
- Add tests for status transitions.
- Ensure no candidate trade is created until the research pipeline completes required checks.

## Prompt 5 — Add Local Adapter Skeleton

Create a separate local adapter service skeleton named:

```text
world-monitor-jax-adapter
```

Requirements:

- Runs locally.
- Reads World Monitor-style events from a file, local endpoint, or manual JSON input.
- Converts them to the Jax research-trigger contract.
- Posts to Jax research endpoint.
- Logs request/response locally.
- Has no broker credentials and no trade execution capability.

## Prompt 6 — Add End-to-End Smoke Test

Create a smoke test proving:

1. Adapter sends a fake macro event.
2. Jax accepts it as a research trigger.
3. Jax stores it.
4. Jax does not create a trade automatically.
5. Jax returns a research status/receipt.

The test must fail if an order, execution instruction, or approved trade is created directly from the World Monitor payload.

## Prompt 7 — Add Separate-System Control Layer

Implement the World Monitor → Jax boundary as a separate adapter and Jax research inbox. Do not merge World Monitor into Jax and do not allow World Monitor to create trade candidates directly.

Acceptance criteria:

- Add a `world_monitor_research_inbox` concept/table or equivalent holding queue.
- Add statuses: `new`, `ignored`, `researching`, `candidate_created`, `rejected`, `archived`.
- Add adapter-side validation for allowed themes, freshness, source count, source quality, duplicate detection, and explainable ETF mapping.
- Add severity levels: `low`, `medium`, `high`, `critical`.
- Add source quality tiers and confidence-reason fields.
- Add audit fields linking `world_monitor_event_id` to Jax research/candidate IDs.
- Enforce that World Monitor payloads cannot create broker orders, execution instructions, or risk overrides.
- Add tests proving raw World Monitor alerts only enter the research inbox and never bypass evidence gates.

