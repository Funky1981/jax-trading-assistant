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

## RD-2026-08-24-01 - WP-01.04 GO

- Date: 2026-08-24
- Phase: 01 - Canonical Contracts, Provenance & Audit
- Work package: WP-01.04 - Define replay/audit event model and compatibility strategy
- Status: Accepted
- Decision authority: explicit independent technical-lead decision supplied for close-out

### Decision

WP-01.04 is **GO**. The accepted implementation establishes immutable audit history, current-projection separation, deterministic replay manifests and verification, explicit compatibility classifications, V1/V2 fail-closed translation semantics, and the normative cross-runtime canonical-byte specification without changing the accepted WP-01.02/WP-01.03 architecture.

Evidence: `../evidence/WP-01.04-REPLAY-AUDIT-COMPATIBILITY.md`.

## RD-2026-08-24-02 - Phase 01 GO

- Date: 2026-08-24
- Phase: 01 - Canonical Contracts, Provenance & Audit
- Status: Accepted
- Decision authority: explicit independent technical-lead phase-gate decision supplied for close-out

### Decision

Phase 01 is **GO PHASE 01**. WP-01.01 established the source-backed contract inventory, WP-01.02 the canonical domain vocabulary, WP-01.03 immutable provenance/version/content identities, and WP-01.04 immutable audit/replay/compatibility semantics. All four packages were independently reviewed; no unresolved NO-GO or blocking CONDITIONAL GO remains.

The exit condition is demonstrated by the deterministic in-memory reconstruction chain in the WP-01.04 evidence: a representative Recommendation traces to immutable inputs, source/provider, versions, and timestamps without transient logs. The proof requires no database, provider, inference, or trading mutation.

The following accepted debts are non-blocking: broad production adoption of the canonical V2/audit model; append-only persistence enforcement; a non-Go canonical-byte conformance implementation; the inability to classify stochastic model re-inference as exact replay; unrelated repository-wide gofmt/lint debt; and the requirement that future adapters preserve immutable raw/provider evidence.

Phase 02 is eligible but not started. Its next package is `WP-02.01 - Provider registry/capability contract`, which awaits separate technical-lead package authorization. See `NEXT-WORK-PACKAGE.md`.

Evidence:

- `../evidence/WP-01.01-JAX-DOMAIN-CONTRACT-INVENTORY.md`
- `../evidence/WP-01.02-CANONICAL-DOMAIN-CONTRACTS.md`
- `../evidence/WP-01.03-CANONICAL-PROVENANCE-IDENTITIES.md`
- `../evidence/WP-01.04-REPLAY-AUDIT-COMPATIBILITY.md`

## RD-2026-08-24-03 - WP-02.01 GO

- Date: 2026-08-24
- Phase: 02 - Data Platform & Provider Architecture
- Work package: WP-02.01 - Provider registry/capability contract
- Status: Accepted
- Decision authority: explicit technical-lead decision supplied with WP-02.03 authorization

### Decision

WP-02.01 is **GO**. The accepted implementation establishes stable provider identity, capability-driven canonical output declarations, explicit provider-raw representation/schema boundaries, deterministic registry behavior, static support semantics, and bounded future runtime-state attachment without provider calls or runtime migration.

Evidence: `../evidence/WP-02.01-PROVIDER-REGISTRY-CAPABILITY-CONTRACT.md`.

## RD-2026-08-24-04 - WP-02.02 GO

- Date: 2026-08-24
- Phase: 02 - Data Platform & Provider Architecture
- Work package: WP-02.02 - Raw payload persistence/reference policy
- Status: Accepted
- Decision authority: explicit technical-lead decision supplied with WP-02.03 authorization

### Decision

WP-02.02 is **GO**, including the corrective identity-namespace commit `9b022fff3cd8d0f2cb08f2b104e3910a3ea4f573`. Accepted raw-payload acquisition identities use `rpa_`; the Phase 01 replay-manifest namespace remains `rpl_`. The implementation establishes exact-byte hashing, immutable acquisition/content separation, provider/capability/schema binding, verified storage-port reads, retention/redistribution metadata, and deterministic in-memory proof without production persistence or later Phase 02 behavior.

Evidence: `../evidence/WP-02.02-RAW-PAYLOAD-PERSISTENCE-REFERENCE-POLICY.md`.

## RD-2026-08-24-05 - WP-02.03 GO

- Date: 2026-08-24
- Phase: 02 - Data Platform & Provider Architecture
- Work package: WP-02.03 - Normalization and validation pipeline
- Status: Accepted
- Decision authority: explicit technical-lead decision supplied with WP-02.04 authorization

### Decision

WP-02.03 is **GO**. The accepted implementation establishes deterministic provider-owned normalization, exact capability/raw-schema/canonical-target routing, typed stage failures, canonical and raw-provenance validation, loss/omission metadata, strict normalizer identity/version binding, storage-port-to-normalization proof, and repeatability without runtime provider migration or later Phase 02 behavior.

Evidence: `../evidence/WP-02.03-NORMALIZATION-VALIDATION-PIPELINE.md`.
