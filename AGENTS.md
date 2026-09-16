# Jax repository instructions

## Skills

Use the minimum applicable repository skill for every task. Start with `skills/STARTER_PROMPT.md`, then read the selected `skills/*/SKILL.md` before substantive work. For documentation-only work, use `jax-repo-guardrails` and keep runtime behaviour unchanged.

## Documentation authority and read order

Before any Jax task, read in this order:

1. `Docs/JAX_PRODUCT_CHARTER.md`
2. `Docs/ROADMAP.md`
3. `Docs/STATUS.md`
4. `Docs/CAPABILITY_MATRIX.md`
5. `Docs/BUILD/CURRENT_PACKAGE.md`
6. `Docs/ARCHITECTURE.md` when architecture is affected
7. Relevant domain documents
8. Relevant repository skill

Do not derive current work from retired `Docs/PHASE_CONTRACTS/`, ProjectOS template roadmaps/current-focus files, archived plans, or `Docs/Jax-Roadmap-v2/NEXT-WORK-PACKAGE.md` unless the current package links them as historical/supporting evidence.

## Default safety

- Preserve `NO_TRADE` as the default decision.
- Do not implement live trading, broker execution, auto-execution, or day trading in the current roadmap.
- Keep exploratory paper observations separate from formal evidence and promotion claims.
- Do not alter immutable validation results or historical evidence to make current status appear stronger.

## Completion requirements

For completed implementation or documentation packages:

- verify the scoped work;
- commit the scoped changes before responding;
- do not leave completed work uncommitted;
- provide a copyable fenced `text` handover summary headed `HANDOVER SUMMARY`;
- include phase/package, commit, branch, files changed, what changed, migrations, tests/verification, known risks, recommended next phase, and `What's Left`.
