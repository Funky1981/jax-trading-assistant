# PAPER-02A2 — Genuine Prospective Event Intake Closure

Status: **IMPLEMENTED / EXTERNAL CONFIGURATION REQUIRED**.

PAPER-02 remains **READY_FOR_EXTERNAL_REVIEW / NOT ACTIVE**, with a sample of
`0 / 50`. This package does not create a pilot identity, admit an opportunity,
or activate a prospective pilot.

## Actual source

The canonical source is the sibling `Jax-World-News-Monitor` integration
service. Its collector reads configured RSS/Atom feeds, persists append-only
`world_monitor_events`, and exposes:

```text
GET /api/v1/jax/events?after=<cursor>&limit=<1..250>
```

Jax consumes that endpoint through `cmd/trader/world_monitor_pull_worker.go`,
whose stable consumer identity is `jax-genuine-event-pull-v1` and whose provider
contract is `world-monitor-events/v1`. No parallel ingestion path was added.

The default sibling collector feeds are BBC World, CNBC Top News, and Federal
Reserve press releases. They are source configuration, not synthetic fixtures.
The source service is optional and is not started by the core Jax profile.

## Operator configuration gate

For a genuine future source, the operator must start the sibling
`world-monitor` Compose profile and set:

```text
WORLD_MONITOR_PULL_ENABLED=true
WORLD_MONITOR_EVENTS_URL=http://worldmonitor-events:8082/api/v1/jax/events
```

When Jax runs on the host while the source port is published, the endpoint is
the equivalent host address:

```text
WORLD_MONITOR_EVENTS_URL=http://127.0.0.1:8082/api/v1/jax/events
```

`WORLD_MONITOR_FEEDS_JSON` may explicitly replace the source service's default
RSS/Atom feeds with `{id,name,url}` objects. The source service has no required
credential for its default feeds. If a future feed requires credentials, those
credentials must be supplied to the source service through its own secret
configuration; they must not appear in Jax URLs, hashes, logs, or read models.

The Jax pull configuration remains disabled by default. It accepts only an
absolute HTTP(S) URL targeting `/api/v1/jax/events`; credentials, query strings,
fragments, arbitrary paths, and placeholder URLs fail closed. The readiness
projection exposes only the source identity, provider contract, and a SHA-256
endpoint identity.

## Prospective semantics and identity

The source's `event_id` and `identity_key` are retained as canonical source
identity. The source's `first_seen_at`, `collected_at`, `last_seen_at`, source
publication timestamp, source URL, native ID, content hash, raw payload, and
provenance are retained. A missing publication timestamp remains `null` and is
reported as `UNKNOWN`; collection time is retained separately and is never
written into the publication-time field.

The existing serializable cursor/page transaction remains authoritative. It
retains exact provider page bytes and a digest, locks the endpoint cursor,
idempotently ingests source events and decisions, then advances the cursor.
Repeated delivery reuses the existing inbox and decision identities. Conflicting
identity reuse fails closed through the existing source-event/dedupe contracts.

## Evidence and candidate compatibility

Ingested events continue through the existing World Monitor research inbox,
normalisation, `assetresolution.Resolver`, `genuine_event_decisions`, and
`candidate_evidence_scores` path. Unknown or ambiguous issuer/instrument
resolution stays unresolved. Insufficient evidence remains `NO_TRADE` or
`WATCH`; no event is forced to become a candidate and no paper intent is
created by intake.

The existing pilot boundary also remains authoritative: an opportunity whose
immutable `FirstSeenAt` precedes pilot activation is rejected. Events observed
before a future PAPER-02 activation may remain in the evidence store but cannot
enter the PAPER-02 opportunity sample.

## Readiness and remaining gate

The protected PAPER-02 readiness surface reports:

- `jax-genuine-event-pull-v1`;
- `world-monitor-events/v1`;
- safe endpoint identity, when configured;
- the exact blocking reason when disabled or invalid;
- PAPER mode and the existing broker/live safety state.

For an enabled configuration, readiness also performs a bounded read-only GET
of the source service's derived `/health` endpoint. A valid URL without a
healthy collector is still blocked. With the repository's default
configuration, `WORLD_MONITOR_PULL_ENABLED` is `false`, so
`ProspectiveEventIntakeReady=false` and `OverallReady=false`.
No connectivity proof was claimed and no pilot identity or opportunity was
created. External operator configuration of the intended World Monitor profile
and endpoint remains required before PAPER-02 activation review.

## Safety

This package does not configure or invoke broker/live execution, `execution.Service`,
IB, historical replay, formal evidence, strategy tuning, Context Engineering,
or Phase 13. The only possible trading venue remains the isolated simulated
paper venue, and PAPER-02 remains inactive.
