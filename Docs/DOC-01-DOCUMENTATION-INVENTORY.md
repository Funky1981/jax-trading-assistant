# DOC-01 Documentation Inventory

Inventory performed on 2026-09-16 before the DOC-01 edits. The active tree was
enumerated with `rg --files Docs`, then the control files, domain roots,
ProjectOS files, Jax-Roadmap-v2 top-level files, validation/evidence roots, and
PAPER-01 files were inspected. Archive and imported plan contents were retained
as historical inventory rather than treated as active routing.

## Classification

| Path or folder | Classification | Role |
| --- | --- | --- |
| `AGENTS.md` | `CANONICAL_CURRENT` | Repository execution/safety workflow |
| `Docs/JAX_PRODUCT_CHARTER.md` | `CANONICAL_CURRENT` | Product truth |
| `Docs/ROADMAP.md` | `CANONICAL_CURRENT` | Roadmap and sequence |
| `Docs/STATUS.md` | `CANONICAL_CURRENT` | Operational status |
| `Docs/CAPABILITY_MATRIX.md` | `CANONICAL_CURRENT` | Capability maturity |
| `Docs/BUILD/` | `CANONICAL_CURRENT` | Current build routing |
| `Docs/ARCHITECTURE.md`, `Docs/ARCHITECTURE/` | `CANONICAL_CURRENT` | Accepted architecture |
| `Docs/TRADING_BRAIN/` | `ACTIVE_SUPPORTING` | Current trader/domain detail |
| `Docs/PAPER_TRADING/` | `ACTIVE_SUPPORTING` | Current paper domain detail |
| `Docs/RESEARCH/`, `Docs/MEMORY_AND_REVIEW/` | `ACTIVE_SUPPORTING` | Research and learning detail |
| `Docs/OPERATIONS/`, `Docs/SETUP/`, `Docs/TESTING/` | `ACTIVE_SUPPORTING` | Operational support |
| `Docs/evidence/` | `IMMUTABLE_EVIDENCE` | Accepted implementation/evidence records |
| `Docs/validation/` | `IMMUTABLE_EVIDENCE` | Scientific validation records and results |
| `Docs/commercial-readiness/` | `HISTORICAL_SUPPORTING` | Technical cleanup evidence |
| `Docs/Jax-Roadmap-v2/` | `HISTORICAL_SUPPORTING` | Detailed implementation pack; no current routing |
| `Docs/PHASE_CONTRACTS/` | `RETIRED_CONTROL_PLANE` | Earlier delivery contracts; no current authority |
| `Docs/PROJECT_MANAGEMENT/` | `RETIRED_CONTROL_PLANE` | ProjectOS process tool only |
| `Docs/plans/` | `ARCHIVE_IMPORT` | Imported and completed planning material |
| `Docs/archive/` | `ARCHIVE_IMPORT` | Historical material and preserved snapshots |
| `Docs/archive/strategies/swing-pre-paper01/` | `HISTORICAL_SUPPORTING` | Pre-PAPER-01 generic swing assumptions |

## DOC-01 additions

- `Docs/DOCUMENTATION-AUTHORITY.md` is the explicit hierarchy.
- `Docs/BUILD/CURRENT_PACKAGE.md` is the only detailed current-build routing
  file.
- `Docs/TRADING_BRAIN/JAX_TRADER_MODEL_V1.md` is the canonical trader model.
- `Docs/BUILD/PAPER-01-CAPABILITY-MAP.md` is the current capability map for
  PAPER-01.

## Conflicts found and disposition

| Path | Current claim | Why stale/conflicting | Final disposition |
| --- | --- | --- | --- |
| `Docs/PROJECT_MANAGEMENT/project/roadmap.md` | `Phase: TBC` | Generic ProjectOS template | Retained as process template; pointer added; no authority |
| `Docs/PROJECT_MANAGEMENT/project/current-focus.md` | Phase 15 adapter is current | Superseded by PAPER-01 | Retained as historical snapshot; pointer added |
| `Docs/Jax-Roadmap-v2/NEXT-WORK-PACKAGE.md` | VAL-01/VAL-02 is current | Superseded package routing | Compatibility pointer; pack is supporting history |
| `Docs/PHASE_CONTRACTS/` | Phase contract controls delivery | Earlier delivery system | Strong retired-control-plane index added |
| `Docs/STRATEGIES/SWING_TRADING/` | Generic 2-day-to-8-week swing | Not current PAPER-01 policy | Moved to historical archive |
| `Docs/trading/` | Lowercase PAPER-01 active root | Duplicated uppercase domain layout | Merged, then removed |
| `Docs/JAX_PRODUCT_CHARTER.md` | `PAPER_PROVEN` in decision ladder | Ambiguous edge/paper terminology | Replaced with explicit paper modes and edge conclusion |

Historical validation terms and old decisions under `Docs/validation/`,
`Docs/evidence/`, `Docs/plans/`, `Docs/Jax-Roadmap-v2/`, and `Docs/archive/`
remain intentionally preserved and are not current routing.
