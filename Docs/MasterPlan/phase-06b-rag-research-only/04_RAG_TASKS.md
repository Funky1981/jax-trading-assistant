# RAG Implementation Tasks (Codex)

1. Add DB migrations for `rag_documents`
2. Add ingestion utilities for curated docs + run artifacts + gate reports
3. Implement retriever (v1 keyword search)
4. Implement RAG orchestrator in `cmd/research` (schema output + ai_decisions logging)
5. Add endpoints:
   - /api/v1/research/rag/query
   - /api/v1/analysis/rag/query
   - /api/v1/research/rag/history
6. Add UI panels in Research/Analysis pages:
   - query box
   - retrieved sources list (hashes)
   - JSON export of answer
7. Enforce import boundaries: `cmd/trader` must not import RAG packages
8. Add tests for determinism of retrieval and for audit logging

Definition of done:
- RAG works only in research/analysis.
- Trading runtime cannot call RAG.
- Outputs are schema-validated, logged, replayable.
