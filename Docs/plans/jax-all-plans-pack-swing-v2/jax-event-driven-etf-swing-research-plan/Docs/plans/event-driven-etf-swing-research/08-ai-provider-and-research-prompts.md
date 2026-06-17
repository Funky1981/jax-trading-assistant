# 08 — AI Provider and Research Prompts

## Goal

Keep AI provider configurable. Chris can use DeepSeek personally because it is cheaper, while final release supports multiple providers.

## Provider Abstraction

Create or extend:

```text
internal/modules/airesearch
```

Interface:

```go
type ResearchProvider interface {
    SummarizeEvent(ctx context.Context, req SummarizeEventRequest) (SummarizeEventResult, error)
    ClassifyEvent(ctx context.Context, req ClassifyEventRequest) (ClassifyEventResult, error)
    SuggestETFMapping(ctx context.Context, req ETFMappingRequest) (ETFMappingResult, error)
    ExplainEvidence(ctx context.Context, req ExplainEvidenceRequest) (ExplainEvidenceResult, error)
    BuildBeginnerSummary(ctx context.Context, req BeginnerSummaryRequest) (BeginnerSummaryResult, error)
}
```

Provider config:

```text
AI_PROVIDER=deepseek
AI_MODEL=deepseek-chat
AI_BASE_URL=https://api.deepseek.com
AI_API_KEY=...
AI_MAX_DAILY_COST_USD=2.00
AI_MAX_TOKENS_PER_EVENT=4000
```

Supported providers:

```text
ollama
lmstudio
deepseek
openai
anthropic_later_optional
mock_test_provider
```

## AI Authority Rules

AI can:

```text
summarise
classify
suggest affected ETFs
explain evidence
list confounders to check
create beginner-friendly notes
```

AI cannot:

```text
create execution instructions
approve candidates
override guardrails
set risk size directly
switch broker mode
bypass paper-only policy
```

## Required JSON Output

Every AI call must use strict JSON schema. If invalid JSON is returned:

```text
retry once with repair prompt
then fail closed
```

## Prompt Versioning

Every prompt must include:

```text
prompt_version
provider
model
input_hash
output_hash
```

Store in `ai_research_audit`.

## Swing Event Classification Prompt

Purpose:

```text
Classify a news/event item for ETF swing research. Do not recommend trades. Return structured research metadata only.
```

Required output:

```json
{
  "event_type": "central_bank",
  "themes": ["rates", "risk_on"],
  "affected_etfs": ["TLT", "QQQ", "SPY"],
  "primary_etf": "TLT",
  "time_sensitivity": "high",
  "swing_research_worthiness": 0.71,
  "intraday_research_worthiness": 0.55,
  "source_quality_notes": "official central bank transcript plus Reuters coverage",
  "possible_confounders_to_check": ["CPI tomorrow", "Treasury auction", "earnings mega-cap"],
  "reason": "Rate expectations can affect bonds and growth equities over multiple sessions.",
  "trade_instruction": null
}
```

Validation:

- `trade_instruction` must be null.
- ETF list must be checked against deterministic allowlist.
- AI ETF suggestions are advisory only.

## Beginner Evidence Summary Prompt

Purpose:

```text
Explain the candidate in plain English for approval review.
```

Must include:

```text
what happened
why this ETF
what history says
whether news may be priced in
what could invalidate the trade
how long it may be held
what Jax will check daily
why Jax might walk away
```

Must not include:

```text
guaranteed profit
language implying certainty
instruction to buy/sell now
```

## Cost Controls

- Cache event summaries by content hash.
- Do not call cloud AI for duplicate headlines.
- Use local provider for obvious rejects.
- Use DeepSeek for normal personal deployment.
- Escalate to stronger provider only if configured.
- Daily cost cap hard-stops provider calls.

## Tests

- Mock provider returns deterministic JSON.
- Invalid JSON fails closed.
- AI output containing order instruction is rejected.
- DeepSeek/OpenAI providers use same interface.
- Provider can be swapped without changing strategy code.
- Audit row is written for each cloud AI call.
