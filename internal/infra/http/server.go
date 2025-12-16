package httpapi

import (
	"net/http"
)

type Server struct {
	mux *http.ServeMux
}

func NewServer() *Server {
	return &Server{mux: http.NewServeMux()}
}

func (s *Server) Handler() http.Handler {
	return s.mux
}
