# Jax Codex Model Routing Policy

## Objective

Use the least expensive model proven adequate for the work while reserving higher-capability reasoning for consequential decisions.

## Default split

### GPT-5.6 Luna — implementation

Use Luna by default for repository exploration, routine coding, bounded refactors, tests, fixtures, already-authorised migrations, documentation, repetitive edits, build/test/debug loops, evidence generation, and roadmap status maintenance.

Luna is the default Codex implementation model for Jax.

### GPT-5.6 Sol — decisions and review

Use Sol for phase-level technical-lead review, GO/CONDITIONAL GO/NO-GO/ROADMAP CHANGE decisions, consequential architecture choices, difficult root-cause analysis after Luna materially struggles, safety-boundary changes, major schema/runtime/infrastructure decisions, validation-methodology decisions, and decisions about paid services/data/model escalation.

Normal workflow:

`Luna implementation -> phase handover -> Sol technical-lead decision`

## Optional internal escalation

If the active Codex runtime explicitly exposes model-specific agents/subagents and runtime metadata confirms the requested model and effort, Codex may request a bounded Sol review for a consequential decision.

Do not claim a Sol review occurred unless runtime evidence confirms the model used.

If model-specific delegation is unavailable or ambiguous, stop on the hard decision and use the normal external Sol review boundary.

## Terra

Terra is an optional middle tier when Luna materially struggles with normal implementation but the problem does not justify Sol.

Do not escalate merely because a package is large.

## Reasoning effort

Recommended defaults when selectable:

- Luna: Medium/High for normal implementation;
- Sol: High for phase review and consequential technical decisions;
- Terra: Medium/High for bounded escalation.

Do not use maximum effort by default.

## Cost policy

Development-model routing is separate from Jax runtime hosted-model policy. Existing hosted-model budget gates remain authoritative for runtime inference.
