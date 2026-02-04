package httpapi

import (
	"jax-trading-assistant/libs/auth"
	"jax-trading-assistant/libs/middleware"
	"log"
	"net/http"
)

type Server struct {
	mux         *http.ServeMux
	jwtManager  *auth.JWTManager
	rateLimiter *middleware.RateLimiter
	corsConfig  middleware.CORSConfig
}

func NewServer() *Server {
	// Initialize JWT manager from environment
	jwtManager, err := auth.NewJWTManagerFromEnv()
	if err != nil {
		log.Printf("WARNING: JWT authentication disabled: %v", err)
		log.Printf("Set JWT_SECRET environment variable to enable authentication")
	}

	// Initialize rate limiter from environment
	rateLimiter := middleware.NewRateLimiterFromEnv()

	// Initialize CORS configuration from environment
	corsConfig := middleware.CORSConfigFromEnv()

	return &Server{
		mux:         http.NewServeMux(),
		jwtManager:  jwtManager,
		rateLimiter: rateLimiter,
		corsConfig:  corsConfig,
	}
}

// Handler returns the HTTP handler with all middleware applied
func (s *Server) Handler() http.Handler {
	handler := http.Handler(s.mux)

	// Apply middleware in reverse order (innermost to outermost)
	// Rate limiting
	handler = s.rateLimiter.Middleware(handler)

	// CORS
	handler = middleware.CORS(s.corsConfig)(handler)

	return handler
}

// RegisterAuth registers authentication endpoints
func (s *Server) RegisterAuth() {
	if s.jwtManager == nil {
		log.Println("WARNING: Authentication endpoints not registered (JWT not configured)")
		return
	}

	s.mux.HandleFunc("/auth/login", auth.LoginHandler(s.jwtManager))
	s.mux.HandleFunc("/auth/refresh", auth.RefreshHandler(s.jwtManager))
	log.Println("Registered authentication endpoints: /auth/login, /auth/refresh")
}

// protect wraps a handler with JWT authentication middleware
func (s *Server) protect(handler http.HandlerFunc) http.HandlerFunc {
	if s.jwtManager == nil {
		// If JWT is not configured, allow all requests (development mode)
		log.Println("WARNING: Running without authentication (development mode)")
		return handler
	}
	return s.jwtManager.MiddlewareFunc(handler)
}
