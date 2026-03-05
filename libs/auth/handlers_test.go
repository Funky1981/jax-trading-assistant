package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type stubAuthenticator struct {
	user *AuthenticatedUser
	err  error
}

func (s stubAuthenticator) Authenticate(context.Context, string, string) (*AuthenticatedUser, error) {
	return s.user, s.err
}

func newTestJWTManager(t *testing.T) *JWTManager {
	t.Helper()
	manager, err := NewJWTManager(Config{
		Secret:        []byte("test-secret"),
		Expiry:        15 * time.Minute,
		RefreshExpiry: 24 * time.Hour,
		Issuer:        "test",
	})
	if err != nil {
		t.Fatalf("new jwt manager: %v", err)
	}
	return manager
}

func TestLoginHandlerSuccess(t *testing.T) {
	handler := LoginHandler(newTestJWTManager(t), stubAuthenticator{
		user: &AuthenticatedUser{ID: "u-1", Username: "alice", Role: "user"},
	})
	body := bytes.NewBufferString(`{"username":"alice","password":"secret"}`)
	req := httptest.NewRequest(http.MethodPost, "/auth/login", body)
	res := httptest.NewRecorder()

	handler.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
	var out LoginResponse
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if out.AccessToken == "" || out.RefreshToken == "" {
		t.Fatalf("expected access and refresh token, got %+v", out)
	}
}

func TestLoginHandlerInvalidCredentials(t *testing.T) {
	handler := LoginHandler(newTestJWTManager(t), stubAuthenticator{
		err: ErrInvalidCredentials,
	})
	body := bytes.NewBufferString(`{"username":"alice","password":"bad"}`)
	req := httptest.NewRequest(http.MethodPost, "/auth/login", body)
	res := httptest.NewRecorder()

	handler.ServeHTTP(res, req)

	if res.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
}

func TestLoginHandlerAccountLocked(t *testing.T) {
	handler := LoginHandler(newTestJWTManager(t), stubAuthenticator{
		err: &AccountLockedError{Until: time.Now().Add(10 * time.Minute)},
	})
	body := bytes.NewBufferString(`{"username":"alice","password":"bad"}`)
	req := httptest.NewRequest(http.MethodPost, "/auth/login", body)
	res := httptest.NewRecorder()

	handler.ServeHTTP(res, req)

	if res.Code != http.StatusLocked {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
	if res.Header().Get("Retry-After") == "" {
		t.Fatal("expected Retry-After header")
	}
}

func TestLoginHandlerAuthUnavailable(t *testing.T) {
	handler := LoginHandler(newTestJWTManager(t), nil)
	body := bytes.NewBufferString(`{"username":"alice","password":"secret"}`)
	req := httptest.NewRequest(http.MethodPost, "/auth/login", body)
	res := httptest.NewRecorder()

	handler.ServeHTTP(res, req)

	if res.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
}
