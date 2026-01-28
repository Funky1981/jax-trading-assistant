# Integration Tests

## Overview
Integration tests verify end-to-end functionality with real services running in Docker Compose.

## Running Integration Tests

### Prerequisites
```bash
# Start the Docker Compose stack
docker compose up -d

# Wait for services to be healthy (~10 seconds)
docker compose ps
```

### Run Tests
```bash
# Run integration tests
go test -tags=integration ./tests/integration/... -v

# Skip integration tests (default behavior)
SKIP_INTEGRATION=1 go test ./tests/integration/...
```

### Services Under Test
- **Hindsight** (`localhost:8888`): Memory backend
- **jax-memory** (`localhost:8090`): Memory facade API
- **jax-api** (`localhost:8081`): Trading API

## Test Coverage

### TestMemoryIntegration
- Verifies memory service is accessible
- Checks health endpoint responds

### TestMemoryRetainRecall
- Tests end-to-end memory storage and retrieval
- Validates bank/item structure

### TestOrchestrationPipeline
- Validates full 7-stage orchestration flow:
  1. Recall memories
  2. Strategy signals
  3. Dexter research
  4. Agent0 planning
  5. Tool execution
  6. Memory retention
  7. Result returned

### TestDockerComposeStack
- Checks all services are running and healthy
- Validates connectivity between services

## CI/CD Integration
```yaml
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
```bash
# Check logs
docker compose logs hindsight
docker compose logs jax-memory

# Restart services
docker compose restart
```

### Tests timing out
```bash
# Increase timeout in test code
# Or give services more time to start
sleep 15
```

### Port conflicts
```bash
# Check what's using ports
netstat -an | findstr "8888 8090 8081"

# Stop conflicting services or change ports in docker-compose.yml
```

## Future Enhancements
- [ ] Database migration tests
- [ ] Performance/load testing
- [ ] Failure scenario testing
- [ ] Multi-service transaction tests
- [ ] Kafka/event stream tests
