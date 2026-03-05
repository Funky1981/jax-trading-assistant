# API Contract Master Plan

## Core groups
- System: runtime/providers/modes/gates
- Events: list/detail/timeline/classify
- Strategy types + instances: metadata and CRUD
- Research: runs/projects/results/timelines
- Trading/Execution: signals/intents/orders/fills/positions/flatten
- AI Audit: ai-decisions and acceptance
- Testing/Gates: recon/replay/failure/provenance/gate history
- Artifacts: create/validate/promote/evidence

## API rules
- all writes audited
- pagination/filtering on list endpoints
- async jobs expose status + artifact refs
- provenance fields included in run/artifact responses
