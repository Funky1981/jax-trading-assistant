# 01 Always-On Trade Watcher

Goal: make Jax continuously evaluate enabled strategy instances and generate **candidate trades** without requiring the frontend to stay open.

This fits the current system because:
- `cmd/trader` is already the authoritative runtime
- `strategy_instances_loader.go` already loads instance configs
- `frontend_api.go` already has signal/recommendation/trade endpoints
