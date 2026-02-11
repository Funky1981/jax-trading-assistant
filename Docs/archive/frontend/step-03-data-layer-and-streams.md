# Step 03: Data Layer & Streams

## Objective

Implement a reliable and performant data access layer that supports real-time trading data with prioritized updates.

## Actions

1. **Define API clients**
   - REST/GraphQL clients with typed responses.
   - Central error handling and retry policy.

2. **Streaming adapters**
   - Connect to WebSocket or streaming feeds.
   - Normalize market data by symbol/timeframe.
   - Batch updates at a stable cadence to avoid UI thrash.

3. **Caching & invalidation**
   - Use a single query cache (e.g., React Query).
   - Define cache TTLs for low-frequency data.

4. **Request prioritization**
   - High-priority: orders, positions, top-of-book.
   - Low-priority: historical analytics and reports.

## Deliverables
- `frontend/src/data/http-client.ts` for typed API access.
- `frontend/src/data/stream-buffer.ts` for throttled stream batching.
- `frontend/src/data/types.ts` for shared stream payloads.

## Exit Criteria
- Streams can be simulated with synthetic data.
- UI updates are stable under high-frequency updates.
