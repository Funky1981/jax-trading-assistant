# Service Contracts

This package provides domain models, service interfaces, and HTTP adapters for the JAX trading system. It enables the modular monolith architecture transition (ADR-0012) by defining clear contracts between services.

## Structure

```
contracts/
├── domain/              # Shared domain models
│   ├── signal.go       # Trading signals
│   ├── order.go        # Trade orders
│   ├── position.go     # Positions & portfolio
│   └── orchestration.go # AI orchestration runs
├── services/           # Service interface definitions
│   ├── signal_generator.go
│   ├── trade_executor.go
│   ├── market_data.go
│   └── orchestrator.go
└── adapters/           # HTTP client implementations
    ├── http_signal_generator.go
    ├── http_trade_executor.go
    ├── http_market_data.go
    └── http_orchestrator.go
```

## Purpose

**Phase 1 Goal**: Define service boundaries while preserving HTTP communication.

- **Domain Models**: Shared types used by all services (e.g., `Signal`, `Order`, `Position`)
- **Service Interfaces**: Go interfaces defining service contracts (e.g., `SignalGenerator`)
- **HTTP Adapters**: Implementations that call services via HTTP (preserves current behavior)

## Usage

### Using HTTP Adapters

```go
import (
    "jax-trading-assistant/libs/contracts/adapters"
    "jax-trading-assistant/libs/contracts/services"
)

// Create HTTP client for signal generator
var signalGen services.SignalGenerator = adapters.NewHTTPSignalGenerator("http://localhost:8096")

// Generate signals (makes HTTP call)
signals, err := signalGen.GenerateSignals(ctx, []string{"AAPL", "TSLA"})
```

### Domain Models

```go
import "jax-trading-assistant/libs/contracts/domain"

signal := domain.Signal{
    ID:         "sig-123",
    Symbol:     "AAPL",
    Type:       "buy",
    Confidence: 0.85,
    EntryPrice: 150.25,
}
```

## Migration Path

**Phase 1 (Current)**: HTTP adapters implement interfaces
- Services communicate via HTTP
- Interfaces enable future in-process calls
- Zero behavior change

**Phase 2 (Future)**: In-process adapters
- Replace HTTP calls with direct function calls
- Same interfaces, different implementations
- Modular monolith achieved

## Testing

```bash
# Run all contract tests
go test ./libs/contracts/...

# Run domain model tests
go test ./libs/contracts/domain/...

# Run adapter tests
go test ./libs/contracts/adapters/...
```

## Design Decisions

1. **JSON Serialization**: All domain models support JSON marshaling for HTTP/API use
2. **Pointer Fields**: Optional fields (e.g., `FilledAt *time.Time`) use pointers for null safety
3. **Context Propagation**: All service methods accept `context.Context` for cancellation
4. **Error Wrapping**: Adapters wrap errors with context using `fmt.Errorf(..., %w, err)`

## Phase 1 Completion Criteria

- [x] Domain models defined for all service types
- [x] Service interfaces declared
- [x] HTTP adapters implemented
- [x] Contract tests written
- [x] Golden tests pass (no behavior change)
- [ ] Existing services updated to use domain models (Task 1.5)

## Related Documentation

- [ADR-0012: Modular Monolith](../../Docs/ADR-0012-two-runtime-modular-monolith.md)
- [Implementation Plan](../../Docs/ADR-0012-IMPLEMENTATION-PLAN.md)
- [Phase 0 Complete](../../Docs/PHASE_0_COMPLETE.md)
