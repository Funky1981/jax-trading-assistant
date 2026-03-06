# 03 Frontend Chat Assistant

Goal: add a true frontend assistant that can answer questions about scenarios, trades, strategies, runs, and blockers — without becoming the trading authority.

This fits the current system because:
- `cmd/research/main.go` already hosts orchestration in-process
- `frontend_api.go` already exposes strategies/signals/trades/runs
- the phased plan already allows research-only RAG

This folder defines the assistant as a **read-mostly tool-using chat layer**.
