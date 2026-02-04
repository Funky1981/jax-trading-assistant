# Jax Strategy Knowledge Base (MD-first)

This repository is a **strategy + risk knowledge layer** for Jax.

Design goals:
- **Baseline** known strategies with clear assumptions and failure modes
- **Quarantine** for newly discovered strategies (no auto-deploy)
- **Anti-patterns** (strategies to avoid) as explicit safety rails
- **Evidence-driven evaluation** (costs, slippage, overfitting controls)
- **Explainability**: Jax must be able to say *why* it is trading something

> **Important:** This is not financial advice. It is an engineering-friendly knowledge base
> describing strategies, their assumptions, and controls.

## Folder map
- `strategies/known/` — vetted templates for common approaches
- `strategies/discovered/` — candidates; must pass the evaluation pipeline
- `patterns/` — reusable signals/features (not full strategies)
- `anti-patterns/` — things that look profitable but usually blow up
- `risk/` — position sizing, drawdown control, kill-switch rules
- `meta/` — regime detection + strategy selection logic
- `evaluation/` — backtesting standards to avoid self-deception

## Strategy lifecycle (tldr)

1. **Candidate** (discovered) → 2. **Paper trade** → 3. **Small capital** → 4. **Approved** → 5. **Monitor & retire**

Start with: `meta/strategy_lifecycle.md` and `evaluation/evaluation_protocol.md`.
