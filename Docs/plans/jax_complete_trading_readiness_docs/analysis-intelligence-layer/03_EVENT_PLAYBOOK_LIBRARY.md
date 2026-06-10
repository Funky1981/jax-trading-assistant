# 03 — Event Playbook Library

## Goal

Give Jax deterministic expert playbooks for common market-moving events.

Each playbook defines:

```text
what to check
which ETFs matter
expected direction
technical confirmation needed
fundamental confirmation needed
confounders
walk-away rules
```

## Playbook contract

```json
{
  "playbook_key": "hot_cpi_rates_hawkish",
  "event_types": ["US_CPI_HEADLINE", "US_CPI_CORE"],
  "expected_direction": "hawkish_rates",
  "primary_etfs": ["QQQ", "SPY", "TLT"],
  "secondary_etfs": ["IWM", "XLK", "SMH", "SOXX"],
  "required_fundamental_checks": [],
  "required_technical_checks": [],
  "confounders_to_check": [],
  "walkaway_rules": []
}
```

## Playbook 1 — Hot CPI

Expected:

```text
inflation_hot
rates hawkish
QQQ bearish
TLT bearish
SPY bearish/mixed
```

Required checks:

```text
headline CPI actual > expected
core CPI actual > expected
2y/10y yields up if available
TLT down
QQQ/SPY down
```

Walk away if:

```text
QQQ rallies and holds VWAP
TLT rallies
actual was only marginally above expected
major dovish Fed headline appears
move already too extended
```

## Playbook 2 — Cool CPI

Expected:

```text
inflation_cool
rates dovish
QQQ bullish
TLT bullish
SPY bullish
```

Required checks:

```text
headline/core CPI below expected
yields down if available
TLT up
QQQ/SPY up
```

Walk away if:

```text
TLT fails to rally
QQQ fades below VWAP
Fed speaker offsets the data
market was already pricing a cool print
```

## Playbook 3 — Strong Jobs / NFP Beat

Expected:

```text
growth strong
rates hawkish
QQQ bearish if yields confirm
TLT bearish
IWM mixed
```

Required checks:

```text
payrolls actual materially above expected
unemployment not worse than expected
wage growth not weak
TLT down
QQQ down vs SPY
```

Walk away if:

```text
unemployment rises unexpectedly
wages weak
yields fall
QQQ reclaims event range
other data released same time contradicts
```

## Playbook 4 — Weak Jobs / NFP Miss

Expected can be mixed:

```text
dovish rates bullish for QQQ/TLT
growth fear bearish for SPY/IWM
```

Required checks:

```text
payrolls miss
unemployment/wages context
TLT direction
QQQ/SPY direction
credit/risk-off signs
```

Walk away if:

```text
market treats it as recession scare
SPY and IWM collapse
TLT does not confirm dovish move
```

## Playbook 5 — Hawkish Fed

Expected:

```text
higher-for-longer
QQQ bearish
TLT bearish
SPY bearish
```

Special rule:

```text
do not trust first move
wait for 15–30 minute confirmation
```

Walk away if:

```text
Powell reverses statement reaction
dot plot and press conference conflict
TLT/QQQ diverge
```

## Playbook 6 — Dovish Fed

Expected:

```text
rate cuts more likely
QQQ bullish
TLT bullish
SPY bullish
```

Special rule:

```text
wait for statement + press conference confirmation
```

## Playbook 7 — Mega-Cap AI/Semiconductor Event

Expected:

```text
SMH/SOXX most affected
QQQ secondary
SPY smaller impact
```

Checks:

```text
is the information new?
is it company-specific or sector-wide?
are other semis confirming?
are rates working against growth?
is valuation already stretched?
```

## Playbook 8 — Bank Stress

Expected:

```text
XLF bearish
SPY bearish
TLT/GLD potentially bullish
```

Checks:

```text
credit spreads if available
bank ETF reaction
regional bank stress
official rescue/backstop news
```

## Playbook 9 — Oil Shock

Expected:

```text
XLE bullish if supply shock
SPY/IWM bearish if inflation shock
TLT bearish if inflationary
```

Checks:

```text
oil price move
cause of shock
duration likely
inflation implication
geopolitical context
```

## Codex task

```text
Create a deterministic playbook library.

Each playbook should define:
- supported event types
- affected ETFs
- expected direction
- required technical checks
- required fundamental checks
- confounders
- walk-away rules
```

## Tests

```text
hot CPI selects hot_cpi_rates_hawkish
cool CPI selects cool_cpi_rates_dovish
NFP beat selects strong_jobs_hawkish_rates
Fed event applies delayed confirmation
unknown event creates no playbook
playbook only returns allowlisted ETFs
```

## Acceptance criteria

```text
all in-scope event types map to playbooks
unknown events are blocked
playbook output feeds TA and FA engines
walk-away rules are included in evidence bundle
```
