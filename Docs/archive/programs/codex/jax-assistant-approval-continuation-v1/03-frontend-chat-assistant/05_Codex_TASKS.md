# Codex Tasks

1. Add chat session/history schema
2. Add chat API endpoint(s)
3. Add tool router for read-mostly assistant tools
4. Connect assistant to research runtime / RAG where available
5. Add frontend assistant page/panel
6. Add streaming response transport
7. Add clear safety banner and permission boundary
8. Add tests:
   - assistant cannot execute or approve trades
   - tool calls resolve current trade/signal/run data
   - chat session history persists
   - invalid tool call returns safe error

## Definition of done
- you can ask Jax about scenarios and current candidates
- assistant replies are grounded in Jax data/tools
- assistant cannot become the execution authority
