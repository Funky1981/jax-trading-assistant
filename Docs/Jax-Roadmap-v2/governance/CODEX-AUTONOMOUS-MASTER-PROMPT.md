# Jax Codex Autonomous Master Prompt

## Read first

1. `Docs/ROADMAP.md`
2. `Docs/Jax-Roadmap-v2/JAX-ROADMAP-OVERVIEW.md`
3. `Docs/Jax-Roadmap-v2/governance/AUTONOMOUS-DEVELOPMENT-MODE.md`
4. `Docs/Jax-Roadmap-v2/governance/CODEX-OPERATING-RULES.md`
5. `Docs/Jax-Roadmap-v2/governance/MODEL-ROUTING-POLICY.md`
6. current phase `README.md`
7. current phase `GATE.md`
8. first incomplete work-package file

## Current position

Phases 00–03 are COMPLETE / GO. Begin or continue Phase 04 at the first incomplete package, expected WP-04.01 unless legitimate repository evidence shows later progress.

## Execution

Use GPT-5.6 Luna as the default implementation model. Proceed autonomously through the current phase one work package at a time.

For every package: inspect existing code, implement only that package, test success/material failure paths, adversarially self-review, correct findings, update evidence/status, create one bounded local commit, verify repository hygiene, and continue.

Do not stop merely because an ordinary work package is complete.

## Stop conditions

Stop when the phase exit condition is demonstrated, a governance hard stop occurs, a roadmap change is required, a consequential decision requires Sol technical-lead review, or repository state is unsafe/ambiguous.

At phase completion return the structure in `PHASE-REVIEW-HANDOVER.md`. Do not begin the next phase without external GO.

## Context recovery

If conversational context is lost, reconstruct state from `Docs/ROADMAP.md`, roadmap governance, current phase docs, evidence docs, recent local commits, tests and current repository state. Repository evidence is durable memory.

## Output format for external technical lead

Whenever Codex returns a phase handover, blocker, hard-stop report, roadmap-change request, or technical decision request, the entire response must be returned inside **one Markdown code block** for direct copy/paste to the external technical lead.

## Safety

Never enable live trading, broker execution, real-money orders or autonomous approval. Do not spend money or add paid dependencies without explicit approval. Do not push or destructively rewrite Git history.
