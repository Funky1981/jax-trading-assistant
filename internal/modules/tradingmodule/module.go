package tradingmodule

import (
	"strings"

	"jax-trading-assistant/internal/modules/instruments"
)

const (
	ModuleETF    = "etf"
	ModuleLegacy = "legacy"
)

// ResolveFromSymbol classifies a symbol into ETF or legacy trading modules.
// Unknown symbols default to legacy to keep pre-ETF workflows isolated.
func ResolveFromSymbol(catalog *instruments.Catalog, symbol string) string {
	if catalog != nil && catalog.IsKnownETF(symbol) {
		return ModuleETF
	}
	return ModuleLegacy
}

func IsETF(module string) bool {
	return strings.EqualFold(strings.TrimSpace(module), ModuleETF)
}
