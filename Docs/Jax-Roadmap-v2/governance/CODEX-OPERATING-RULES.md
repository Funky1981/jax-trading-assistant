# Codex Operating Rules

Apply these rules to every roadmap work package.

1. Verify repository, branch, HEAD, upstream, ahead/behind, working tree and untracked files before changes.
2. Do not reset, rebase, clean, stash, discard, overwrite, amend, switch branch or push unless explicitly instructed by the user.
3. Work only on the named work package. Do not opportunistically implement later roadmap items.
4. Preserve current paper-safe boundaries unless the work package explicitly changes them after a future approved gate.
5. Never silently create or enable live execution, broker orders, fills, leverage, approvals or execution intents.
6. Add tests/evidence that directly prove the work package acceptance criteria.
7. Document deviations rather than hiding them.
8. At completion, produce the standard `CODEX-REVIEW-HANDOVER` and STOP.
9. Do not start the next package until the architecture reviewer returns GO or accepted CONDITIONAL GO.
10. If a work package reveals a roadmap assumption is wrong, stop and request `ROADMAP CHANGE`; do not patch around the architecture.
