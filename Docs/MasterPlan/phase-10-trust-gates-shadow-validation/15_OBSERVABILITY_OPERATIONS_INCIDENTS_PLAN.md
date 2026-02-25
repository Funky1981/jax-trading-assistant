<!-- Reused in phased pack from 15_OBSERVABILITY_OPERATIONS_INCIDENTS_PLAN.md for phase scoping -->

# Observability, Operations, and Incidents Plan

## Observability
- structured logs with correlation IDs
- runtime metrics, gate metrics, data quality metrics
- broker latency/errors
- strategy/run metrics

## Operations
- one-command bootstrap
- env matrix (dev/research/paper/staging/live)
- migration/backup/restore runbooks
- secret management
- deployment runbooks

## Incidents
- incident records with severity/owner/status
- unresolved critical incidents block progression
- postmortems produce timeline, root cause, remediation, new tests
