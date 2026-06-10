# Phase 7 - Production Hardening

Goal:
Make the harness production-grade.

Add:
- timeouts per tool
- rate limits per session
- max answer size
- feature flags
- model fallback behavior
- per-mode policy (dev/research/paper/live)
- redaction for sensitive values
- structured metrics
- shadow mode testing before full rollout

Operational rules:
- live mode should be strictest
- fallback text should be boring, not clever
- model failure should degrade gracefully
- no hidden side effects

Acceptance:
- Harness survives model outages
- Harness fails closed on policy uncertainty
- Live mode remains advisory-only
