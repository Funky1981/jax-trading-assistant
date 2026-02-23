module jax-trading-assistant/libs/utcp

go 1.24.0

require (
	github.com/DATA-DOG/go-sqlmock v1.5.2
	github.com/jackc/pgx/v5 v5.8.0
	jax-trading-assistant/libs/backtest v0.0.0
	jax-trading-assistant/libs/contracts v0.0.0
	jax-trading-assistant/libs/observability v0.0.0
	jax-trading-assistant/libs/strategies v0.0.0
)

replace jax-trading-assistant/libs/backtest => ../backtest

replace jax-trading-assistant/libs/contracts => ../contracts

replace jax-trading-assistant/libs/observability => ../observability

replace jax-trading-assistant/libs/strategies => ../strategies
