<!-- Reused in phased pack from 14_RESEARCH_TESTING_TRUST_GATES_MASTER.md for phase scoping -->

# Research, Testing, and Trust Gates Master Plan

## Research requirements
- real deterministic backtests
- parameter sweeps
- walk-forward validation
- reproducibility checks
- dataset provenance
- artifact evidence packaging

## Gate set
- Gate 0 Config/Schema Integrity
- Gate 1 Data Reconciliation
- Gate 2 Deterministic Replay
- Gate 3 Artifact Promotion Controls
- Gate 4 Execution Path Integration
- Gate 5 P/L Reconciliation
- Gate 6 Failure Injection
- Gate 7 Flatten-by-Close Proof
- Gate 8 AI Audit Completeness
- Gate 9 Data Provenance Integrity
- Gate 10 Shadow/Parity Validation

## Promotion rule
No gate pass = no promotion = no trading.
