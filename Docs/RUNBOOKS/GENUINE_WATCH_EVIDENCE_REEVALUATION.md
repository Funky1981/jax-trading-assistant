# Genuine WATCH evidence re-evaluation

This runbook describes the deterministic, read-only progression from genuine World Monitor evidence to a durable Jax evidence subject. A subject is an evolving question under review; it is not a trade, recommendation, position, or approval.

## Operator workflow

1. Start the combined paper-only stack from `C:\Projects\Jax` with `./jax-trading-assistant/jax-world-monitor.ps1 up`.
2. Confirm System Safety reports paper mode, live trading disabled, execution disabled, the execution worker disabled, broker execution disabled, and maximum leverage `1x`.
3. Open Evidence Inbox and expand a WATCH item.
4. Open **Evidence progression** to inspect the subject state, linked-event and source-group counts, first/latest timestamps, missing evidence, source independence, contradictions, and deterministic transition history.
5. Treat unknown assets and zero candidates as valid outcomes. Do not infer a ticker, direction, or recommendation from a WATCH.
6. Candidate Review remains a separate human workflow. Subject evaluation never creates approvals, paper tickets, execution instructions, order intents, broker orders, trades, or fills.

## Deterministic model

Association uses a versioned subject key built from stable category, canonical entity, specific topic anchor or proceeding, jurisdiction, and UTC publication-date bucket. A canonical URL is only a fallback for a known event category. Ambiguous or unknown evidence remains event-scoped. Broad keyword overlap and title similarity alone never merge subjects.

Primary-source hosts receive stable source groups. Exact repeated title/summary content shares a syndication group. Repeats within one group are `not_independent`; secondary evidence whose independence cannot be established is `unknown`, not independent.

The `genuine-watch-evidence-v1` evaluator considers the complete linked evidence set. Explicit contradiction and stale evidence can move a subject to `NO_TRADE`. Incomplete relevant evidence remains `WATCH`. `CANDIDATE` requires a pre-existing, structurally complete safe candidate from the event decision path, resolved assets, fresh evidence, at least two primary/independent groups including a primary source, and no contradiction. It reuses that candidate; articles alone cannot manufacture one. Downward transitions are enabled.

The genuine event decision, evidence link, evaluation history, and current subject projection are persisted in the same serializable transaction. Per-subject advisory locking and uniqueness constraints prevent concurrent projection corruption. No external network call occurs within that transaction.

## Replay semantics

- Same event and subject ruleset: reuse the subject/link/evaluation without changing timestamps.
- New evidence: append one link and one evaluation, then atomically update the projection.
- New ruleset or changed evidence fingerprint: append a versioned evaluation; historical reasoning remains intact.
- Asset resolution: changes the evidence fingerprint and causes deterministic re-evaluation.
- Candidate re-evaluation: reuse the one subject-linked candidate.

## July 2026 proof

The persisted replay selected 202 genuine inbox events: 197 were eligible and five were excluded. It produced 189 subjects and eight justified associations, 197 links, 197 evaluations, 18 event-level `NO_TRADE` decisions, 179 event-level `WATCH` decisions, and zero candidates. An identical replay created nothing and reused all 197 decisions, subjects, links, and evaluations.

The genuine World Monitor subset contained 172 events and conservatively produced 172 subjects: 115 `NO_TRADE`, 57 `WATCH`, zero `CANDIDATE`, and no unjustified association. The wider multi-event review found only exact repeated reports sharing a subject and source group; no obvious false merge was found. A live collection cycle persisted 12 new items, Jax ingested all 12, and each remained an independent WATCH subject with unknown assets.

Controlled transition coverage uses explicitly synthetic fixtures in isolated test databases. No synthetic article was inserted into the operator database.

Before and after runtime proof, the existing operator database counts were unchanged: two approvals, two candidate approvals, one paper ticket, one execution instruction, and zero order intents, broker order identifiers, trades, or fills. The expected integration-created delta was zero for every trading-side table.

## Troubleshooting

If evidence does not advance, verify that the subject keys genuinely match, then inspect the bounded subject detail endpoint and transition history. Repeated wire copies should not raise the independent-source count. If a subject remains WATCH, read `missing_evidence`; do not weaken association or readiness thresholds to force a candidate.

If the World Monitor cursor advances but no link appears, inspect pull-worker logs and transaction rollback diagnostics. A failed link/evaluation/projection write must leave no partial subject change.
