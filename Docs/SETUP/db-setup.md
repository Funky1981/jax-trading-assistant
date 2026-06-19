# Postgres Setup

## Start Database

From repo root:

```powershell
docker compose up -d postgres
docker compose ps postgres
docker compose logs -f postgres
```

Default host access:
- Host: `localhost`
- Port: `5433`
- DB: `jax`
- User: `jax`
- Password: `jax`

## Verify Connectivity

```powershell
psql "postgresql://jax:jax@localhost:5433/jax" -c "SELECT 1;"
```

## Migrations

The root compose stack mounts `db/postgres/migrations` for first-run initialization.

To inspect migration state:

```powershell
docker compose exec postgres psql -U jax -d jax -c "SELECT version, dirty FROM schema_migrations ORDER BY version;"
```

## App Configuration

Use:

```text
DATABASE_URL=postgresql://jax:jax@localhost:5433/jax
```

Set this for local `cmd/trader` / `cmd/research` runs or via compose env overrides.
