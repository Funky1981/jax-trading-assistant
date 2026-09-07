# Phase 06 internal verification — Research & Recommendation Engine

**Status:** Exit condition demonstrated; external technical-lead phase review
pending.

## Scope and proof

The authorised sequence WP-06.01 through WP-06.07 is implemented on branch
`capability-reset`. The executable proof is
`internal/modules/researchrecommendation/phase06_exit_test.go`. It constructs a
fixed UTC packet containing instrument, company, market, macro, World Monitor
and Phase-05 quant evidence; builds a bounded context; validates a structured
research artifact; evaluates freshness and sufficiency; applies deterministic
eligibility; attaches explicitly uncalibrated confidence; and builds the
research-only recommendation read model.

The proof independently checks the known quant value (`1.25`), six independent
sources, all required research surfaces (thesis, bull case, counter-evidence,
contradiction, unknown and invalidation), byte-for-byte JSON reproducibility,
content identities, context bounds, and the absence of approval/order/trade/fill
state. The retained raw inference artifact is hashed for provenance but is not
included in the UI/API read model.

## Work-package commits

| Package | Commit | Internal result |
| --- | --- | --- |
| WP-06.01 | `d991b5f` | Evidence packet contract and Phase-06 authorization record |
| WP-06.02 | `358e20c` | Bounded planner, deterministic selection, deduplication and incremental context |
| WP-06.03 | `a6a98cf`, `6662dc3` | Validated thesis/cases/contradictions/unknowns and hardened identities |
| WP-06.04 | `1dbf523` | Research-only recommendation grammar and fail-closed eligibility |
| WP-06.05 | `cbb2e3a` | Freshness and evidence sufficiency gates |
| WP-06.06 | `e0cd407` | Explicitly uncalibrated confidence contract |
| WP-06.07 | `359de13` | Deterministic UI/API read model and bounded provider boundary |

## Reproducible verification

```text
go test ./internal/modules/researchrecommendation -run Phase06Exit -count=1 PASS
go test ./internal/modules/researchrecommendation -count=1 PASS
go test ./internal/modules/researchrecommendation -run 'Phase06Exit|ExecuteStructuredResearch|RecommendationReadModel' -count=100 -shuffle=on PASS
go test ./... -count=1 PASS
go vet ./... PASS
go test ./tests/golden ./tests/replay ./db/postgres/migrations -count=1 PASS
git diff --check PASS
```

Roadmap-pack manifest recomputation is valid. No Phase-06 migration was added;
the existing migration tests pass. No live-source call, paid service,
credential, approval, order, trade, fill or execution mutation is part of the
implementation or proof.

## Adversarial review record

The dedicated fresh review pass inspected the actual Phase-06 diff from
`34deb6f..CURRENT_HEAD`, all production and test files in
`internal/modules/researchrecommendation`, roadmap status/evidence, the
manifest, dependency/provider boundary and existing execution/approval
packages. It reread the Phase-06 gate and every package acceptance criterion,
then test-tested the phase proof with independent expected values and repeated
shuffled execution.

Material findings corrected and reverified:

1. Recommendation construction silently normalized non-UTC timestamps. It now
   rejects non-UTC or zero timestamps.
2. Context-plan, research-output and recommendation IDs summarized too few
   fields. They now hash their complete contract bodies with the identity field
   removed; tampered rendered context and structured claims fail validation.
3. The execution boundary did not enforce the declared retry budget. It now
   caps attempts at `MaxRetries + 1`, stamps execution-owned retry provenance,
   validates each artifact against the exact packet/plan, and reports provider
   versus validation failures.
4. The read model did not fully revalidate bound artifacts or packet references.
   It now checks packet/context/research/freshness/confidence identities,
   packet content fingerprint and selected/duplicate/omitted evidence
   references, while omitting raw response content.

Final result: **Adversarial phase review: PASS**. No unresolved blocking finding
remains.

Remaining limitations are intentional: no hosted provider or transport endpoint
is selected in this phase; the provider is an injected boundary, persistence is
not introduced, stochastic re-inference is not claimed to be bit-exact, and
empirical confidence calibration remains a later evaluation concern. No later
phase capability is started.
