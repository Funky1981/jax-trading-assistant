# No Fake Data Implementation Plan

## Immediate tasks
1. Audit fake/synthetic/stub/demo generators (including `libs/utcp/backtest_local_tools.go`)
2. Disable/remove fabricated run generation outside test/dev
3. Add runtime modes and startup validation
4. Add provenance fields to runs and artifact evidence
5. Expose provenance via APIs and UI badges
6. Add provenance trust gate
7. Block artifact promotion from synthetic/unknown provenance

## Required fields (runs/artifacts)
- data_source_type
- source_provider
- dataset_id
- dataset_hash
- is_synthetic
- synthetic_reason
- provenance_verified_at

## Tests
- startup rejects fake provider in paper/live
- promotion rejects synthetic evidence
- provenance gate catches missing fields
