# ETF Scope and Instrument Policy Plan

## Objective

Define a narrow, enforceable ETF trading scope for phase 1 so the platform does not treat ETFs as generic stock symbols.

## Current-state signals

- The market ingester currently uses a generic symbol list and includes `SPY` alongside equities in `cmd/trader/market_ingester.go` and `config/jax-market.json`.
- Existing strategy typing already references `SPY` and `QQQ`, which shows symbol-level support exists but not ETF-specific policy enforcement.
- Current runbooks describe paper/live runtime controls, but they do not define an ETF-only trading policy.

## Required outcomes

- establish a first-class instrument classification model
- introduce a phase-1 ETF allowlist
- explicitly exclude leveraged, inverse, volatility, illiquid, or otherwise disallowed ETF classes
- document who can change ETF eligibility and what evidence is required

## Phase-1 allowlist candidate

Start with a deliberately small liquid universe:

- SPY
- QQQ
- DIA
- IWM
- XLK
- XLF
- XLE
- SMH
- SOXX
- TLT
- GLD

## Explicit exclusions

- leveraged ETFs
- inverse ETFs
- volatility ETFs and ETNs
- options on ETFs
- thinly traded thematic ETFs
- any symbol not explicitly approved

## Deliverables

### 1. Instrument catalog decision

Create a single source of truth for:

- symbol
- asset class
- instrument type
- tradable mode (`paper`, later `live`)
- eligibility state
- effective date and change owner

### 2. ETF eligibility policy

Document:

- the initial approved ETF list
- the exclusion rules
- the review process for adding or removing ETFs
- the evidence needed for any eligibility change

### 3. Config and runtime wiring plan

Identify the narrowest runtime/config locations that must enforce the policy, including:

- trader runtime startup/config loading
- strategy registration or symbol selection paths
- execution entry points
- operator-facing documentation

## Acceptance criteria

- no ETF can be traded unless it exists in the approved instrument catalog
- exclusions are rule-based, not only name-based
- the approved set is visible to operators and testable in automation
- policy changes are auditable

## Dependencies

- runtime and execution gate work from `02-execution-and-risk-controls.md`
- workflow enforcement from `03-workflow-and-operator-guardrails.md`
- validation evidence from `04-validation-and-rollout.md`

## Exit gate

Do not begin paper ETF rollout until the allowlist and exclusion policy are implemented, documented, and testable.
