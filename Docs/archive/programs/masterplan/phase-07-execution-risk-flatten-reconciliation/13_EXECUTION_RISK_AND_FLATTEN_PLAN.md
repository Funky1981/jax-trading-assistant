# Execution, Risk, and Flatten Plan

## Authoritative risk controls
- risk per trade
- max daily loss
- max open positions
- per-symbol caps
- no averaging down by default
- strategy-specific constraints

## Execution chain
Signal -> Approval -> Order Intent -> Broker Order -> Fill -> Position/P&L

## Safeguards
- idempotency / duplicate prevention
- restart-safe processing
- order intent before submit
- reconciliation jobs

## Flatten-by-close
- cancel open orders
- close positions
- verify positions=0 and orders=0
- persist proof artifact
- gate failure on residuals
