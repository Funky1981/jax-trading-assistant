# Frontend Architecture

## Principles
- **Responsiveness first:** all user interactions must be reflected immediately with optimistic UI updates and non‑blocking async flows.
- **Separation of concerns:** data fetching, domain state, and presentation layers remain independent and testable.
- **Composable UI:** build screens from a stable, documented component library.
- **Predictable state flow:** use unidirectional data flow and explicit event modeling.

## Layered Design

### 1) Data Layer
- **Responsibilities:** real‑time streams, REST/GraphQL polling, caching, request prioritization, retry policies.
- **Guidelines:**
  - Use a single data access layer with typed APIs.
  - Normalize market data by symbol/timeframe to prevent redundant updates.
  - Throttle streaming updates to render at a consistent UI cadence.

### 2) Domain State Layer
- **Responsibilities:** trading domain models (orders, positions, risk limits), derived views, and business rules.
- **Guidelines:**
  - Keep domain logic framework‑agnostic.
  - Model events explicitly (e.g., `OrderPlaced`, `PriceUpdated`).
  - Encapsulate calculations (PnL, margin, Greeks) in testable utilities.

### 3) Presentation Layer
- **Responsibilities:** visual components, layout composition, keyboard navigation, accessibility.
- **Guidelines:**
  - Presentational components only receive props; they do not fetch data.
  - Use memoization and virtualization for heavy lists (blotter, orders, logs).
  - Keep a strict boundary between “containers” and “views.”

## Performance Targets
- **Input latency:** < 50 ms for UI responses (type, click, drag).
- **UI render cadence:** 30–60 fps during streaming updates.
- **Background processing:** heavy calculations off the main render path.

## Suggested Structure (Documentation Only)

```text
frontend/
  src/
    app/               # App shell, routing, providers
    data/              # API clients, query caches, stream adapters
    domain/            # Trading models, business logic
    components/        # Reusable UI components
    pages/             # Screen assemblies
    features/          # Cross‑cutting feature modules
    styles/            # Design tokens, themes, global styles
    tests/             # Shared test utilities
```
