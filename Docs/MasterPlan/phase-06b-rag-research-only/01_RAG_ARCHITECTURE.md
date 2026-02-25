# RAG Architecture (Research-only)

## Components
1. Document store
   - Stores curated news snapshots, internal run artifacts, gate reports, and runbooks.
   - Must store `doc_id`, `doc_type`, `source`, `hash`, and content/ref.

2. Retriever
   - v1: keyword search (fast, transparent)
   - later: vector search (semantic) if needed

3. RAG Orchestrator (in cmd/research)
   - Retrieve top-k docs
   - Build prompt with citations
   - LLM produces strict JSON output + optional narrative
   - Store ai_decision with retrieved doc IDs + hashes

4. APIs
   - `/api/v1/research/rag/query`
   - `/api/v1/analysis/rag/query`
   - `/api/v1/research/rag/history`

## Replay
Store:
- prompt template version
- model name
- retrieved doc IDs + hashes
- final prompt payload

## Security
Only index curated sources (your snapshots + internal artifacts/runbooks).
No live web retrieval inside trader runtime.
