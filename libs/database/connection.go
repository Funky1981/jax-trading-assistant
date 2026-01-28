package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// DB wraps sql.DB with additional functionality
type DB struct {
	*sql.DB
	config *Config
}

// Connect establishes a connection to the database with retry logic and connection pooling
func Connect(ctx context.Context, config *Config) (*DB, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	var db *sql.DB
	var err error

	// Retry connection with exponential backoff
	delay := config.RetryDelay
	for attempt := 0; attempt <= config.RetryAttempts; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
				delay *= 2 // Exponential backoff
			}
		}

		db, err = sql.Open("pgx", config.DSN)
		if err != nil {
			if attempt == config.RetryAttempts {
				return nil, fmt.Errorf("failed to open database after %d attempts: %w", config.RetryAttempts+1, err)
			}
			continue
		}

		// Configure connection pool
		db.SetMaxOpenConns(config.MaxOpenConns)
		db.SetMaxIdleConns(config.MaxIdleConns)
		db.SetConnMaxLifetime(config.ConnMaxLifetime)
		db.SetConnMaxIdleTime(config.ConnMaxIdleTime)

		// Test the connection
		if err = db.PingContext(ctx); err != nil {
			db.Close()
			if attempt == config.RetryAttempts {
				return nil, fmt.Errorf("failed to ping database after %d attempts: %w", config.RetryAttempts+1, err)
			}
			continue
		}

		// Success
		return &DB{
			DB:     db,
			config: config,
		}, nil
	}

	return nil, fmt.Errorf("failed to connect to database: %w", err)
}

// ConnectWithMigrations connects to the database and runs migrations
func ConnectWithMigrations(ctx context.Context, config *Config, migrationsPath string) (*DB, error) {
	db, err := Connect(ctx, config)
	if err != nil {
		return nil, err
	}

	if err := RunMigrations(db.DB, migrationsPath); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return db, nil
}

// HealthCheck performs a health check on the database connection
func (db *DB) HealthCheck(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("database health check failed: %w", err)
	}

	return nil
}

// Stats returns database connection pool statistics
func (db *DB) Stats() sql.DBStats {
	return db.DB.Stats()
}

// Config returns the database configuration
func (db *DB) Config() *Config {
	return db.config
}
