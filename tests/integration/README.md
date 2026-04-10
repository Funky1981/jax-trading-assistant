# Integration Tests

## Overview

Integration tests verify end-to-end functionality with the active trader/research runtime stack running in Docker Compose.

## Running Integration Tests

### Prerequisites

```

# Start the Docker Compose stack

docker compose up -d

# Wait for services to be healthy (~10 seconds)

docker compose ps

```

### Run Tests

```

# Run integration tests

go test -tags=integration ./tests/integration/... -v

# Skip integration tests (default behavior)

SKIP_INTEGRATION=1 go test ./tests/integration/...

```

### Services Under Test
- **jax-trader** (`localhost:8081`): Trading/frontend API
- **jax-research** (`localhost:8091`): Orchestration and memory tools
- **Postgres** (`localhost:5433`): Persistence

## Test Coverage

### TestSignalLifecycle
- Inserts a signal directly into Postgres
- Verifies list/detail approval flow through `jax-trader`
- Confirms final status in the database

## CI/CD Integration

```

# Example GitHub Actions workflow
- name: Start services
  run: docker compose up -d

- name: Wait for healthy
  run: sleep 10

- name: Run integration tests
  run: go test -tags=integration ./tests/integration/... -v

- name: Stop services
  run: docker compose down

```

## Troubleshooting

### Services not starting

```

# Check logs

docker compose logs jax-trader
docker compose logs jax-research

# Restart services

docker compose restart

```

### Tests timing out

```

# Increase timeout in test code

# Or give services more time to start

sleep 15

```

### Port conflicts

```

# Check what's using ports

netstat -an | findstr "8081 8091 5433"

# Stop conflicting services or change ports in docker-compose.yml

```

## Future Enhancements
- [ ] Runtime smoke coverage for memory and orchestration tools
- [ ] Database migration tests
- [ ] Failure scenario testing
