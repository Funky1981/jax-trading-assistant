# Jax Golden Test Skill

Every decision feature must include golden cases.

Golden case contains:

- input event
- expected decision
- expected primary reason
- expected forbidden actions
- expected allowed actions
- expected review horizon

Golden tests must be deterministic.

The FTSE/oil/labour-data conflict case must always return:

```text
NO_TRADE
```
