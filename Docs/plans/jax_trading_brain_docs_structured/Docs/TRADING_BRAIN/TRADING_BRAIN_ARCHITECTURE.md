# Trading Brain Architecture

## Target flow

```text
Market Event
  -> Event Intelligence Core
  -> Decision Core
  -> Opportunity Router
  -> Swing Brain v1
  -> Risk Veto
  -> Paper Approval
  -> Decision Review
```

## Default

```text
NO_TRADE
```

Every decision must include structured reasons, scores, allowed actions, forbidden actions, and review schedule.
