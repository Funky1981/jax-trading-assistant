# Event Taxonomy

## Purpose

This document defines the market events Jax is allowed to reason about.

If an event cannot be classified, Jax must use:

```text
UNKNOWN
```

and default to:

```text
NO_TRADE
```

## Event types

| Event Type | Description | Common Assets | Common Confounders |
|---|---|---|---|
| MACRO_DATA | Inflation, labour data, GDP, retail sales | Index, FX, rates, sectors | Central bank timing, revisions |
| CENTRAL_BANK | Rate decision, speech, minutes | Index, FX, rates, banks, housebuilders | Market already priced it |
| COMMODITY_SHOCK | Oil, gas, metals, supply disruption | Producers, consumers, ETFs, FX | Geopolitical risk, inventories |
| EARNINGS | Company results | Single stock, sector peers | Guidance, one-off items |
| GUIDANCE | Forward outlook change | Single stock, sector peers | Valuation, market expectations |
| REGULATORY | Approval, fine, ban, investigation | Company, sector | Legal uncertainty |
| GEOPOLITICAL | War, sanctions, trade conflict | Commodities, FX, defence, indices | Rumour quality |
| SECTOR_ROTATION | Capital moving between sectors | Sector ETFs, leading stocks | Macro driver unclear |
| INDEX_COMPOSITION | Index move driven by heavyweight names | Index, heavy constituents | Not broad market move |
| COMPANY_SPECIFIC | Management, product, M&A, debt | Single stock | Financial impact unclear |
| MARKET_STRUCTURE | Liquidity, flows, technical breaks | Indices, ETFs, liquid assets | False breakouts |
| MACRO_COMMODITY_INDEX_MOVE | Index move explained by commodity pressure and macro context | Index, commodity producers, FX, rates | Composition-driven move, central bank timing |
| UNKNOWN | Unclear event | Unknown | Default reject |

## Driver categories

```text
rates
inflation
labour_data
oil
gas
earnings
guidance
liquidity
currency
geopolitical
regulatory
sentiment
technical_breakout
technical_breakdown
valuation
sector_rotation
index_composition
central_bank
```

## Conflict examples

| Event | Conflict | Expected Bias |
|---|---|---|
| FTSE down because oil falls, UK labour strong | Oil bearish for energy, labour data policy-positive/negative conflict | NO_TRADE |
| Company beats earnings but cuts guidance | Historic good, future bad | NO_TRADE or WATCH |
| Index falls but only one heavyweight sector is dragging | Not broad market weakness | NO_TRADE |
| Strong macro data but FX/rates disagree | Market interpretation unclear | NO_TRADE |
| Central bank decision tomorrow | Event risk unresolved | NO_TRADE or WATCH |
| Rumour-only takeover | Source weak, price may already move | NO_TRADE |

## Asset mapping examples

| Driver | Affected Assets |
|---|---|
| oil | BP, SHEL, energy ETFs, Brent/WTI proxies |
| UK labour data | GBP, gilts, FTSE, UK banks, housebuilders |
| rates | banks, housebuilders, REITs, bonds, FX |
| inflation | bonds, FX, rate-sensitive equities |
| earnings beat | company, peers, sector ETF |
| guidance cut | company, peers |
| geopolitical oil risk | oil, energy, airlines, defence, indices |

## Classification rule

When multiple event types apply, classify both:

```json
{
  "event_type": "MACRO_COMMODITY_INDEX_MOVE",
  "secondary_event_types": ["MACRO_DATA", "COMMODITY_SHOCK", "INDEX_COMPOSITION"]
}
```

## Unknown events

If Jax cannot classify the event with confidence:

```text
decision = NO_TRADE
reason = Event type unclear.
```
