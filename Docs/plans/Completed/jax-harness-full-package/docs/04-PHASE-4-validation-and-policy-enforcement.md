# Phase 4 - Validation and Policy Enforcement

Goal:
Reduce harmful hallucinations and unsupported trading claims.

Truth:
You cannot make an LLM incapable of hallucinating.
You can make unsupported outputs fail validation and be rewritten or refused.

Validation layers:
1. Policy validation
   - no execution language
   - no approval language
   - no "I placed" or "I executed"
2. Evidence validation
   - claims about Jax state must be backed by evidence
   - stale evidence must be labeled stale
3. Trading-safety validation
   - no certainty language from weak evidence
   - no unsupported price targets
   - no fabricated real-time market statements
4. Output-shape validation
   - must include uncertainty when evidence is weak
   - must cite evidence ids or tool names in internal trace

Failure behavior:
- first failure: regenerate with validator feedback
- second failure: refuse with reason
- log both failures

Acceptance:
- Unsupported claims are blocked
- Weak evidence forces uncertainty language
- External data claims are rejected unless source is present and allowed
