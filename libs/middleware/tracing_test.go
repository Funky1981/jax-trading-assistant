package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// sentinel handler that records the flow_id it sees in context.
func echoFlowIDHandler(t *testing.T, got *string) http.Handler {
	t.Helper()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*got = FlowIDFromRequest(r)
		w.WriteHeader(http.StatusOK)
	})
}

func TestFlowID_HeaderPresent_Propagated(t *testing.T) {
	const want = "flow_12345_abcdef"
	var got string

	handler := FlowID(echoFlowIDHandler(t, &got))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Flow-ID", want)
	rw := httptest.NewRecorder()

	handler.ServeHTTP(rw, req)

	if got != want {
		t.Errorf("context flow_id = %q; want %q", got, want)
	}
	if rw.Header().Get("X-Flow-ID") != want {
		t.Errorf("response X-Flow-ID = %q; want %q", rw.Header().Get("X-Flow-ID"), want)
	}
}

func TestFlowID_HeaderAbsent_NewIDGenerated(t *testing.T) {
	var got string

	handler := FlowID(echoFlowIDHandler(t, &got))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rw := httptest.NewRecorder()

	handler.ServeHTTP(rw, req)

	if got == "" {
		t.Error("expected a generated flow_id in context, got empty string")
	}
	if !strings.HasPrefix(got, "flow_") {
		t.Errorf("generated flow_id %q does not start with 'flow_'", got)
	}
	if rw.Header().Get("X-Flow-ID") != got {
		t.Errorf("response X-Flow-ID = %q; want %q", rw.Header().Get("X-Flow-ID"), got)
	}
}

func TestFlowID_AlwaysSetsResponseHeader(t *testing.T) {
	handler := FlowID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	rw := httptest.NewRecorder()

	handler.ServeHTTP(rw, req)

	if rw.Header().Get("X-Flow-ID") == "" {
		t.Error("expected X-Flow-ID response header to always be set")
	}
}

func TestFlowID_UniquePerRequest(t *testing.T) {
	ids := make([]string, 3)
	for i := range ids {
		var got string
		FlowID(echoFlowIDHandler(t, &got)).ServeHTTP(
			httptest.NewRecorder(),
			httptest.NewRequest(http.MethodGet, "/", nil),
		)
		ids[i] = got
	}

	seen := map[string]bool{}
	for _, id := range ids {
		if seen[id] {
			t.Errorf("duplicate flow_id generated: %q", id)
		}
		seen[id] = true
	}
}
