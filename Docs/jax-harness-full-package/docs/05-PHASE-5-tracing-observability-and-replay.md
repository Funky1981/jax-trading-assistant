# Phase 5 - Tracing, Observability, and Replay

Goal:
Make every advisory answer auditable.

Trace record should capture:
- session id
- question
- selected tools
- tool inputs
- tool outputs summary
- evidence ids used
- model request metadata
- validator failures
- final answer classification

Why it matters:
If a bad answer influences a trade decision, you need to know:
- what evidence existed
- what the model saw
- what tools ran
- whether the validator warned
- whether the evidence was weak or stale

Acceptance:
- Each assistant answer has a trace id
- Trace can be replayed for debugging
- Tool usage is visible in UI or admin logs
