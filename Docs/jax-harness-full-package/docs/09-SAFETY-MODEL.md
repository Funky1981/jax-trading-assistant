# Safety Model

Separate two systems mentally:

1. Research harness
- exploratory
- evidence-based
- read-only
- can summarize, compare, explain, rank, and warn

2. Trading control plane
- approvals
- execution instructions
- broker submission
- risk enforcement
- audit

These must stay separate.

Design rule:
The research harness can inform the human.
The trading control plane can act only through deterministic business flows.
