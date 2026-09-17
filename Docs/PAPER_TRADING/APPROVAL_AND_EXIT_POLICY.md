# Exploratory Approval and Exit Policy

Human approval is required before a simulated PAPER-01 order is created. Approval records the reviewer, time, thesis version, assumptions, provenance, risk boundary, invalidation condition, and explicit statement that the record is limited to the isolated simulated paper venue.

The runtime entry handoff is protected by `POST /api/v1/exploratory-paper/entry-queue`.
It accepts only a workflow already in `PAPER_INTENT_CREATED` with an explicit,
unexpired human approval. The runtime reloads the canonical evidence projection
before creating a simulated paper order.

Exit review is rule-based with one canonical precedence: `RISK_KILL`, then
`MANUAL_OPERATOR`, then `THESIS_INVALIDATED`, then `STOP`, then `TARGET`, then
`TIME_LIMIT`. A closed-market decision is deferred to the next known tradable
session. Missing or contradictory evidence is retained through thesis review,
uncertainty, or unresolved state and does not invent an exit reason. Simulated
paper exit orders/fills remain inside the isolated venue; no broker/live order
or fill can be created, and exploratory observations cannot become formal
evidence.

If an exit is recommended but no exit approval is available, the review is
durably labelled `EXIT_RECOMMENDED` and remains visible. The worker does not
approve its own exit.
