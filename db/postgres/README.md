# Database Setup and Operations

This directory contains the Postgres database configuration, schema migrations, and operational scripts for Jax Trading Assistant.

## Quick Start

### 1. Start Postgres

```powershell
cd db/postgres
docker-compose up -d
```

This starts Postgres 16 with:
- Database: `jaxdb`
- User: `jaxuser`
- Password: `jaxpass`
- Port: `5432`
- Persistent volume for data

### 2. Run Migrations

```powershell
# From repository root
.\scripts\migrate.ps1 up
```

This applies all pending migrations from `db/postgres/migrations/`.

### 3. Verify Schema

```powershell
# Connect to database
docker exec -it jax-postgres psql -U jaxuser -d jaxdb

# List tables
\dt

# Describe a table
\d events
\d trades
\d audit_events

# Exit
\q
```

## Migration Management

### Apply Migrations

```powershell
# Apply all pending migrations
.\scripts\migrate.ps1 up

# Apply next N migrations
.\scripts\migrate.ps1 up -Steps 2
```

### Rollback Migrations

```powershell
# Rollback last migration
.\scripts\migrate.ps1 down

# Rollback last N migrations
.\scripts\migrate.ps1 down -Steps 2
```

### Check Version

```powershell
# Show current migration version
.\scripts\migrate.ps1 version
```

### Create New Migration

```powershell
# Creates .up.sql and .down.sql files
.\scripts\migrate.ps1 create -Name "add_positions_table"
```

### Force Version (Use with caution!)

```powershell
# Force migration version (for fixing dirty state)
.\scripts\migrate.ps1 force -Version 2
```

## Schema Overview

### Tables

**events**
- Stores market events (earnings gaps, volatility spikes, etc.)
- Indexed by symbol, type, and time
- JSONB payload for flexible event data

**trades**
- Trade setups generated from events
- Includes entry, stop, targets, and risk calculations
- References events table via `event_id`
- Indexed by symbol, strategy, direction, and creation time

**audit_events**
- Compliance and observability audit trail
- Tracks all system decisions with correlation IDs
- Indexed by correlation, category, action, outcome, and timestamp

### Indexes

Performance-optimized indexes for common query patterns:
- Time-series queries (DESC indexes on timestamps)
- Symbol lookups (composite indexes on symbol + time)
- Correlation tracking (correlation_id index)
- Filtering by category/outcome (category + timestamp composite)

## Connection Configuration

### Environment Variable

```powershell
$env:DATABASE_URL = "postgres://jaxuser:jaxpass@localhost:5432/jaxdb?sslmode=disable"
```

### Connection Pooling

Use the `libs/database` package for production-ready connection management:

```go
import "jax-trading-assistant/libs/database"

config := database.DefaultConfig()
config.DSN = "postgres://jaxuser:jaxpass@localhost:5432/jaxdb?sslmode=disable"

db, err := database.ConnectWithMigrations(ctx, config, "file://db/postgres/migrations")
```

Default pool settings:
- Max open connections: 25
- Max idle connections: 5
- Connection max lifetime: 5 minutes
- Connection max idle time: 1 minute
- Retry attempts: 3 with exponential backoff

## Backup and Restore

### Backup Database

```powershell
# Backup to file
docker exec jax-postgres pg_dump -U jaxuser -d jaxdb -F c -f /tmp/jaxdb_backup.dump

# Copy backup out of container
docker cp jax-postgres:/tmp/jaxdb_backup.dump ./backups/jaxdb_$(Get-Date -Format 'yyyyMMdd_HHmmss').dump
```

### Restore Database

```powershell
# Copy backup into container
docker cp ./backups/jaxdb_backup.dump jax-postgres:/tmp/restore.dump

# Restore from file
docker exec jax-postgres pg_restore -U jaxuser -d jaxdb -c /tmp/restore.dump
```

## Troubleshooting

### Connection Refused

```powershell
# Check if Postgres is running
docker ps | Select-String jax-postgres

# Check logs
docker logs jax-postgres

# Restart container
docker-compose restart postgres
```

### Dirty Migration State

If migrations fail mid-way, the migration version may be marked "dirty":

```powershell
# Check version and dirty state
.\scripts\migrate.ps1 version

# Fix dirty state by forcing to last known good version
.\scripts\migrate.ps1 force -Version 1

# Then re-apply migrations
.\scripts\migrate.ps1 up
```

### Reset Database

```powershell
# Stop and remove container + volume
docker-compose down -v

# Start fresh
docker-compose up -d

# Re-run migrations
.\scripts\migrate.ps1 up
```

## Production Considerations

1. **Connection Pooling**: Use `libs/database` package for proper connection management
2. **Migrations**: Always test migrations in staging before production
3. **Backups**: Set up automated daily backups with retention policy
4. **Monitoring**: Monitor connection pool stats, query performance, and disk usage
5. **Security**: Use strong passwords, SSL/TLS, and restrict network access
6. **Indexes**: Review query patterns and add indexes as needed
7. **Retention**: Implement data retention policies for audit_events table
