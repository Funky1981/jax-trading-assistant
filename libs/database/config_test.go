package database

import (
	"context"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.MaxOpenConns != 25 {
		t.Errorf("expected MaxOpenConns=25, got %d", config.MaxOpenConns)
	}
	if config.MaxIdleConns != 5 {
		t.Errorf("expected MaxIdleConns=5, got %d", config.MaxIdleConns)
	}
	if config.ConnMaxLifetime != 5*time.Minute {
		t.Errorf("expected ConnMaxLifetime=5m, got %v", config.ConnMaxLifetime)
	}
	if config.ConnMaxIdleTime != 1*time.Minute {
		t.Errorf("expected ConnMaxIdleTime=1m, got %v", config.ConnMaxIdleTime)
	}
	if config.RetryAttempts != 3 {
		t.Errorf("expected RetryAttempts=3, got %d", config.RetryAttempts)
	}
}

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: &Config{
				DSN:             "postgres://localhost:5432/test",
				MaxOpenConns:    10,
				MaxIdleConns:    2,
				ConnMaxLifetime: 5 * time.Minute,
				ConnMaxIdleTime: 1 * time.Minute,
				RetryAttempts:   3,
				RetryDelay:      1 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "empty DSN",
			config: &Config{
				DSN: "",
			},
			wantErr: true,
		},
		{
			name: "applies defaults for missing values",
			config: &Config{
				DSN:             "postgres://localhost:5432/test",
				MaxOpenConns:    0,
				MaxIdleConns:    0,
				ConnMaxLifetime: 0,
				ConnMaxIdleTime: 0,
				RetryAttempts:   -1,
				RetryDelay:      0,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				if tt.config.MaxOpenConns <= 0 {
					t.Error("expected MaxOpenConns to be set to default")
				}
				if tt.config.MaxIdleConns <= 0 {
					t.Error("expected MaxIdleConns to be set to default")
				}
			}
		})
	}
}

func TestConfigIdleConnsConstraint(t *testing.T) {
	config := &Config{
		DSN:          "postgres://localhost:5432/test",
		MaxOpenConns: 5,
		MaxIdleConns: 10, // More than MaxOpenConns
	}

	err := config.Validate()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if config.MaxIdleConns > config.MaxOpenConns {
		t.Errorf("expected MaxIdleConns (%d) <= MaxOpenConns (%d)", config.MaxIdleConns, config.MaxOpenConns)
	}
}

func TestConnectInvalidDSN(t *testing.T) {
	config := &Config{
		DSN:           "invalid-dsn",
		RetryAttempts: 0, // No retries for faster test
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := Connect(ctx, config)
	if err == nil {
		t.Error("expected error for invalid DSN, got nil")
	}
}

func TestConnectContextCancellation(t *testing.T) {
	config := &Config{
		DSN:           "postgres://nonexistent:5432/test",
		RetryAttempts: 5,
		RetryDelay:    100 * time.Millisecond,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := Connect(ctx, config)
	if err == nil {
		t.Error("expected error due to context cancellation, got nil")
	}
}
