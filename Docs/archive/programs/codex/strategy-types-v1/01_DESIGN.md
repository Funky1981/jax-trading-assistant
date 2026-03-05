# 01_DESIGN: Strategy Type Model (code) vs Strategy Instance (config)

## Strategy Type (code) responsibilities
- Declare:
  - `StrategyId`
  - `Name`
  - `Description`
  - `ParameterSchema` (typed, defaults, ranges)
  - `RequiredInputs` (candles timeframe, earnings/news dependency)
- Validate:
  - parameter values
  - session window sanity (entry before flatten)
- Execute:
  - Given market data window(s), generate:
    - 0..N trade **signals** for a single day/session
  - Must be deterministic for a given input data slice

## Strategy Instance (config) responsibilities
- Provide:
  - instanceId
  - strategyId
  - symbols/universe
  - session window + flatten time
  - risk limits (max trades/day, max daily loss, risk per trade)
  - concrete parameter values
- Stored in DB and exportable as JSON file.

## Minimal interface (Go)
Create in `libs/strategytypes/`:

- `type StrategyType interface`:
  - `Metadata() StrategyMetadata`
  - `Validate(params map[string]any) error`
  - `Generate(ctx, input StrategyInput) ([]Signal, error)`

Where:
- `StrategyInput` includes:
  - symbol
  - session date + timezone
  - candles by timeframe (at least 1m or 5m)
  - optional earnings/news events (v1 can stub, but schema must exist)

## Why not reuse existing `libs/strategies`?
Existing registry is indicator strategies (RSI/MACD/MA) and likely assumes simple periodic generation.
Same-day strategies need:
- session windows
- flatten-by-close logic
- event inputs (earnings/news)
- intraday timeframes

It is cleaner to add `libs/strategytypes` and later decide whether to merge.
