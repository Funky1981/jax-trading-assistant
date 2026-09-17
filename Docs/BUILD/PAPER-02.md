# PAPER-02 — Prospective Exploratory Paper Pilot

Status: **READY_FOR_EXTERNAL_REVIEW / NOT ACTIVE**.

PAPER-02 is a prospective exploratory pilot design and readiness package. The
control plane is implemented, but it is not active and no prospective
observation has been admitted. External activation is still required.

The canonical protocol is `Docs/PAPER_TRADING/PAPER-02-PROSPECTIVE-PILOT-PROTOCOL.md`.
Before execution it defines:

- prospective real event and evidence intake with source provenance;
- event, issuer, and instrument resolution;
- an explicit causal thesis with supporting quant/technical context;
- human approval before every entry;
- isolated simulated paper execution only;
- a bounded 1–5 trading-session horizon and scheduled review;
- deterministic exit and thesis-invalidation rules;
- preserved `NO_TRADE`, `WATCH`, rejected, unresolved, and ambiguous cases;
- observability, operational stop conditions, and learning/evidence capture;
- a strict exploratory/formal firewall;
- no profitability claim and no demonstrated-edge claim;
- explicit confirmation that no real, broker, live, or autonomous execution capability is introduced.

The initial sample target is at least 50 eligible prospective event
opportunities, with a maximum duration of 90 calendar days. All eligible
opportunities receive an identity before later decision or outcome and retain
non-trade classifications.

The state machine is `DRAFT → READY_FOR_EXTERNAL_REVIEW → ACTIVE`. This package
implements DRAFT and READY_FOR_EXTERNAL_REVIEW only. It does not activate
PAPER-02 or start a prospective pilot.

PAPER-02 IS EXPLORATORY AND CANNOT ESTABLISH TRADING EDGE.

Formal sampling, preregistration, frozen scientific-policy requirements, and
formal evidence acceptance belong to the later `FORMAL-01` /
`FORMAL_FORWARD_PAPER` stage. PAPER-02 must not be used to create formal
evidence or to relabel exploratory observations.
