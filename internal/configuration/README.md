# Configuration test scope

The adjacent test file protects the small set of canonical configuration and
Compose invariants that must not regress during commercial-readiness cleanup.
It is intentionally test-only; runtime configuration ownership remains in the
existing service loaders and root Compose file.
