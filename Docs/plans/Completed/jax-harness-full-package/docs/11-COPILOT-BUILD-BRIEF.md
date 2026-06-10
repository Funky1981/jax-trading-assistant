# Copilot Build Brief - Jax Harness

Build a production-grade read-only research harness for Jax.

Constraints:
- advisory only
- no trade execution
- no approvals
- no mutation of trading state
- bounded loop only
- evidence-backed answers
- validator-enforced uncertainty when evidence is weak

Implement package:
`internal/modules/harness`

Files:
- registry.go
- policy.go
- prompt_builder.go
- evidence.go
- validator.go
- trace.go
- service.go

Integrate with existing chat module by:
- replacing direct prompt text with prompt builder
- using registry metadata for tool discovery
- routing tool calls through policy check
- adding a bounded model->tool->answer loop
- validating final answer before persistence

Quality bar:
- simple
- testable
- explicit
- fail closed
- no autonomous trading behavior
