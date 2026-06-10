# Phase 1 - Boundary and Cleanup

Goal:
Make the current assistant boundaries explicit and impossible to bypass accidentally.

Required outcomes:
- One clear read-only assistant boundary
- No tool in assistant path can mutate trading state
- Centralized policy check before any tool runs
- Centralized system prompt builder instead of scattered strings

Work:
- Move assistant prompt construction into `internal/modules/harness/prompt_builder.go`
- Add `internal/modules/harness/policy.go`
- Add a single `AllowedTool(name string) bool`
- Add capability flags:
  - advisory_only
  - no_execution
  - no_approval
  - no_external_price_claims
- Make `ToolRouter` call the policy layer before dispatch
- Remove any future temptation to add write tools into the chat package directly

Acceptance:
- Unknown tools are rejected
- Known but forbidden tools are rejected with a policy reason
- Prompt always includes advisory-only boundary
- Assistant path cannot call execution or approval code
