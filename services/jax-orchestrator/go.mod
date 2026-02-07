module jax-trading-assistant/services/jax-orchestrator

go 1.24.0

replace jax-trading-assistant/libs/contracts => ../../libs/contracts

replace jax-trading-assistant/libs/observability => ../../libs/observability

replace jax-trading-assistant/libs/testing => ../../libs/testing

replace jax-trading-assistant/libs/agent0 => ../../libs/agent0

replace jax-trading-assistant/libs/strategies => ../../libs/strategies

replace jax-trading-assistant/libs/dexter => ../../libs/dexter

replace jax-trading-assistant/libs/database => ../../libs/database

replace jax-trading-assistant/libs/utcp => ../../libs/utcp

require (
	github.com/google/uuid v1.6.0
	jax-trading-assistant/libs/agent0 v0.0.0
	jax-trading-assistant/libs/contracts v0.0.0
	jax-trading-assistant/libs/database v0.0.0
	jax-trading-assistant/libs/dexter v0.0.0
	jax-trading-assistant/libs/observability v0.0.0
	jax-trading-assistant/libs/strategies v0.0.0
	jax-trading-assistant/libs/testing v0.0.0
	jax-trading-assistant/libs/utcp v0.0.0
)

require (
	github.com/golang-migrate/migrate/v4 v4.17.0 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/pgx/v5 v5.8.0 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/lib/pq v1.10.9 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	golang.org/x/sync v0.17.0 // indirect
	golang.org/x/text v0.29.0 // indirect
)
