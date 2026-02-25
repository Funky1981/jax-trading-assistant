# RAG Research-Only Boundary (Non-Negotiable)

## Goal
Add Retrieval-Augmented Generation (RAG) to improve **research and analysis workflows** only.

## Definition (for this project)
RAG = retrieve relevant documents/data -> inject into LLM prompt -> LLM produces structured output and/or narrative with citations.

## Allowed (research-only)
RAG may be used for:
- Finding similar historical events (IBM-like) and summarising outcomes
- Pulling supporting evidence for an event classification (with citations)
- Summarising research runs, anomalies, and parameter sweep results
- Producing postmortems using internal run artifacts and gate reports
- Helping operators navigate runbooks and internal docs

## Disallowed (authoritative system paths)
RAG MUST NOT be used for:
- Order placement decisions
- Risk sizing decisions
- Stop/flatten logic
- Trust gate pass/fail authority
- Any real-time “fast path” signal generation logic
- Any mutation of strategy instance parameters without explicit human action

## Enforcement requirements
1. Runtime separation
   - RAG service must only be callable from `cmd/research` and read-only `analysis` endpoints.
   - `cmd/trader` must not import or call RAG packages (enforce via import boundaries).
2. API separation
   - RAG endpoints live under `/api/v1/research/rag/*` and `/api/v1/analysis/rag/*` only.
   - No `/api/v1/trading/*` endpoint can call RAG.
3. Data separation
   - RAG can read: event DB, run artifacts, curated doc store.
   - RAG cannot write: signals, orders, risk settings.
4. Audit
   - Every RAG query is stored as an `ai_decision` with purpose `rag_research` (schema validated).
   - Retrieved document IDs + hashes are logged for replay.

## Failure rule
If RAG fails (timeout, retrieval error, schema mismatch):
- Research continues without RAG output.
- No trading/execution is impacted.
