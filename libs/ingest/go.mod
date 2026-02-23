module jax-trading-assistant/libs/ingest

go 1.24.0

require (
	jax-trading-assistant/libs/contracts v0.0.0
	jax-trading-assistant/libs/observability v0.0.0
	jax-trading-assistant/libs/utcp v0.0.0
)

require (
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/pgx/v5 v5.8.0 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	golang.org/x/sync v0.18.0 // indirect
	golang.org/x/text v0.31.0 // indirect
)

replace jax-trading-assistant/libs/contracts => ../contracts

replace jax-trading-assistant/libs/observability => ../observability

replace jax-trading-assistant/libs/utcp => ../utcp
