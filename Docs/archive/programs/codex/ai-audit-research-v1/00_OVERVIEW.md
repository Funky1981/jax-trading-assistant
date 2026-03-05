# AI Decision Verification + Research/Test Orchestration (Jax)

You asked for:
1) Verify **every decision** the AI makes.
2) Trigger **research** and **tests**.
3) Define **how AI is involved** (what it checks).
4) Save **all info** for analysis.

This pack defines an **audit-first, replayable** architecture.

Core principle:
- AI is allowed to *recommend*, *classify*, *summarise*, and *explain*.
- Deterministic code is responsible for *risk*, *orders*, *P/L*, and *gates*.
- Every AI output must be stored with: input, model, prompt/template, tool calls, confidence, and a correlation id.

