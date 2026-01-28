package marketdata

import "errors"

var (
	// ErrNoProviderAvailable is returned when all providers fail
	ErrNoProviderAvailable = errors.New("no market data provider available")

	// ErrInvalidSymbol is returned when a symbol is invalid
	ErrInvalidSymbol = errors.New("invalid symbol")

	// ErrRateLimited is returned when a provider rate limit is hit
	ErrRateLimited = errors.New("rate limit exceeded")

	// ErrProviderError is returned when a provider returns an error
	ErrProviderError = errors.New("provider error")

	// ErrCacheError is returned when cache operations fail
	ErrCacheError = errors.New("cache error")

	// ErrInvalidTimeframe is returned when an unsupported timeframe is requested
	ErrInvalidTimeframe = errors.New("invalid timeframe")

	// ErrNoData is returned when no data is available for the request
	ErrNoData = errors.New("no data available")
)
