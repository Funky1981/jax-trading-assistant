module jax-trading-assistant/services/jax-orchestrator

go 1.25.4

replace jax-trading-assistant/libs/contracts => ../../libs/contracts

replace jax-trading-assistant/libs/observability => ../../libs/observability

replace jax-trading-assistant/libs/testing => ../../libs/testing

replace jax-trading-assistant/libs/agent0 => ../../libs/agent0

replace jax-trading-assistant/libs/strategies => ../../libs/strategies

replace jax-trading-assistant/libs/dexter => ../../libs/dexter

require (
	jax-trading-assistant/libs/agent0 v0.0.0
	jax-trading-assistant/libs/contracts v0.0.0
	jax-trading-assistant/libs/dexter v0.0.0
	jax-trading-assistant/libs/observability v0.0.0
	jax-trading-assistant/libs/strategies v0.0.0
	jax-trading-assistant/libs/testing v0.0.0
)
