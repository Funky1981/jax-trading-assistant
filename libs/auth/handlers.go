package auth

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"time"
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
func LoginHandler(jwtManager *JWTManager, authenticator Authenticator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Only POST method is allowed")
			return
		}
		if authenticator == nil {
			writeError(w, http.StatusServiceUnavailable, "auth_unavailable", "Authentication backend unavailable")
			return
		}

		var req LoginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", "Invalid request body")
			return
		}

		if req.Username == "" || req.Password == "" {
			writeError(w, http.StatusUnauthorized, "invalid_credentials", "Invalid username or password")
			return
		}

		user, err := authenticator.Authenticate(r.Context(), req.Username, req.Password)
		if err != nil {
			var lockedErr *AccountLockedError
			switch {
			case errors.As(err, &lockedErr):
				retryAfter := int(time.Until(lockedErr.Until).Seconds())
				if retryAfter < 1 {
					retryAfter = 1
				}
				w.Header().Set("Retry-After", strconv.Itoa(retryAfter))
				writeError(w, http.StatusLocked, "account_locked", "Account temporarily locked due to failed login attempts")
			case errors.Is(err, ErrInvalidCredentials):
				writeError(w, http.StatusUnauthorized, "invalid_credentials", "Invalid username or password")
			default:
				log.Printf("auth login failed for %q: %v", req.Username, err)
				writeError(w, http.StatusInternalServerError, "auth_failed", "Failed to authenticate user")
			}
			return
		}

		// Generate tokens
		accessToken, err := jwtManager.GenerateToken(user.ID, user.Username, user.Role)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "token_generation_failed", "Failed to generate access token")
			return
		}

		refreshToken, err := jwtManager.GenerateRefreshToken(user.ID, user.Username, user.Role)
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
		if err := json.NewEncoder(w).Encode(response); err != nil {
			http.Error(w, "failed to encode login response", http.StatusInternalServerError)
		}
		log.Printf("auth login success for %q", user.Username)
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
		if err := json.NewEncoder(w).Encode(response); err != nil {
			http.Error(w, "failed to encode refresh response", http.StatusInternalServerError)
		}
	}
}

// writeError writes a JSON error response
func writeError(w http.ResponseWriter, status int, errorCode, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(ErrorResponse{
		Error:   errorCode,
		Message: message,
	}); err != nil {
		http.Error(w, "failed to encode error response", http.StatusInternalServerError)
	}
}
