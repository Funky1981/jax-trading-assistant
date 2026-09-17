# Exploratory Paper Trade Lifecycle

The lifecycle is a hypothetical record of a thesis, not an execution workflow:

1. `THESIS_DRAFT` — assumptions and provenance are recorded.
2. `APPROVAL_PENDING` — a human review is required.
3. `PAPER_APPROVED` — a paper ticket may be recorded with explicit hypothetical labels.
4. `MONITORING` — the thesis is checked across the bounded five-session window.
5. `THESIS_REVIEW` — relevant evidence and thesis state are reassessed without rewriting history.
6. `CLOSED` — deterministic exit reason, outcome, ambiguity, and missing data are retained.

The existing isolated simulated paper venue may create paper orders and fills.
No state creates a real order, broker order, live order, broker fill, live
position, or live execution instruction.
