// trading_modes_handlers.go — Part of cmd/trader (package main).
// Exposes the trading mode catalog at /api/v1/trading-modes.
package main

import (
	"net/http"
	"strings"

	"jax-trading-assistant/internal/modules/tradingmodes"
)

// tradingModesHandler serves the full trading mode catalog.
func tradingModesHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		jsonOK(w, tradingmodes.DefaultCatalog())
	}
}

// tradingModeDetailHandler serves a single trading mode by ID.
func tradingModeDetailHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		id := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/trading-modes/"), "/")
		mode, ok := tradingmodes.DefaultCatalog().Get(id)
		if !ok {
			http.NotFound(w, r)
			return
		}
		jsonOK(w, mode)
	}
}
