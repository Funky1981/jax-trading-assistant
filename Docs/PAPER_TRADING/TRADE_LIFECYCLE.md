# Exploratory Paper Trade Lifecycle

The lifecycle is a hypothetical record of a thesis, not an execution workflow:

1. `THESIS_DRAFT` — source-backed event, candidate, and provenance are recorded.
2. `APPROVAL_PENDING` — a human review is required.
3. `PAPER_APPROVED` — a validated binding may create an isolated simulated paper order/fill with explicit exploratory labels.
4. `MONITORING` — the thesis is checked across the bounded five-session window;
   session five begins the hard `TIME_LIMIT` boundary.
5. `THESIS_REVIEW` — relevant evidence and thesis state are reassessed without rewriting history.
6. `CLOSED` — deterministic exit reason, outcome, ambiguity, and missing data are retained.

The existing isolated simulated paper venue may create simulated paper orders,
fills, ledger mutations, and positions. These are not real execution.
No state creates a real order, broker order, live order, broker fill, live
position, or live execution instruction.

Runtime review uses the explicit configured market/session calendar. Entry must
occur during a known open session. Reviews are scheduled at sessions 1–5,
skipping weekends and holidays. At the opening of session five, `TIME_LIMIT`
is actionable unless a higher-precedence reason already applies. Unknown or
incomplete calendar coverage fails closed; a caller-provided session flag never
overrides the calendar.
