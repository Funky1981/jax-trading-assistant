# DB Schema and Migrations Plan

## New / extended tables needed

### Truth and provenance
- extend research-related tables with provenance fields
- possibly `datasets` / `dataset_versions`

### Strategy and candidates
- `candidate_trades`
- `candidate_trade_events`

### Approval and execution
- `candidate_approvals`
- `execution_instructions`

### AI audit
- `ai_decisions`
- `ai_decision_acceptance`

### Chat
- `chat_sessions`
- `chat_messages`

### Gates
- `gate_results`
- `proof_artifacts`

## Migration sequencing
1. provenance and no-fake-data fields
2. candidate model
3. approval and execution instruction model
4. ai audit model
5. chat model
6. gates / proof model
