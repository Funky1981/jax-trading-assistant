package auth

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func TestNewJWTManager(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: Config{
				Secret:        []byte("test-secret-key"),
				Expiry:        time.Hour,
				RefreshExpiry: 24 * time.Hour,
				Issuer:        "test",
			},
			wantErr: false,
		},
		{
			name: "empty secret",
			config: Config{
				Secret: []byte{},
			},
			wantErr: true,
		},
		{
			name: "defaults applied",
			config: Config{
				Secret: []byte("test-secret"),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager, err := NewJWTManager(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewJWTManager() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && manager == nil {
				t.Error("NewJWTManager() returned nil manager")
			}
		})
	}
}

func TestGenerateAndValidateToken(t *testing.T) {
	manager, err := NewJWTManager(Config{
		Secret:        []byte("test-secret-key-for-testing"),
		Expiry:        time.Hour,
		RefreshExpiry: 24 * time.Hour,
		Issuer:        "test-issuer",
	})
	if err != nil {
		t.Fatalf("Failed to create JWT manager: %v", err)
	}

	userID := "user123"
	username := "testuser"
	role := "admin"

	// Generate token
	token, err := manager.GenerateToken(userID, username, role)
	if err != nil {
		t.Fatalf("GenerateToken() failed: %v", err)
	}
	if token == "" {
		t.Fatal("GenerateToken() returned empty token")
	}

	// Validate token
	claims, err := manager.ValidateToken(token)
	if err != nil {
		t.Fatalf("ValidateToken() failed: %v", err)
	}

	// Check claims
	if claims.UserID != userID {
		t.Errorf("UserID = %v, want %v", claims.UserID, userID)
	}
	if claims.Username != username {
		t.Errorf("Username = %v, want %v", claims.Username, username)
	}
	if claims.Role != role {
		t.Errorf("Role = %v, want %v", claims.Role, role)
	}
	if claims.Issuer != "test-issuer" {
		t.Errorf("Issuer = %v, want %v", claims.Issuer, "test-issuer")
	}
}

func TestValidateToken_Expired(t *testing.T) {
	manager, err := NewJWTManager(Config{
		Secret: []byte("test-secret-key"),
		Expiry: -time.Hour, // Already expired
		Issuer: "test",
	})
	if err != nil {
		t.Fatalf("Failed to create JWT manager: %v", err)
	}

	token, err := manager.GenerateToken("user123", "testuser", "user")
	if err != nil {
		t.Fatalf("GenerateToken() failed: %v", err)
	}

	// Should fail validation due to expiry
	_, err = manager.ValidateToken(token)
	if err == nil {
		t.Error("ValidateToken() should fail for expired token")
	}
}

func TestValidateToken_InvalidSignature(t *testing.T) {
	manager1, _ := NewJWTManager(Config{
		Secret: []byte("secret-1"),
		Expiry: time.Hour,
	})

	manager2, _ := NewJWTManager(Config{
		Secret: []byte("secret-2"),
		Expiry: time.Hour,
	})

	// Generate token with manager1
	token, err := manager1.GenerateToken("user123", "testuser", "user")
	if err != nil {
		t.Fatalf("GenerateToken() failed: %v", err)
	}

	// Try to validate with manager2 (different secret)
	_, err = manager2.ValidateToken(token)
	if err == nil {
		t.Error("ValidateToken() should fail for token signed with different secret")
	}
}

func TestExtractTokenFromRequest(t *testing.T) {
	tests := []struct {
		name    string
		header  string
		want    string
		wantErr bool
	}{
		{
			name:    "valid bearer token",
			header:  "Bearer abc123def456",
			want:    "abc123def456",
			wantErr: false,
		},
		{
			name:    "missing authorization header",
			header:  "",
			want:    "",
			wantErr: true,
		},
		{
			name:    "invalid format - no bearer",
			header:  "abc123def456",
			want:    "",
			wantErr: true,
		},
		{
			name:    "invalid format - wrong prefix",
			header:  "Basic abc123def456",
			want:    "",
			wantErr: true,
		},
		{
			name:    "case insensitive bearer",
			header:  "bearer abc123def456",
			want:    "abc123def456",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.header != "" {
				req.Header.Set("Authorization", tt.header)
			}

			got, err := ExtractTokenFromRequest(req)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExtractTokenFromRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ExtractTokenFromRequest() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMiddleware(t *testing.T) {
	manager, _ := NewJWTManager(Config{
		Secret: []byte("test-secret-key"),
		Expiry: time.Hour,
	})

	// Create a test handler
	var handlerCalled bool
	var capturedUserID string
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		claims, ok := ClaimsFromContext(r.Context())
		if ok {
			capturedUserID = claims.UserID
		}
		w.WriteHeader(http.StatusOK)
	})

	// Wrap with middleware
	protected := manager.Middleware(testHandler)

	t.Run("valid token", func(t *testing.T) {
		handlerCalled = false
		capturedUserID = ""

		token, _ := manager.GenerateToken("user123", "testuser", "user")
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()

		protected.ServeHTTP(w, req)

		if !handlerCalled {
			t.Error("Handler should have been called")
		}
		if w.Code != http.StatusOK {
			t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
		}
		if capturedUserID != "user123" {
			t.Errorf("UserID = %v, want %v", capturedUserID, "user123")
		}
	})

	t.Run("missing token", func(t *testing.T) {
		handlerCalled = false

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()

		protected.ServeHTTP(w, req)

		if handlerCalled {
			t.Error("Handler should not have been called")
		}
		if w.Code != http.StatusUnauthorized {
			t.Errorf("Status = %v, want %v", w.Code, http.StatusUnauthorized)
		}
	})

	t.Run("invalid token", func(t *testing.T) {
		handlerCalled = false

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer invalid-token")
		w := httptest.NewRecorder()

		protected.ServeHTTP(w, req)

		if handlerCalled {
			t.Error("Handler should not have been called")
		}
		if w.Code != http.StatusUnauthorized {
			t.Errorf("Status = %v, want %v", w.Code, http.StatusUnauthorized)
		}
	})
}

func TestGenerateRefreshToken(t *testing.T) {
	manager, _ := NewJWTManager(Config{
		Secret:        []byte("test-secret-key"),
		Expiry:        time.Hour,
		RefreshExpiry: 24 * time.Hour,
	})

	token, err := manager.GenerateRefreshToken("user123", "testuser", "user")
	if err != nil {
		t.Fatalf("GenerateRefreshToken() failed: %v", err)
	}

	claims, err := manager.ValidateToken(token)
	if err != nil {
		t.Fatalf("ValidateToken() failed: %v", err)
	}

	// Refresh token should have longer expiry
	expiryTime := claims.ExpiresAt.Time
	now := time.Now()
	duration := expiryTime.Sub(now)

	if duration < 23*time.Hour {
		t.Errorf("Refresh token expiry too short: %v", duration)
	}
}

func TestNewJWTManagerFromEnv(t *testing.T) {
	// Save original env vars
	origSecret := os.Getenv("JWT_SECRET")
	origExpiry := os.Getenv("JWT_EXPIRY")
	origRefresh := os.Getenv("JWT_REFRESH_EXPIRY")

	// Restore after test
	defer func() {
		os.Setenv("JWT_SECRET", origSecret)
		os.Setenv("JWT_EXPIRY", origExpiry)
		os.Setenv("JWT_REFRESH_EXPIRY", origRefresh)
	}()

	t.Run("valid env vars", func(t *testing.T) {
		os.Setenv("JWT_SECRET", "test-secret-from-env")
		os.Setenv("JWT_EXPIRY", "2h")
		os.Setenv("JWT_REFRESH_EXPIRY", "48h")

		manager, err := NewJWTManagerFromEnv()
		if err != nil {
			t.Fatalf("NewJWTManagerFromEnv() failed: %v", err)
		}
		if manager == nil {
			t.Fatal("NewJWTManagerFromEnv() returned nil")
		}
	})

	t.Run("missing secret", func(t *testing.T) {
		os.Unsetenv("JWT_SECRET")

		_, err := NewJWTManagerFromEnv()
		if err == nil {
			t.Error("NewJWTManagerFromEnv() should fail when JWT_SECRET is missing")
		}
	})
}

func TestGenerateSecureRandomString(t *testing.T) {
	length := 32
	str1, err := GenerateSecureRandomString(length)
	if err != nil {
		t.Fatalf("GenerateSecureRandomString() failed: %v", err)
	}

	str2, err := GenerateSecureRandomString(length)
	if err != nil {
		t.Fatalf("GenerateSecureRandomString() failed: %v", err)
	}

	if len(str1) != length {
		t.Errorf("String length = %v, want %v", len(str1), length)
	}

	if str1 == str2 {
		t.Error("Two random strings should not be identical")
	}
}
