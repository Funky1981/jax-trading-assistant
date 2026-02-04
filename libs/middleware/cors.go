package middleware

import (
	"net/http"
	"os"
	"strings"
)

// CORSConfig holds CORS configuration
type CORSConfig struct {
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	AllowCredentials bool
	MaxAge           int // Preflight cache duration in seconds
}

// DefaultCORSConfig returns a default CORS configuration for development
func DefaultCORSConfig() CORSConfig {
	return CORSConfig{
		AllowedOrigins: []string{
			"http://localhost:3000",
			"http://localhost:5173",
			"http://127.0.0.1:3000",
			"http://127.0.0.1:5173",
		},
		AllowedMethods: []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodDelete,
			http.MethodOptions,
			http.MethodPatch,
		},
		AllowedHeaders: []string{
			"Content-Type",
			"Authorization",
			"X-Requested-With",
			"Accept",
			"Origin",
		},
		AllowCredentials: true,
		MaxAge:           3600, // 1 hour
	}
}

// CORSConfigFromEnv creates a CORS configuration from environment variables
func CORSConfigFromEnv() CORSConfig {
	config := DefaultCORSConfig()

	// Parse allowed origins from comma-separated list
	if origins := os.Getenv("CORS_ALLOWED_ORIGINS"); origins != "" {
		config.AllowedOrigins = parseCommaSeparated(origins)
	}

	// Parse allowed methods
	if methods := os.Getenv("CORS_ALLOWED_METHODS"); methods != "" {
		config.AllowedMethods = parseCommaSeparated(methods)
	}

	// Parse allowed headers
	if headers := os.Getenv("CORS_ALLOWED_HEADERS"); headers != "" {
		config.AllowedHeaders = parseCommaSeparated(headers)
	}

	// Parse allow credentials
	if creds := os.Getenv("CORS_ALLOW_CREDENTIALS"); creds != "" {
		config.AllowCredentials = strings.ToLower(creds) == "true"
	}

	return config
}

// CORS returns a middleware that handles CORS headers
func CORS(config CORSConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			// Check if origin is allowed
			if origin != "" && isOriginAllowed(origin, config.AllowedOrigins) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
			}

			// Set other CORS headers
			if config.AllowCredentials {
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}

			// Handle preflight requests
			if r.Method == http.MethodOptions {
				w.Header().Set("Access-Control-Allow-Methods", strings.Join(config.AllowedMethods, ", "))
				w.Header().Set("Access-Control-Allow-Headers", strings.Join(config.AllowedHeaders, ", "))
				w.Header().Set("Access-Control-Max-Age", string(rune(config.MaxAge)))
				w.WriteHeader(http.StatusNoContent)
				return
			}

			// Expose headers for actual requests
			w.Header().Set("Access-Control-Expose-Headers", "Content-Length, Content-Type")

			next.ServeHTTP(w, r)
		})
	}
}

// CORSFunc returns a middleware function that handles CORS headers
func CORSFunc(config CORSConfig) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			// Check if origin is allowed
			if origin != "" && isOriginAllowed(origin, config.AllowedOrigins) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
			}

			// Set other CORS headers
			if config.AllowCredentials {
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}

			// Handle preflight requests
			if r.Method == http.MethodOptions {
				w.Header().Set("Access-Control-Allow-Methods", strings.Join(config.AllowedMethods, ", "))
				w.Header().Set("Access-Control-Allow-Headers", strings.Join(config.AllowedHeaders, ", "))
				w.Header().Set("Access-Control-Max-Age", string(rune(config.MaxAge)))
				w.WriteHeader(http.StatusNoContent)
				return
			}

			// Expose headers for actual requests
			w.Header().Set("Access-Control-Expose-Headers", "Content-Length, Content-Type")

			next(w, r)
		}
	}
}

// isOriginAllowed checks if the given origin is in the allowed list
func isOriginAllowed(origin string, allowedOrigins []string) bool {
	for _, allowed := range allowedOrigins {
		if allowed == "*" {
			return true
		}
		if allowed == origin {
			return true
		}
		// Support wildcard subdomains (e.g., "https://*.example.com")
		if strings.Contains(allowed, "*") {
			pattern := strings.ReplaceAll(allowed, "*", ".*")
			// Simple pattern matching - for production, consider using regexp
			if strings.HasPrefix(origin, strings.Split(pattern, "*")[0]) {
				return true
			}
		}
	}
	return false
}

// parseCommaSeparated parses a comma-separated string into a slice
func parseCommaSeparated(s string) []string {
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
