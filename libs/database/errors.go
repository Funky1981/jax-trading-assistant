package database

import "errors"

var (
	// ErrInvalidDSN is returned when the DSN is empty or invalid
	ErrInvalidDSN = errors.New("invalid or empty DSN")

	// ErrMigrationFailed is returned when migrations fail to apply
	ErrMigrationFailed = errors.New("migration failed")

	// ErrConnectionFailed is returned when connection attempts are exhausted
	ErrConnectionFailed = errors.New("database connection failed")
)
