# Codex Instructions for Jax

Before any Jax task:

1. Read `Docs/JAX_PRODUCT_CHARTER.md`.
2. Read `Docs/CAPABILITY_MATRIX.md`.
3. Read the relevant `Docs/PHASE_CONTRACTS/` file.
4. Read the relevant `Docs/PROJECT_MANAGEMENT/` process docs.
5. Read relevant `.codex/skills/`.
6. Identify which capability changes.
7. Add or update `tests/golden` cases where behaviour changes.
8. State explicit exclusions.
9. Preserve `NO_TRADE` as the default decision.
10. Do not implement live trading.
11. Do not implement day trading in the current roadmap.

Project-manager docs control delivery process.
Jax product docs control product direction.
If the project-manager docs conflict with Jax product truth, Jax product truth wins.
