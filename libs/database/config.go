package database

import "time"

// Config holds database connection configuration
type Config struct {
	// DSN is the database connection string
	DSN string

	// MaxOpenConns is the maximum number of open connections to the database
	MaxOpenConns int

	// MaxIdleConns is the maximum number of connections in the idle connection pool
	MaxIdleConns int

	// ConnMaxLifetime is the maximum amount of time a connection may be reused
	ConnMaxLifetime time.Duration

	// ConnMaxIdleTime is the maximum amount of time a connection may be idle
	ConnMaxIdleTime time.Duration

	// HealthCheckInterval is how often to ping the database to check health
	HealthCheckInterval time.Duration

	// RetryAttempts is the number of times to retry connecting on failure
	RetryAttempts int

	// RetryDelay is the initial delay between retry attempts (uses exponential backoff)
	RetryDelay time.Duration
}

// DefaultConfig returns a Config with sensible production defaults
func DefaultConfig() *Config {
	return &Config{
		MaxOpenConns:        25,
		MaxIdleConns:        5,
		ConnMaxLifetime:     5 * time.Minute,
		ConnMaxIdleTime:     1 * time.Minute,
		HealthCheckInterval: 30 * time.Second,
		RetryAttempts:       3,
		RetryDelay:          1 * time.Second,
	}
}

// Validate checks that the configuration is valid
func (c *Config) Validate() error {
	if c.DSN == "" {
		return ErrInvalidDSN
	}
	if c.MaxOpenConns <= 0 {
		c.MaxOpenConns = 25
	}
	if c.MaxIdleConns <= 0 {
		c.MaxIdleConns = 5
	}
	if c.MaxIdleConns > c.MaxOpenConns {
		c.MaxIdleConns = c.MaxOpenConns
	}
	if c.ConnMaxLifetime <= 0 {
		c.ConnMaxLifetime = 5 * time.Minute
	}
	if c.ConnMaxIdleTime <= 0 {
		c.ConnMaxIdleTime = 1 * time.Minute
	}
	if c.RetryAttempts < 0 {
		c.RetryAttempts = 0
	}
	if c.RetryDelay <= 0 {
		c.RetryDelay = 1 * time.Second
	}
	return nil
}
