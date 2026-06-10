# 05 — Multi-Analyst Review Flow

## Goal

Make Jax behave like a disciplined desk with separate expert roles.

The roles are:

```text
Fundamental Analyst
Technical Analyst
Risk Manager
Trade Reviewer
```

## Flow

```text
Research trigger
  ↓
Fundamental Analyst
  ↓
Technical Analyst
  ↓
Risk Manager
  ↓
Trade Reviewer
  ↓
Evidence Bundle
  ↓
Candidate or No Trade
```

## Role 1 — Fundamental Analyst

Responsible for:

```text
event meaning
actual vs expected
market expectation
ETF relevance
macro/sector context
other events/confounders
priced-in risk
```

Output:

```text
fundamental verdict
fundamental score
reasons
confounders
missing evidence
```

## Role 2 — Technical Analyst

Responsible for:

```text
trend
levels
event candle
confirmation candle
VWAP
volume
ATR
relative strength
entry/stop/target quality
```

Output:

```text
technical verdict
technical score
reasons
invalidation levels
chart evidence
```

## Role 3 — Risk Manager

Responsible for:

```text
position risk
reward:risk
max exposure
correlation
chase risk
drawdown guard
paper/live mode
approval requirement
```

Output:

```text
risk verdict
risk score
position size
hard blocks
```

## Role 4 — Trade Reviewer

Responsible for final consistency:

```text
Do all analysts agree?
Are there contradictions?
Is evidence missing?
Does the trade match the playbook?
Is this a good trade or just an interesting event?
```

Output:

```text
candidate_allowed
watch_only
candidate_rejected
insufficient_evidence
```

## Review contract

```json
{
  "symbol": "QQQ",
  "event_id": "...",
  "fundamental": {
    "verdict": "strong_bearish",
    "score": 82,
    "reasons": []
  },
  "technical": {
    "verdict": "confirmed_bearish",
    "score": 78,
    "reasons": []
  },
  "risk": {
    "verdict": "pass",
    "score": 74,
    "hard_blocks": []
  },
  "review": {
    "decision": "candidate_allowed",
    "candidate_score": 78.4,
    "approval_required": true
  }
}
```

## Human-readable output

Jax must produce:

```text
Fundamental Analyst:
...

Technical Analyst:
...

Risk Manager:
...

Final Review:
...
```

## Codex task

```text
Create multi-analyst review orchestration.

Use deterministic services first.
LLM summarisation may be used only after the deterministic results exist.
LLM must not override hard vetoes.
```

## Tests

```text
all roles pass = candidate_allowed
FA fail blocks candidate
TA fail blocks candidate
Risk fail blocks candidate
LLM summary cannot remove veto
missing analyst output = insufficient_evidence
```

## Acceptance criteria

```text
each role has independent output
final decision links to each role
hard vetoes cannot be overridden
evidence bundle includes analyst sections
```
