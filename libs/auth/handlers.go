package auth

import (
	"encoding/json"
	"net/http"
)

// LoginRequest represents a login request body
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginResponse represents a successful login response
type LoginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"` // seconds
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// LoginHandler creates an HTTP handler for the login endpoint
// For production, integrate with a proper user authentication system
func LoginHandler(jwtManager *JWTManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Only POST method is allowed")
			return
		}

		var req LoginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", "Invalid request body")
			return
		}

		// SECURITY NOTE: This is a simplified authentication for development
		// In production, implement proper user authentication with:
		// - Password hashing (bcrypt, argon2)
		// - Database lookup
		// - Account lockout policies
		// - Multi-factor authentication
		// - Audit logging

		// For now, use a simple check (change this in production!)
		if req.Username == "" || req.Password == "" {
			writeError(w, http.StatusUnauthorized, "invalid_credentials", "Invalid username or password")
			return
		}

		// TODO: Replace with actual user authentication
		// For development, accept any non-empty credentials
		userID := "user-" + req.Username
		role := "user"
		if req.Username == "admin" {
			role = "admin"
		}

		// Generate tokens
		accessToken, err := jwtManager.GenerateToken(userID, req.Username, role)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "token_generation_failed", "Failed to generate access token")
			return
		}

		refreshToken, err := jwtManager.GenerateRefreshToken(userID, req.Username, role)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "token_generation_failed", "Failed to generate refresh token")
			return
		}

		// Send response
		response := LoginResponse{
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
			TokenType:    "Bearer",
			ExpiresIn:    int(jwtManager.config.Expiry.Seconds()),
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}
}

// RefreshHandler creates an HTTP handler for token refresh endpoint
func RefreshHandler(jwtManager *JWTManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Only POST method is allowed")
			return
		}

		// Extract refresh token from request
		token, err := ExtractTokenFromRequest(r)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "missing_token", err.Error())
			return
		}

		// Validate refresh token
		claims, err := jwtManager.ValidateToken(token)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "invalid_token", "Invalid or expired refresh token")
			return
		}

		// Generate new access token
		accessToken, err := jwtManager.GenerateToken(claims.UserID, claims.Username, claims.Role)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "token_generation_failed", "Failed to generate access token")
			return
		}

		// Optionally generate new refresh token (refresh token rotation)
		refreshToken, err := jwtManager.GenerateRefreshToken(claims.UserID, claims.Username, claims.Role)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "token_generation_failed", "Failed to generate refresh token")
			return
		}

		response := LoginResponse{
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
			TokenType:    "Bearer",
			ExpiresIn:    int(jwtManager.config.Expiry.Seconds()),
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}
}

// writeError writes a JSON error response
func writeError(w http.ResponseWriter, status int, errorCode, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(ErrorResponse{
		Error:   errorCode,
		Message: message,
	})
}
