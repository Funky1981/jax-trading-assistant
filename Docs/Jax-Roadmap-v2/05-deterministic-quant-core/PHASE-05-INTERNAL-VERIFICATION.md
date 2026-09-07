# Phase 05 internal verification — Deterministic Quant Core

**Status:** Exit condition demonstrated; external technical-lead phase review
pending.

## Scope and proof

The authorised sequence WP-05.01 through WP-05.09 is implemented on branch
`capability-reset`. The executable proof is
`internal/modules/quant/phase05_exit_test.go`. It constructs a fixed UTC
canonical dataset and benchmark, invokes every Phase-05 result family twice,
compares their serialized responses byte-for-byte, checks frozen-input and
algorithm-version identity, and independently recomputes representative
returns, volatility, ATR, drawdown, liquidity, sizing, and exposure values.

The proof covers:

- versioned frozen-dataset, request, result, algorithm, and secondary-input
  contracts;
- simple, log, and benchmark-relative returns;
- volatility, ATR, and drawdown;
- correlation, beta, and covariance through the package known-value tests;
- provenance-aware liquidity/volume anomaly metrics;
- Sharpe, Sortino, and Calmar;
- bounded position sizing and signed portfolio exposure;
- library-selection/dependency evaluation with no new runtime or paid service.

## Reproducible verification

```text
go test ./internal/modules/quant -count=1                         PASS
go test ./internal/modules/quant -count=100 -shuffle=on           PASS
go test ./internal/modules/quant -run 'Phase05Exit' -count=1      PASS
go test ./... -count=1                                            PASS
go vet ./...                                                       PASS
go test ./tests/golden ./tests/replay ./db/postgres/migrations -count=1 PASS
```

Roadmap-pack manifest recomputation is valid. `git diff --check` is clean for
the Phase-05 diff. No Phase-05 migration was added; the existing migration
test package passes. No live-source call, paid service, credential, order,
trade, fill, recommendation, approval, or execution mutation is part of the
implementation or proof.

## Adversarial review record

The dedicated fresh review pass inspected the actual diff from
`ec7774e..HEAD`, all quant production and test files, the phase gate, package
evidence, dependency boundary, roadmap status, and manifest. Test-the-tests
checks included repeated shuffled execution, independent arithmetic fixtures,
negative-path coverage, and regression tests for latest-window indexing,
secondary SHA-256 validation, and non-zero-risk-free Sortino semantics.

Two material findings were corrected and reverified:

1. Risk-adjusted latest-window indexing skipped the wrong starting bar and
   could panic on the exact-size fixture. It now starts at
   `len(bars)-1-window`.
2. Secondary input references accepted arbitrary 64-character strings, and
   Sortino used the Sharpe excess-return numerator. Input digests now require
   canonical lowercase hexadecimal SHA-256, and Sortino uses mean return less
   periodic downside target.

Final result: **Adversarial phase review: PASS**. No unresolved blocking
finding remains. The remaining limitation is intentional: advanced
optimization, tail-risk, statistical modelling, venue lot-size, leverage,
fees, slippage, and live/paper execution semantics remain outside Phase 05.
