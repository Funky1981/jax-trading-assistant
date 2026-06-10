# Phase 2 - Tool Registry and Contracts

Goal:
Replace ad hoc tool switching with a structured registry.

Why:
Your current switch-based router is safe but limited. A registry makes tools easier to grow safely.

Required outcomes:
- Tool metadata stored centrally
- JSON input schema per tool
- Output classification per tool
- Evidence flags per tool
- Read-only enforcement per tool

Tool contract fields:
- Name
- Description
- ReadOnly
- InputSchema
- OutputKind
- EvidenceLevel
- FreshnessExpectation
- Handler

Suggested output kinds:
- entity_lookup
- list_result
- explanation_support
- research_reference
- system_status

Suggested evidence levels:
- hard_internal_data
- derived_internal_data
- weak_inference

Acceptance:
- Frontend can discover tools from registry metadata
- Tool args are validated before execution
- Every tool advertises whether it is safe for trading advisory use
