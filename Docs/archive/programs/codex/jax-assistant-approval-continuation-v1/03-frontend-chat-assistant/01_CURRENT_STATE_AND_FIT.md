# Current State and Fit

## Current usable pieces
- research runtime exists in `cmd/research`
- orchestrate endpoint already exists
- orchestration run storage already exists
- strategy metadata endpoint already exists
- signal/trade data already available via trader API

## Missing pieces for chat
- chat API endpoint
- websocket or SSE streaming for assistant replies
- tool schema contracts for chat actions
- frontend chat UI
- assistant memory/session model
- explicit boundary preventing chat from executing trades
