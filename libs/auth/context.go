package auth

import (
	"context"
)

type contextKey string

const claimsKey contextKey = "jwt_claims"

// withClaims stores JWT claims in the context
func withClaims(ctx context.Context, claims *Claims) context.Context {
	return context.WithValue(ctx, claimsKey, claims)
}

// ClaimsFromContext retrieves JWT claims from the context
func ClaimsFromContext(ctx context.Context) (*Claims, bool) {
	claims, ok := ctx.Value(claimsKey).(*Claims)
	return claims, ok
}

// UserIDFromContext retrieves the user ID from the context
func UserIDFromContext(ctx context.Context) (string, bool) {
	claims, ok := ClaimsFromContext(ctx)
	if !ok {
		return "", false
	}
	return claims.UserID, true
}

// UsernameFromContext retrieves the username from the context
func UsernameFromContext(ctx context.Context) (string, bool) {
	claims, ok := ClaimsFromContext(ctx)
	if !ok {
		return "", false
	}
	return claims.Username, true
}

// RoleFromContext retrieves the user role from the context
func RoleFromContext(ctx context.Context) (string, bool) {
	claims, ok := ClaimsFromContext(ctx)
	if !ok {
		return "", false
	}
	return claims.Role, true
}
