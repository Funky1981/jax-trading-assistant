# Documentation Authority

## Precedence

1. `AGENTS.md` — repository execution and safety workflow.
2. `Docs/JAX_PRODUCT_CHARTER.md` — what Jax is and non-negotiable behaviour.
3. `Docs/ROADMAP.md` — where Jax is going and package sequencing.
4. `Docs/STATUS.md` — where Jax is now.
5. `Docs/CAPABILITY_MATRIX.md` — what capabilities exist and their maturity.
6. `Docs/BUILD/CURRENT_PACKAGE.md` — what Codex may build now.
7. `Docs/ARCHITECTURE.md` — accepted runtime/system architecture.
8. Relevant domain documentation.

Product Charter > Roadmap > Status > Capability Matrix > Current Package >
Domain Detail > Historical Material.

## Current routing

The active product package is PAPER-02-2026-01 exploratory paper observation
collection, with the pilot identity and policies frozen as recorded in the
current handover and `Docs/STATUS.md`. HARNESS-00 is an orthogonal,
documentation-only architecture package and does not change that pilot. No
lower-level or historical document can authorize formal forward paper, live
execution, or Phase 13.

The detailed harness sources are `Docs/HARNESS/ARCHITECTURE.md`,
`Docs/HARNESS/EVALUATION.md`, `Docs/HARNESS/ROADMAP.md`, and
`Docs/HARNESS/ENGINEERING-HARNESS-GAP-ANALYSIS.md`. They do not authorize
HARNESS-01 or any runtime implementation.

## Non-authorities

`Docs/PHASE_CONTRACTS/`, `Docs/PROJECT_MANAGEMENT/project/`,
`Docs/Jax-Roadmap-v2/NEXT-WORK-PACKAGE.md`, `Docs/plans/`, and `Docs/archive/`
are retained for history or process only. They must not direct current product
work.

## Conflict rule

When a lower-level or historical document conflicts with a higher-level current
document, the higher-level document wins. The conflict is recorded in
`Docs/DOC-01-DOCUMENTATION-INVENTORY.md` or
`Docs/DOC-01-CONSISTENCY-REPORT.md`.
