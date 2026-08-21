# Roadmap Decision Log

## RD-2026-08-21-01 - Phase 00 GO

- Date: 2026-08-21
- Phase: 00 - Current Issuer & Asset Resolution
- Status: Accepted
- Decision authority: explicit technical-lead decision supplied for close-out

### Decision

Phase 00 is **GO**. Jax has sufficiently demonstrated that the current architecture can reliably identify the affected issuer/instrument or safely remain unresolved:

`Event -> typed causal attribution -> deterministic policy -> DIRECT / PROXY / UNRESOLVED -> deterministic resolver`

The decision is supported by the unseen Luna Generalization gate pass, Luna r3 repeatability at 46/48 (95.83%), all frozen retention gates passing, zero incorrect deterministic ticker/rule resolutions, and zero safety/persistence violations. Terra t1 independently achieved 47/48 semantic correctness and 5/6 PROXY recall with all six retention gates passing, and is classified `MATERIALLY BETTER`.

Residual limitations are accepted: Luna is weaker on difficult macro/proxy attribution; Terra is one-shot only; and Terra case 042 selected `US_RATES_CATEGORY` instead of `FEDERAL_RESERVE_OFFICIAL`. They do not invalidate GO.

Evidence: `../evidence/PHASE-00-ISSUER-RESOLUTION-CLOSEOUT.md`.

## RD-2026-08-21-02 - Hosted-model policy after Phase 00

- Date: 2026-08-21
- Status: Accepted

Luna (`gpt-5.6-luna`) remains the default Jax development/runtime hosted model. Terra (`gpt-5.6-terra`) is a validated higher-capability option that may be reconsidered for high-value/ambiguous escalation or when economics justify it.

No runtime default switch, escalation architecture, further Terra experiment, Terra repeatability, Sol challenger, additional model comparison, or Phase 00 prompt/schema tuning is authorized. Model evaluation is closed.

## RD-2026-08-21-03 - Roadmap reconciliation and next package

- Date: 2026-08-21
- Status: Accepted

WP-00.01, WP-00.02, and the expanded WP-00.03 sequence are complete. WP-00.04 was the only formally unclosed Phase 00 package; implementation evidence reduced it to a gate/closure and compact evidence-preservation action, which is completed by this close-out. No Phase 00 item remains.

No **ROADMAP CHANGE** is required. The implementation evolved beyond the roadmap's terse WP-00.03 wording, but it proved rather than invalidated the stated Phase 00 outcome and preserves the sequencing into canonical contracts/provenance/audit.

The exact first incomplete authorized-by-roadmap package is `WP-01.01 - Inventory existing Jax domain contracts before adding new ones`. It is not authorized for implementation by this decision and has not started. See `NEXT-WORK-PACKAGE.md`.
