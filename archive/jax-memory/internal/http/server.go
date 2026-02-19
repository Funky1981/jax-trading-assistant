package httpapi

import (
	"net/http"

	"jax-trading-assistant/libs/contracts"
)

type Server struct {
	mux   *http.ServeMux
	store contracts.MemoryStore
}

func NewServer(store contracts.MemoryStore) *Server {
	return &Server{
		mux:   http.NewServeMux(),
		store: store,
	}
}

// corsMiddleware adds CORS headers to all responses
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Server) Handler() http.Handler {
	return corsMiddleware(s.mux)
}
