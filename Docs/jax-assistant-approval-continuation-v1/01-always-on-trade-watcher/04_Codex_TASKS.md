# Codex Tasks

1. Add `candidate_trades` schema and migration
2. Add watcher service in `cmd/trader`
3. Read enabled strategy instances from DB on startup
4. Schedule scans by timeframe/session
5. Generate candidate trades for qualified setups
6. Persist candidate events and status changes
7. Add `/api/v1/candidates*` endpoints
8. Add SSE endpoint for live candidate updates
9. Add tests:
   - no duplicate candidate generation
   - blocked candidates never reach approval queue
   - disabled instances do not scan
10. Update frontend later to consume candidates and SSE

## Definition of done
- Jax keeps scanning with browser closed
- qualified setups appear as candidate trades
- no execution occurs without approval
