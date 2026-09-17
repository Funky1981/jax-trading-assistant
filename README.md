# Jax Trading Assistant

Jax is an evidence-first market research and trading decision platform with deterministic decisioning, explicit provenance, and human-controlled paper boundaries.

## Read first

- [Project overview](Docs/PROJECT_OVERVIEW.md)
- [Product charter](Docs/JAX_PRODUCT_CHARTER.md)
- [Roadmap](Docs/ROADMAP.md)
- [Current build package](Docs/BUILD/CURRENT_PACKAGE.md)
- [Current status](Docs/STATUS.md)
- [Documentation authority](Docs/DOCUMENTATION-AUTHORITY.md)

PAPER-01A remediation is complete and requires external re-review of PAPER-01. It does not demonstrate a trading edge, authorize PAPER-02, or permit live trading.

## Runtime

| Runtime | Port | Role |
|---|---:|---|
| `cmd/trader` | 8100 | Deterministic trader runtime and API |
| `cmd/research` | 8091 | Research, orchestration, replay, and memory |
| `ib-bridge` | 8092 | Explicit broker connectivity boundary |
| frontend | 5173 | Operator dashboard |

## Quick start

```powershell
.\start.ps1
```

See [Quickstart](Docs/SETUP/QUICKSTART.md) for setup details. Paper-trading validation remains hypothetical and bounded; use [local paper-trading testing](Docs/TESTING/LOCAL_PAPER_TRADING_TESTING.md) only when separately authorized.
