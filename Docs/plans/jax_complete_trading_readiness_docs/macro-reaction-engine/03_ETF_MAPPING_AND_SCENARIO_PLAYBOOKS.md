# 03 — ETF Mapping and Scenario Playbooks

## Goal

Convert macro events into explicit ETF scenarios.

Jax should not simply say "rates bad, short tech". It should use deterministic playbooks.

## ETF allowlist

Phase 1 allowlist:

```text
SPY
QQQ
DIA
IWM
XLK
XLF
XLE
SMH
SOXX
TLT
GLD
```

Blocked:

```text
single stocks
options
leveraged ETFs
inverse ETFs
volatility ETFs
crypto
forex
futures
```

## Scenario playbooks

### Scenario: strong jobs / hot inflation

```text
event_direction: hawkish_rates
primary ETFs: QQQ, SPY, TLT
secondary ETFs: IWM, XLK, SMH, SOXX
expected reaction:
  QQQ down
  SPY down
  TLT down
  IWM down
candidate bias:
  short/avoid-long QQQ
  short/avoid-long TLT
phase 1 execution:
  paper-only ETF candidate
```

### Scenario: weak jobs / cool inflation

```text
event_direction: dovish_rates
primary ETFs: QQQ, SPY, TLT
expected reaction:
  QQQ up
  SPY up
  TLT up
candidate bias:
  long QQQ
  long TLT
```

### Scenario: hawkish Fed

```text
event_direction: hawkish_rates
primary ETFs: QQQ, SPY, TLT
extra rule:
  avoid first candle during press conference
  require 15-30 minute confirmation
```

### Scenario: dovish Fed

```text
event_direction: dovish_rates
primary ETFs: QQQ, SPY, TLT
extra rule:
  require statement + yields + ETF confirmation
```

### Scenario: bank stress

```text
event_direction: financial_credit
primary ETFs: XLF, SPY, TLT, GLD
expected reaction:
  XLF down
  SPY down
  TLT up or mixed
  GLD up or mixed
```

### Scenario: oil shock

```text
event_direction: energy_oil
primary ETFs: XLE, SPY, IWM
expected reaction:
  XLE up if oil supply shock
  SPY/IWM may fall if inflationary
```

## Data model

### macro_scenario_results

```sql
CREATE TABLE macro_scenario_results (
    id UUID PRIMARY KEY,
    macro_event_id UUID NOT NULL REFERENCES macro_events(id),
    scenario_key TEXT NOT NULL,
    candidate_bias TEXT NOT NULL,
    primary_symbols TEXT[] NOT NULL,
    secondary_symbols TEXT[] NOT NULL,
    required_confirmations TEXT[] NOT NULL,
    result TEXT NOT NULL,
    reason TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(macro_event_id, scenario_key)
);
```

## Result values

```text
eligible_for_reaction_check
blocked_unknown_event
blocked_no_etf_mapping
blocked_conflicting_scenario
blocked_disallowed_instrument
```

## Codex task

```text
Add deterministic macro scenario playbooks.

Given a macro_event:
1. classify scenario_key
2. map to allowlisted ETFs
3. define expected reaction direction
4. define required confirmations
5. persist scenario result
6. return eligible/blocked result
```

## Tests

```text
NFP beat maps to hawkish_rates playbook
hot CPI maps to hawkish_rates playbook
cool CPI maps to dovish_rates playbook
Fed press conference requires delayed confirmation
unknown event blocks candidate creation
disallowed ETF mapping rejected
```

## Acceptance criteria

```text
Every macro event has a scenario result
No unknown scenario can create candidate
ETF mapping reasons stored
Only approved ETFs allowed
Scenario output feeds chart reaction engine
```
