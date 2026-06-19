# Jax Decision Core Skill

Decision Core owns:

- Event
- Decision
- EvidenceBundle
- Scores
- Verdict

NO_TRADE is the default and must be first-class.

Allowed decisions:

- NO_TRADE
- WATCH
- SETUP_FORMING
- TRADE_CANDIDATE
- PAPER_APPROVAL_REQUIRED
- REJECTED_BY_RISK

Decision output must be structured.

Free-form prose is explanation only, never the source of truth.

A trade candidate must include:

- catalyst
- affected asset
- setup family
- edge reason
- risk reason
- invalidation condition
- required confirmation
- review horizon
