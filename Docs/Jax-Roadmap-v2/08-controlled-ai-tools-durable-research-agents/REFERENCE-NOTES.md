# Phase 08 Reference Notes — AI Tools & Durable Agents

## Fincept concepts
MCP guide:
- JSON-schema input validation;
- sync vs async separation;
- per-tool timeouts and cancellation;
- auth levels;
- explicit confirmation for destructive tools.

Agentic research notes:
- durable checkpoint state;
- adaptive replanning;
- reflection/self-correction;
- budgets;
- HITL interrupt;
- memory/skills;
- evaluation harness.

## Jax target
Tools should expose controlled capabilities such as market/history, filings, macro series, event evidence and quant calculations. A research agent may orchestrate these tools but cannot mutate trading state.

Avoid persona swarms until a benchmark proves they improve research quality.
