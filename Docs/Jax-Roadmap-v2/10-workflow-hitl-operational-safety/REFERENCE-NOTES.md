# Phase 10 Reference Notes — Workflow / HITL / Safety

## Fincept concepts
- WorkflowExecutor/NodeRegistry/ExpressionEngine.
- AuditLogger.
- ConfirmationService.
- RiskManager.
- Alpha Arena kill switches and crash recovery.

## Jax target
Do not recreate a general visual workflow platform. Build the explicit state machine Jax needs:
research -> recommendation -> risk -> human approval -> paper intent.

Every transition needs actor, time, input version, reason and recovery semantics. Destructive capabilities remain impossible without the required approval gate.
