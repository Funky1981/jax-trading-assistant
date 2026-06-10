# Jax Read-Only Research Harness Package

Purpose:
Turn the current Jax assistant into a safer, stronger, evidence-backed research harness.

Core design:
- Research gets more freedom.
- Trading execution stays deterministic and separated.
- The assistant never approves or executes trades.
- The assistant must prefer evidence over fluency.
- Weak evidence must produce uncertainty or refusal, not confident guesses.

Package structure:
- docs/: phased implementation plan
- skeletons/: starter Go package layout for `internal/modules/harness`

Non-negotiable rules:
1. No assistant path may mutate approvals, candidates, execution instructions, or live trades.
2. Every research answer should be backed by Jax data, tool results, or explicitly marked external sources.
3. If evidence is missing, stale, contradictory, or low quality, the assistant must say so.
4. The model is advisory only.
5. Final trading decisions remain human.

Recommended build order:
1. Read `10-IMPLEMENTATION-ORDER.md`
2. Use `11-COPILOT-BUILD-BRIEF.md`
3. Implement phases in order
4. Keep execution and approvals outside the harness
