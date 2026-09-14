# Jax VAL-03 MA Crossover Validation Summary

## Terminal status

`CONTAMINATED_FOR_FORMAL_OOS`

This result package is retained for audit. It is not a valid scientific
promotion or rejection of `ma_crossover_v1`.

## Frozen experiment identity

- Candidate: `ma_crossover_v1`.
- Manifest: v1.5, SHA-256
  `96a0a21ae6b4d25047da0b90e34bdc839b241e5bc4b2da5a37f89135bea9360a`.
- Dataset readiness SHA-256:
  `b7081deb3bf55be5a23c1f1fcafbe44ff90c4fcee924de47f738cdcad86861ce`.
- Formal OOS: `2023-01-01..2024-12-31`.
- Holdout: `2025` sealed and not accessed.
- Formal pipeline attempts: `1`.

## Integrity finding

The formal primary partition path used an actionable confidence gate of `1.0`
instead of the frozen operational `0.60` gate. The robustness helper used
`0.60`, so the pipeline was internally inconsistent. The raw artifacts report
zero primary actionable-long episodes and an `INSUFFICIENT_EVIDENCE` result,
but that output cannot be interpreted as evidence for the frozen strategy.

The post-OOS no-rerun rule prohibits correcting the defect and rerunning this
formal OOS. The final classification is consequently
`CONTAMINATED_FOR_FORMAL_OOS`.

## Artifact interpretation

The following artifacts are the unmodified outputs of the single invalid
execution:

- `VAL-03-HYP-MA-001-DATASET.json` — data-quality output only.
- `VAL-03-HYP-MA-001-DEVELOPMENT.json` — raw partition output; not a valid
  performance result.
- `VAL-03-HYP-MA-001-VALIDATION.json` — raw partition output; not a valid
  performance result.
- `VAL-03-HYP-MA-001-OOS.json` — raw formal OOS output; not interpretable.
- `VAL-03-HYP-MA-001-FALSIFICATION.json` — raw registered diagnostic output;
  not interpretable as a coherent suite because the primary path used the
  wrong gate.
- `VAL-03-HYP-MA-001-RUN-MANIFEST.json` — raw run manifest, including the raw
  classification and run metadata.

`VAL-03-HYP-MA-001-INTEGRITY-REVIEW.json` is the authoritative post-run
integrity disposition.

## Scientific conclusion

No valid VAL-03 promotion conclusion was generated. Trading edge remains
`NOT DEMONSTRATED`. Real forward-paper evidence remains `0 DAYS / 0 ORDERS`.

## Safety conclusion

No broker calls, orders, fills, approvals, positions, hosted inference, paid
data calls, forward paper or Phase 13 activity occurred.
