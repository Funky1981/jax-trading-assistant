# Postgres Docker setup for this project

Quick steps to run Postgres locally (from repo root):

1. Start the DB with the bundled Compose file:

```bash
docker-compose -f db/postgres/docker-compose.yml up -d

```

2. Confirm the container is healthy:

```bash
docker ps --filter name=jax-postgres
docker-compose -f db/postgres/docker-compose.yml logs -f postgres

```

3. The `postgres` service initializes databases using files in `db/postgres/migrations` on first run. If you need to apply `schema.sql` manually (after the container is running):

```bash
docker exec -i jax-postgres psql -U jaxuser -d jaxdb < db/postgres/schema.sql

```

4. Configure the backend to connect to Postgres:

- Option A (recommended): create a local `.env` file in `services/jax-api` with `DATABASE_URL=postgresql://<user>:<pass>@postgres:5432/<db>`.
- Option B: edit `config/jax-core.json` and set `postgresDsn` to the DSN string.

Notes:
- Do NOT commit files containing real passwords. Use `.env` (gitignored) or a secrets manager.
- When running the backend in Docker Compose with the `db/postgres` stack, use hostname `postgres` in the DSN. When running the backend on your host, use `localhost`.
