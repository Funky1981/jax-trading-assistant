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

func (s *Server) Handler() http.Handler {
	return s.mux
}
