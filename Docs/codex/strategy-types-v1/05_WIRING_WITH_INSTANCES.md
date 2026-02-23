# 05_WIRING: How this connects to Strategy Instances later

When UI/DB instances are implemented:
- Instance stores `strategyId` + `parameters`.
- Runner loads instance, finds StrategyType by `strategyId` from registry.
- Runner calls:
  - `type.Validate(instance.parameters)`
  - fetch candles/events required by `RequiredInputs`
  - `type.Generate(...)`
- Generated signals are stored with `instanceId` and can be approved/executed.

Result:
- You can create 20 instances of `opening_range_to_close_v1` with different symbols and parameters **without code changes**.
- To add a new strategy *type*, you implement one new file and register it.

This is the minimal “strategy catalog” architecture.
