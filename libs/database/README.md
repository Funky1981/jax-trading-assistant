# Database Package

Shared database connection management and migration utilities for Jax Trading Assistant services.

## Features

- **Connection Pooling**: Configurable connection pool with sensible defaults for production
- **Health Checks**: Built-in readiness and liveness probes
- **Retry Logic**: Automatic retry with exponential backoff for transient errors
- **Migrations**: Integrated golang-migrate support
- **Observability**: Connection pool metrics and query logging

## Usage

### Basic Connection

```go
import "jax-trading-assistant/libs/database"

config := database.DefaultConfig()
config.DSN = "postgres://user:pass@localhost:5432/jaxdb?sslmode=disable"

db, err := database.Connect(ctx, config)
if err != nil {
    log.Fatal(err)
}
defer db.Close()
```

### With Migrations

```go
db, err := database.ConnectWithMigrations(ctx, config, "file://db/postgres/migrations")
if err != nil {
    log.Fatal(err)
}
```

### Configuration

```go
config := &database.Config{
    DSN:                 "postgres://...",
    MaxOpenConns:        25,
    MaxIdleConns:        5,
    ConnMaxLifetime:     5 * time.Minute,
    ConnMaxIdleTime:     1 * time.Minute,
    HealthCheckInterval: 30 * time.Second,
    RetryAttempts:       3,
    RetryDelay:          1 * time.Second,
}
```

## Running Migrations

### Command Line

```powershell
# Install migrate CLI
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# Run migrations up
migrate -path db/postgres/migrations -database "postgres://localhost:5432/jaxdb?sslmode=disable" up

# Rollback one migration
migrate -path db/postgres/migrations -database "postgres://localhost:5432/jaxdb?sslmode=disable" down 1

# Check version
migrate -path db/postgres/migrations -database "postgres://localhost:5432/jaxdb?sslmode=disable" version
```

### Programmatically

```go
err := database.RunMigrations(ctx, db, "file://db/postgres/migrations")
if err != nil {
    log.Fatal(err)
}
```

## Local Development

```powershell
# Start Postgres via Docker Compose
cd db/postgres
docker-compose up -d

# Run migrations
cd ../..
migrate -path db/postgres/migrations -database "postgres://jaxuser:jaxpass@localhost:5432/jaxdb?sslmode=disable" up

# Verify schema
docker exec -it postgres psql -U jaxuser -d jaxdb -c "\dt"
```
