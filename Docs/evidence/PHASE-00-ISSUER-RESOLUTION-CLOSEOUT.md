# Phase 00 issuer-resolution close-out evidence

Status: **GO**

Decision date: 2026-08-21

Evidence disposition: provider artifacts retained outside Git; compact identities and hashes retained in Git

## Accepted conclusion

Jax has demonstrated that the current causal-attribution architecture can reliably identify the affected issuer/instrument or safely remain unresolved:

`Event -> typed causal attribution -> deterministic policy -> DIRECT / PROXY / UNRESOLVED -> deterministic resolver`

The Luna unseen Generalization gates passed. The accepted Luna r3 repeat achieved semantic effective-mapping agreement of 46/48 (95.83%), passed every frozen quality/safety retention gate, produced zero incorrect deterministic ticker/rule resolutions, and produced zero safety/persistence violations. The Terra t1 challenger was materially better at 47/48 semantic mapping correctness and 5/6 PROXY recall while retaining all six quality/safety gates.

## Retained provider evidence inventory

All paths are repository-relative. The files under `.runtime/` remain ignored/local and were not modified or copied into Git.

| Evidence | Provider/model | Run ID | Retained artifact path | Artifact-index SHA-256 |
|---|---|---|---|---|
| Accepted Luna C1F3 Generalization baseline | OpenAI / `gpt-5.6-luna` | `0a650e09-1c64-4349-bf5d-09bf4dd697d9` | `.runtime/diagnostics/ai-shadow-issuer-hosted/openai-hosted-c1f3-generalization-v3/WP-00.03C1F3-GENERALIZATION/0a650e09-1c64-4349-bf5d-09bf4dd697d9/` | `681db26c0d3e83e85537671e519db55c2b77397b03feb0c8a9395dd87b0ea0cb` |
| Accepted Luna C1F3 r3 repeatability | OpenAI / `gpt-5.6-luna` | `77bf44ba-2d3e-49bb-b7b1-a3006def8c5c` | `.runtime/diagnostics/ai-shadow-issuer-hosted/openai-hosted-c1f3-repeatability-generalization-v3-r3/WP-00.03C1F3D-GENERALIZATION-R3/77bf44ba-2d3e-49bb-b7b1-a3006def8c5c/` | `ee6baeed27770d0f77581e49c6b99e3b9d1d974f90123272d6fad9d116f65439` |
| Terra frozen challenger t1 | OpenAI / `gpt-5.6-terra` | `03028685-69c2-42ed-9209-e6bb5c9a4f33` | `.runtime/diagnostics/ai-shadow-issuer-hosted/openai-hosted-c1f3-challenger-generalization-v3-terra-t1/WP-00.03T1/03028685-69c2-42ed-9209-e6bb5c9a4f33/` | `4e8ba9bc03e49b9361d315fdeb0631b33f1b79beee25ec73c78ff601801012f7` |

## Derived scoring and preservation evidence

| Evidence | Retained path | SHA-256 |
|---|---|---|
| Luna r3 semantic analysis | `.runtime/c1f3-r3-analysis.json` | `6acc2711056bc20f598784ceed98a69546db0fb73a1a38923f76aa7019b47a4f` |
| Luna baseline/r3 comparison score | `.runtime/c1f3-repeatability-r3-score.json` | `3788c31a5ada41c4a96227136a7dcb343c0a8285483a152f21942479423a80d0` |
| Terra challenger score | `.runtime/diagnostics/ai-shadow-c1f3-terra-challenger-score-v1/WP-00.03T1/03028685-69c2-42ed-9209-e6bb5c9a4f33/score.json` | `d201354fabc2cf7eadff885852d8edebcb66264f96a2db8fad8f27c46d0bf2b9` |
| Terra preservation manifest | `.runtime/diagnostics/ai-shadow-c1f3-terra-challenger-score-v1/WP-00.03T1/03028685-69c2-42ed-9209-e6bb5c9a4f33/preservation-manifest.json` | `47a3d17134434a421423009252696e6ce364aed140c5e7d4b69497b8631b9507` |

Terra was frozen by commit `d4548350f1994044d83e36546bef575b2c438b4d` (`test: freeze Terra C1F3 challenger cell`). The preservation manifest binds preflight `7cd247cb-acc4-4812-8c22-11332808c4d6` at SHA-256 `14b0360cc4fa6a140a005174104bed219c7b5e5e0ebd6687d4568764b77d8341`. It records 54 indexed Terra artifacts, 48 raw responses, a consumed cell, and no rerun authority.

## Frozen semantic and rubric bindings

| Binding | Identity | Fingerprint / SHA-256 |
|---|---|---|
| Generalization profile | `openai-hosted-c1f3-generalization-v3` | `dc38761583c8856db7b79d515d0f799e416581e84f369649a32aab3053dadd9d` |
| Luna r3 profile | `openai-hosted-c1f3-repeatability-generalization-v3-r3` | `0e66075694190cfa5aeac4457862d59dedc4965d46d4db22d24c1c2f70f19912` |
| Terra t1 profile | `openai-hosted-c1f3-challenger-generalization-v3-terra-t1` | `d58e9af61c2ac482a4641933ed422535c55400d862a571367693d42a7e20a915` |
| Generalization manifest | `ai-shadow-issuer-generalization-holdout-v3` | semantic `6e4cdd0133b1a12e980650e18786070586a522375c01d0ed0f2e61823a42f86c`; file `6abd7767e0031945e71f2a1d3ef49536adc2d9e9b7d4a78bd938f1f469d27502` |
| Generalization input lock | `ai-shadow-issuer-generalization-holdout-input-fingerprints-v3` | semantic `890dd084bc65828b839b250d5c0a7663016d11ee6bfade7a9b1dc7cd61e7d501`; file `0b6ab35b963c99dae44d1c43fa657dc23c202a65f6b78f17dcc73af0a145ee8a` |
| Typed labels | `ai-shadow-causal-attribution-labels-generalization-v3-v1` | semantic `46d5e6b09bea1b2c24b513052973ac776865e9a307bd13bf00443b0e2a1ff739`; file `702fc698ee0d17af97321074f49bd5e0e414260248aff77eca40e7e15e920bc4` |
| C1F3 scoring rubric | `ai-shadow-causal-attribution-scoring-c1f3-v1` | semantic `744a197191d2226005dd823b5793bafc86bf5e52164fd515efbe1efdc514162d`; file `7f795302a4c33c1a7c103294c1cab590a548c9e2cd746ca7908a10dc2a4a65a9` |
| Repeatability rubric | `ai-shadow-c1f3-repeatability-scoring-v1` | semantic `70313e7f77f5f82b912c382d96aa513f8e67dc03f53abe09c26189494677d1f5`; file `ff6fe9618ad8e89440e4d015724a9949c95f0542e6b0e0048bcb5d73f7e4c45d` |
| Terra challenger rubric | `ai-shadow-c1f3-terra-challenger-rubric-v1` | semantic `abd02e30a62f93516aa90488ef9e2b28f1611b9d75dfe5b4bc974c1367024ee0`; file `94ad6df14a58cc605d6498818c43dc585eaff84ae59f63528a12d5f105916ff2` |
| Prompt | `ai-shadow-prompt-v6-causal-attribution-boundaries` | `9a4ee7e3bcc5a2a7e1fdb8f5542ac158f2ae18f46fd457b21bcd643b150db3ca` |
| Output contract | `ai-shadow-output-v5-causal-attribution` | `8dc2a787bd7a33ec768a570d0d3588243561d85b06051d9e01bf2435fd88f960` |
| Validator | `ai-shadow-causal-attribution-validator-c1f-v1` | `0590cf5582d46bd08a0c053be6d83232ac00d83a06a212583190b0a3c0edc207` |
| Attribution policy | `ai-shadow-causal-attribution-c1e-v1` | `e319b3ceca80e9c2c43ab2edabb60402d8c9e534645b0dcb0e81baa953349c2e` |
| Semantic comparator | `ai-shadow-issuer-semantic-identity-v1` | `75cedcc94bc8d5c5eb7f4fabba83067ac00857744a10c806be77cea1733e51c8` |
| Deterministic resolver | `event-asset-resolution-v1` | `5170cc7c8dede5f1a095dfae1ffa7f0060bb8358c85b6dbb0609dab8ef30ede0` |
| C1F3 scorer | `ai-shadow-causal-attribution-scoring-c1f3-v1` | `f54ec91af32a9a11f3c0a7f27eefd3c52624d2360fce613820e7aa39380d8269` |

## Final model policy

- `gpt-5.6-luna` remains the default development/runtime hosted model.
- `gpt-5.6-terra` is a validated higher-capability challenger/escalation option.
- Terra may be reconsidered for high-value or ambiguous escalation, or when Jax economics justify it.
- No escalation architecture is authorized in Phase 00.
- Phase 00 model evaluation is closed: no Terra rerun/repeatability, Sol challenger, further model comparison, or prompt/schema tuning is authorized.

## Residual limitations

- Luna remains weaker than Terra on difficult macro/proxy attribution.
- Terra has only one-shot challenger evidence; repeatability is unmeasured.
- Terra case `issuer-generalization-v3-042` selected `US_RATES_CATEGORY` instead of expected `FEDERAL_RESERVE_OFFICIAL`.

These limitations are accepted and do not invalidate the Phase 00 GO.

## Safety boundary

The retained plans record paper runtime, live trading disabled, execution disabled, execution worker disabled, broker execution disabled, and maximum leverage 1x. The close-out created no approval, ticket, execution instruction, order intent, order, trade, or fill. No hosted inference was run during close-out, and `JAX_AI_HOSTED_INFERENCE_AUTHORIZED` remained absent.

## Independent check

Recompute SHA-256 for each listed `artifact-index.json` and derived evidence file with `Get-FileHash -Algorithm SHA256 -LiteralPath <path>`. Artifact indexes provide the per-artifact inventory needed to validate retained response content without committing provider responses or secrets.
