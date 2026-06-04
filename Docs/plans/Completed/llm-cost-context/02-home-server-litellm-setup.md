# 01 — Home Server LiteLLM Setup

Run privately:

```text
LiteLLM Proxy
Redis
Postgres
Ollama
optional Qdrant later
```

Folder:

```text
~/ai-gateway/
  docker-compose.yml
  .env
  litellm/config.yaml
```

Local-first model aliases:

```text
local-small
local-coder
paid-cheap
paid-strong
```

Do not expose LiteLLM publicly.

Acceptance:

```text
local model works through LiteLLM
usage block includes tokens
Redis cache enabled
Postgres stores LiteLLM state
paid APIs disabled by default
```
