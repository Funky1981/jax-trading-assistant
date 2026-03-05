# RAG Data and DB Schema Additions

## Tables (minimum)
1) rag_documents
- doc_id (PK)
- doc_type (news_snapshot, run_artifact, gate_report, runbook, misc)
- symbol (nullable), event_id (nullable), run_id (nullable)
- source, title (nullable)
- hash_sha256
- content_text (text) OR content_ref (url/path)
- created_at

2) ai_decisions (reuse AI audit tables)
- store rag queries with purpose = rag_research
- store retrieved_docs[] (ids + hashes + score)

## Provenance
- inserted_by (system/user)
- inserted_reason
- is_synthetic (must be false for any real-world doc)
