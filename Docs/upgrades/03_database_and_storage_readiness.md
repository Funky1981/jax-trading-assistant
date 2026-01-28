# Database and Storage Readiness

The Jax trading assistant persists trades and events via a UTCP storage adapter backed by Postgres. For production use, this layer must be hardened, schema migrations must be automated, and storage errors must not propagate silently.

## Why it matters

A reliable persistence layer is foundational. Without robust migrations and connection handling, you risk corrupting data or losing records during load. Production systems also need clear run books for bringing up, migrating, and backing up databases.

## Tasks

1. **Finalise the Postgres schema**
   - Review `db/postgres/schema.sql` and ensure that all necessary tables exist for trades (`utcp.trades`), audit events (`utcp.stored_events`) and any future features (e.g. portfolios, positions).
   - Add indexes on commonly queried fields such as `symbol`, `type`, and `created_at` to improve lookup performance.

2. **Introduce migrations**
   - Add a migration tool (e.g. [golang-migrate/migrate](https://github.com/golang-migrate/migrate) or [pressly/goose](https://github.com/pressly/goose)).
   - Create versioned migration scripts corresponding to `schema.sql` and future changes. Document how to run migrations locally and in CI.

3. **Connection management**
   - Ensure the Postgres connection pool (e.g. `database/sql` with `pgx`) is configured with sensible limits and timeouts for production workloads.
   - Implement retry logic on transient errors and surface failures to the caller with meaningful messages.

4. **Storage adapter hardening**
   - Review the UTCP storage adapter (`libs/utcp`) for error handling. All database operations should return context‑aware errors; panics should be avoided.
   - Validate input parameters before writing to the database. For example, enforce non‑empty `ID` and `Symbol` fields on trades and audit events.

5. **Testing**
   - Use `sqlmock` or a local Postgres instance to write unit tests that cover `SaveTrade`, `SaveAuditEvent`, and query functions.
   - Write integration tests that spin up a Postgres container via Docker Compose and ensure migrations run correctly.

6. **Operational readiness**
   - Provide documentation on how to run Postgres locally (`docker compose` commands) and how to apply migrations.
   - Plan backup and restore procedures and document them in the run book.