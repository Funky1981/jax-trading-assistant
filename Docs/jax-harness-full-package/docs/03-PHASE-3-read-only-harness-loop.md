# Phase 3 - Read-Only Harness Loop

Goal:
Introduce a real harness loop without making Jax autonomous.

Design:
- User asks question
- Harness builds context
- Harness asks model for either:
  - direct answer
  - tool request
- Harness validates requested tool
- Harness executes tool
- Harness stores result as evidence
- Harness gives evidence back to model
- Harness generates final answer
- Harness validates final answer before returning it

Important:
This is not an infinite loop agent. Keep it bounded.

Suggested limits:
- max_steps = 4
- max_tool_calls = 2
- max_tokens_answer = controlled by model settings

Stop conditions:
- sufficient evidence gathered
- no valid tool available
- validator rejects repeated weak answers
- step limit reached

Acceptance:
- Model can choose from allowed read-only tools
- Tool results are fed back into context
- Final answer is based on evidence bundle
