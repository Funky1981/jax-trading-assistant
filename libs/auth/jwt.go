package auth

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	// ErrInvalidToken is returned when token validation fails
	ErrInvalidToken = errors.New("invalid or expired token")
	// ErrMissingToken is returned when no token is provided
	ErrMissingToken = errors.New("missing authorization token")
	// ErrInvalidAuthHeader is returned when the Authorization header format is invalid
	ErrInvalidAuthHeader = errors.New("invalid authorization header format")
)

// Claims represents the JWT claims structure
type Claims struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

// Config holds JWT configuration
type Config struct {
	Secret        []byte
	Expiry        time.Duration
	RefreshExpiry time.Duration
	Issuer        string
}

// JWTManager handles JWT token operations
type JWTManager struct {
	config Config
}

// NewJWTManager creates a new JWT manager with the provided configuration
func NewJWTManager(config Config) (*JWTManager, error) {
	if len(config.Secret) == 0 {
		return nil, errors.New("JWT secret cannot be empty")
	}
	if config.Expiry == 0 {
		config.Expiry = 24 * time.Hour // Default 24 hours
	}
	if config.RefreshExpiry == 0 {
		config.RefreshExpiry = 7 * 24 * time.Hour // Default 7 days
	}
	if config.Issuer == "" {
		config.Issuer = "jax-trading-assistant"
	}

	return &JWTManager{config: config}, nil
}

// NewJWTManagerFromEnv creates a JWT manager from environment variables
func NewJWTManagerFromEnv() (*JWTManager, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return nil, errors.New("JWT_SECRET environment variable is required")
	}

	expiry, err := parseDuration(os.Getenv("JWT_EXPIRY"), 24*time.Hour)
	if err != nil {
		return nil, fmt.Errorf("invalid JWT_EXPIRY: %w", err)
	}

	refreshExpiry, err := parseDuration(os.Getenv("JWT_REFRESH_EXPIRY"), 7*24*time.Hour)
	if err != nil {
		return nil, fmt.Errorf("invalid JWT_REFRESH_EXPIRY: %w", err)
	}

	return NewJWTManager(Config{
		Secret:        []byte(secret),
		Expiry:        expiry,
		RefreshExpiry: refreshExpiry,
		Issuer:        "jax-trading-assistant",
	})
}

// GenerateToken creates a new JWT token for the given user
func (m *JWTManager) GenerateToken(userID, username, role string) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID:   userID,
		Username: username,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(m.config.Expiry)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    m.config.Issuer,
			Subject:   userID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.config.Secret)
}

// GenerateRefreshToken creates a refresh token with longer expiry
func (m *JWTManager) GenerateRefreshToken(userID, username, role string) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID:   userID,
		Username: username,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(m.config.RefreshExpiry)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    m.config.Issuer,
			Subject:   userID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.config.Secret)
}

// ValidateToken validates a JWT token and returns the claims
func (m *JWTManager) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return m.config.Secret, nil
	})

	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, ErrInvalidToken
}

// ExtractTokenFromRequest extracts JWT token from Authorization header
func ExtractTokenFromRequest(r *http.Request) (string, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return "", ErrMissingToken
	}

	// Expected format: "Bearer <token>"
	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return "", ErrInvalidAuthHeader
	}

	return parts[1], nil
}

// Middleware returns an HTTP middleware that validates JWT tokens
func (m *JWTManager) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, err := ExtractTokenFromRequest(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		claims, err := m.ValidateToken(token)
		if err != nil {
			http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
			return
		}

		// Store claims in request context for use by handlers
		ctx := r.Context()
		ctx = withClaims(ctx, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// MiddlewareFunc returns an HTTP middleware function that validates JWT tokens
func (m *JWTManager) MiddlewareFunc(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token, err := ExtractTokenFromRequest(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		claims, err := m.ValidateToken(token)
		if err != nil {
			http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
			return
		}

		// Store claims in request context for use by handlers
		ctx := r.Context()
		ctx = withClaims(ctx, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

// GenerateSecureRandomString generates a cryptographically secure random string
func GenerateSecureRandomString(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes)[:length], nil
}

// parseDuration parses a duration string or returns a default value
func parseDuration(s string, defaultDuration time.Duration) (time.Duration, error) {
	if s == "" {
		return defaultDuration, nil
	}
	return time.ParseDuration(s)
}
