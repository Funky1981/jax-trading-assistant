# Codex Execution Order

Use this rebaselined order.

## Block 0 - Rebaseline
1. mark already-landed capabilities as complete or partial
2. keep ADR-0012 runtime boundaries intact
3. start with contract hardening and approval-driven execution, not page creation

## Block 1 - Truth path and strategy-instance contract hardening
4. canonicalize `strategy_instances.configJson.symbols`
5. preserve backward-compatible reads for legacy `universe`
6. tighten enabled-instance validation
7. extend candidate provenance to signal / strategy / artifact

## Block 2 - Watcher and candidate lifecycle
8. persist blocked candidates with structured reason codes
9. preserve duplicate suppression while recording blocked reasons
10. emit consistent lifecycle SSE events
11. expose candidate approval / execution linkage in DTOs

## Block 3 - Approval-driven paper execution
12. make `execution_instructions` the only paper execution path
13. run a trader-runtime worker that consumes pending instructions
14. write back instruction status, trade linkage, and fills
15. reject operator-originated direct `/api/v1/execute` in paper mode

## Block 4 - Assistant and AI audit
16. expand assistant tools for queue, blocked candidates, and knowledge search
17. keep assistant useful without `OPENAI_API_KEY`
18. record assistant decisions and tool use in `ai_decisions` / `ai_decision_acceptance`
19. expose linkage fields in AI decision APIs

## Block 5 - Trust gates and sign-off
20. harden Gate4 for candidate -> approval -> instruction -> trade -> fill linkage
21. harden Gate7 for post-flatten proof artifacts
22. harden Gate8 for orchestration + assistant AI audit coverage
23. harden Gate9 for run + candidate + artifact provenance
24. publish paper-readiness evidence under `reports/` and `Docs/runs/`

## Final sign-off
25. require Gate0-Gate9 passing
26. require Gate10 in staging when shadow DB is configured
27. require 20+ paper sessions with no unresolved P0 issues
