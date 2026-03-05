package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

var (
	// ErrInvalidCredentials is returned when username/password verification fails.
	ErrInvalidCredentials = errors.New("invalid credentials")
)

// AccountLockedError indicates an account is temporarily locked.
type AccountLockedError struct {
	Until time.Time
}

func (e *AccountLockedError) Error() string {
	return fmt.Sprintf("account locked until %s", e.Until.UTC().Format(time.RFC3339))
}

// AuthenticatedUser is a verified identity loaded from persistent storage.
type AuthenticatedUser struct {
	ID       string
	Username string
	Role     string
}

// Authenticator validates login credentials.
type Authenticator interface {
	Authenticate(ctx context.Context, username, password string) (*AuthenticatedUser, error)
}

// PostgresAuthenticator validates credentials against auth_users.
type PostgresAuthenticator struct {
	pool              *pgxpool.Pool
	maxFailedAttempts int
	lockoutDuration   time.Duration
}

// NewPostgresAuthenticator creates a credential authenticator backed by Postgres.
func NewPostgresAuthenticator(pool *pgxpool.Pool) *PostgresAuthenticator {
	return &PostgresAuthenticator{
		pool:              pool,
		maxFailedAttempts: envInt("AUTH_MAX_FAILED_ATTEMPTS", 5),
		lockoutDuration:   time.Duration(envInt("AUTH_LOCKOUT_MINUTES", 15)) * time.Minute,
	}
}

// Authenticate validates username/password and applies failed-attempt lockout policy.
func (a *PostgresAuthenticator) Authenticate(ctx context.Context, username, password string) (*AuthenticatedUser, error) {
	if a == nil || a.pool == nil {
		return nil, errors.New("authenticator not configured")
	}
	username = normalizeUsername(username)
	if username == "" || password == "" {
		return nil, ErrInvalidCredentials
	}

	var (
		userID         string
		storedUsername string
		passwordHash   string
		role           string
		failedAttempts int
		lockedUntil    sql.NullTime
	)
	err := a.pool.QueryRow(ctx, `
		SELECT id::text, username, password_hash, role, failed_attempts, locked_until
		FROM auth_users
		WHERE lower(username) = lower($1)
	`, username).Scan(&userID, &storedUsername, &passwordHash, &role, &failedAttempts, &lockedUntil)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf("load auth user: %w", err)
	}

	now := time.Now().UTC()
	if lockedUntil.Valid && now.Before(lockedUntil.Time.UTC()) {
		return nil, &AccountLockedError{Until: lockedUntil.Time.UTC()}
	}

	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password)); err != nil {
		failedAttempts++
		var newLockedUntil any
		if failedAttempts >= a.maxFailedAttempts {
			newLockedUntil = now.Add(a.lockoutDuration)
		}
		if _, updateErr := a.pool.Exec(ctx, `
			UPDATE auth_users
			SET failed_attempts = $2,
			    locked_until = $3,
			    updated_at = NOW()
			WHERE id = $1::uuid
		`, userID, failedAttempts, newLockedUntil); updateErr != nil {
			return nil, fmt.Errorf("record failed login: %w", updateErr)
		}
		if lockUntil, ok := newLockedUntil.(time.Time); ok {
			return nil, &AccountLockedError{Until: lockUntil.UTC()}
		}
		return nil, ErrInvalidCredentials
	}

	if _, err := a.pool.Exec(ctx, `
		UPDATE auth_users
		SET failed_attempts = 0,
		    locked_until = NULL,
		    last_login_at = NOW(),
		    updated_at = NOW()
		WHERE id = $1::uuid
	`, userID); err != nil {
		return nil, fmt.Errorf("record successful login: %w", err)
	}

	return &AuthenticatedUser{
		ID:       userID,
		Username: storedUsername,
		Role:     role,
	}, nil
}

// HashPassword creates a bcrypt hash suitable for auth_users.password_hash.
func HashPassword(password string) (string, error) {
	password = strings.TrimSpace(password)
	if password == "" {
		return "", errors.New("password is required")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}
	return string(hash), nil
}

// BootstrapAuthUserFromEnv seeds/updates one initial credential when configured.
func BootstrapAuthUserFromEnv(ctx context.Context, pool *pgxpool.Pool) error {
	username := strings.TrimSpace(os.Getenv("AUTH_BOOTSTRAP_USERNAME"))
	password := strings.TrimSpace(os.Getenv("AUTH_BOOTSTRAP_PASSWORD"))
	if username == "" && password == "" {
		return nil
	}
	if username == "" || password == "" {
		return errors.New("AUTH_BOOTSTRAP_USERNAME and AUTH_BOOTSTRAP_PASSWORD must both be set")
	}
	if pool == nil {
		return errors.New("auth bootstrap requires database pool")
	}
	role := strings.ToLower(strings.TrimSpace(os.Getenv("AUTH_BOOTSTRAP_ROLE")))
	if role == "" {
		role = "admin"
	}
	if role != "admin" && role != "user" {
		return fmt.Errorf("invalid AUTH_BOOTSTRAP_ROLE %q", role)
	}

	hash, err := HashPassword(password)
	if err != nil {
		return err
	}
	_, err = pool.Exec(ctx, `
		INSERT INTO auth_users (username, password_hash, role)
		VALUES ($1, $2, $3)
		ON CONFLICT (lower(username))
		DO UPDATE SET password_hash = EXCLUDED.password_hash,
		              role = EXCLUDED.role,
		              updated_at = NOW(),
		              failed_attempts = 0,
		              locked_until = NULL
	`, normalizeUsername(username), hash, role)
	if err != nil {
		return fmt.Errorf("bootstrap auth user: %w", err)
	}
	return nil
}

func normalizeUsername(username string) string {
	return strings.ToLower(strings.TrimSpace(username))
}

func envInt(key string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}
