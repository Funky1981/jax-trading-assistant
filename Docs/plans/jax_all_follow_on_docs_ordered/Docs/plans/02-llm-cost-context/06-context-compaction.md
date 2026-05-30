# 06 — Context Compaction

Bad:

```text
raw articles + candles + logs + full memory -> paid model
```

Good:

```text
raw data -> deterministic extraction -> event summary -> price windows -> priced-in score -> confounders -> compact evidence bundle -> model summary
```

Retrieve only:

```text
top 5 similar events
top 5 relevant memories
current evidence bundle
current guardrail state
```
