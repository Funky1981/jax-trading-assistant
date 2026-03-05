# RAG API Contracts (Research-only)

## POST /api/v1/research/rag/query
Body:
```json
{
  "query": "Find similar IBM-like event repricing cases and summarise outcomes",
  "filters": {"symbols":["IBM"], "from":"2020-01-01", "to":"2026-12-31"},
  "topK": 12
}
```

Returns:
```json
{
  "ragDecisionId": "uuid",
  "retrieved": [{"docId":"...","hash":"...","score":0.82,"title":"...","source":"reuters"}],
  "answer": {"summary":"...","keyPoints":["..."],"similarCases":[...],"unknowns":["..."]}
}
```

## POST /api/v1/analysis/rag/query
Same shape, but intended to link to a `runId` and use run artifacts as retrieval.

## Rules
- Read-only w.r.t. trading state
- Must create ai_decision rows for audit/replay
