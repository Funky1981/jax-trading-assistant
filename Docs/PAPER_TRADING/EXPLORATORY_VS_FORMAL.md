# Exploratory vs Formal Paper Evidence

## Exploratory paper

Exploratory paper work is bounded observation of an implemented contract. It may expose workflow defects, ambiguity, missing provenance, and operational questions. It is not a performance claim and must remain labelled exploratory.

## Formal paper evidence

Formal evidence requires a separately approved package with frozen rules, predeclared sampling, complete provenance, explicit missing-data treatment, and independent review. Formal evidence has not started in the current roadmap state.

## Firewall

Exploratory observations must not be used to claim a demonstrated edge, promote a strategy, or satisfy a formal evidence gate. The repository must preserve the distinction in filenames, status labels, reports, and review decisions.

Persisted exploratory lifecycle rows are immutably labelled `EXPLORATORY_PAPER`
and `formal_evidence_eligible=false`. The read model exposes `NOT FORMAL
EVIDENCE`. There is no supported relabelling or promotion path; future formal
sampling and policy decisions remain a separate, not-started stage.
