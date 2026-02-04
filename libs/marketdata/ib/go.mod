module jax-trading-assistant/libs/marketdata/ib

go 1.24.0

require jax-trading-assistant/libs/resilience v0.0.0

require jax-trading-assistant/libs/marketdata v0.0.0

require (
	cloud.google.com/go v0.112.0 // indirect
	github.com/alpacahq/alpaca-trade-api-go/v3 v3.3.0 // indirect
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/gabriel-vasile/mimetype v1.4.3 // indirect
	github.com/go-playground/form/v4 v4.2.1 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/go-playground/validator/v10 v10.18.0 // indirect
	github.com/go-resty/resty/v2 v2.11.0 // indirect
	github.com/gofinance/ib v0.0.0-20190131202149-a7abd0c5d772 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/leodido/go-urn v1.4.0 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/polygon-io/client-go v1.16.4 // indirect
	github.com/redis/go-redis/v9 v9.4.0 // indirect
	github.com/shopspring/decimal v1.3.1 // indirect
	github.com/sony/gobreaker/v2 v2.0.0 // indirect
	golang.org/x/crypto v0.19.0 // indirect
	golang.org/x/net v0.21.0 // indirect
	golang.org/x/sys v0.17.0 // indirect
	golang.org/x/text v0.14.0 // indirect
)

replace jax-trading-assistant/libs/resilience => ../../resilience

replace jax-trading-assistant/libs/marketdata => ../
