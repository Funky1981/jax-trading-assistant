// tracing.go — FlowID propagation middleware for all HTTP handlers.
// A flow_id traces the full lifecycle of a trade decision:
//   quote ingested → signal generated → orchestration → approval → execution
//
// Usage:
//   handler = middleware.FlowID(existingHandler)
//
// The middleware reads X-Flow-ID from the request header. If absent it
// generates a new one via observability.NewFlowID(). The id is injected into
// the request context via observability.WithFlowID so every log statement in
// the call chain automatically includes it.
package middleware

import (
	"net/http"

	"jax-trading-assistant/libs/observability"
)

const flowIDHeader = "X-Flow-ID"

// FlowID is an HTTP middleware that propagates a trade-decision flow identifier.
// It reads X-Flow-ID from the incoming request, generates one if absent,
// injects it into the request context, and echoes it back in the response.
func FlowID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		flowID := r.Header.Get(flowIDHeader)
		if flowID == "" {
			flowID = observability.NewFlowID()
		}

		ctx := observability.WithFlowID(r.Context(), flowID)
		w.Header().Set(flowIDHeader, flowID)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// FlowIDFromRequest retrieves the flow_id from the request context.
// Returns empty string if not set.
func FlowIDFromRequest(r *http.Request) string {
	return observability.FlowIDFromContext(r.Context())
}
