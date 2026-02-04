module jax-trading-assistant

go 1.24.0

replace jax-trading-assistant/libs/contracts => ./libs/contracts

replace jax-trading-assistant/libs/observability => ./libs/observability

replace jax-trading-assistant/libs/database => ./libs/database

replace jax-trading-assistant/libs/utcp => ./libs/utcp

replace jax-trading-assistant/libs/testing => ./libs/testing

replace jax-trading-assistant/libs/marketdata => ./libs/marketdata

replace jax-trading-assistant/libs/resilience => ./libs/resilience

replace jax-trading-assistant/libs/ingest => ./libs/ingest

require (
	github.com/DATA-DOG/go-sqlmock v1.5.2
	github.com/golang-jwt/jwt/v5 v5.2.1
	github.com/google/uuid v1.6.0
	github.com/jackc/pgx/v5 v5.8.0
	jax-trading-assistant/libs/contracts v0.0.0
	jax-trading-assistant/libs/database v0.0.0-00010101000000-000000000000
	jax-trading-assistant/libs/ingest v0.0.0-00010101000000-000000000000
	jax-trading-assistant/libs/marketdata v0.0.0-00010101000000-000000000000
	jax-trading-assistant/libs/observability v0.0.0
	jax-trading-assistant/libs/testing v0.0.0-00010101000000-000000000000
)

require (
	cloud.google.com/go v0.121.6 // indirect
	github.com/alpacahq/alpaca-trade-api-go/v3 v3.3.0 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/gabriel-vasile/mimetype v1.4.3 // indirect
	github.com/go-playground/form/v4 v4.2.1 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/go-playground/validator/v10 v10.18.0 // indirect
	github.com/go-resty/resty/v2 v2.11.0 // indirect
	github.com/gofinance/ib v0.0.0-20190131202149-a7abd0c5d772 // indirect
	github.com/golang-migrate/migrate/v4 v4.19.1 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/leodido/go-urn v1.4.0 // indirect
	github.com/lib/pq v1.10.9 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/polygon-io/client-go v1.16.4 // indirect
	github.com/redis/go-redis/v9 v9.4.0 // indirect
	github.com/shopspring/decimal v1.3.1 // indirect
	github.com/sony/gobreaker/v2 v2.0.0 // indirect
	golang.org/x/crypto v0.45.0 // indirect
	golang.org/x/net v0.47.0 // indirect
	golang.org/x/sync v0.18.0 // indirect
	golang.org/x/sys v0.38.0 // indirect
	golang.org/x/text v0.31.0 // indirect
	jax-trading-assistant/libs/resilience v0.0.0-00010101000000-000000000000 // indirect
)
