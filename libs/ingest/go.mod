module jax-trading-assistant/libs/ingest

go 1.24.0

require (
	jax-trading-assistant/libs/contracts v0.0.0
	jax-trading-assistant/libs/observability v0.0.0
	jax-trading-assistant/libs/utcp v0.0.0
)

replace jax-trading-assistant/libs/contracts => ../contracts

replace jax-trading-assistant/libs/observability => ../observability

replace jax-trading-assistant/libs/utcp => ../utcp
