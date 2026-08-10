# WP-00.03 issuer diagnostic forensic analysis

## Provenance and scope

This is the pre-implementation failure taxonomy required by Phase 00 WP-00.03. It analyses the accepted, append-only WP-00.02 run `8b89113a-51d1-4688-a098-6120130ed808` without changing or regenerating it.

- Starting commit: `53b3dfe304e7796dad2c922836b6460b7a083cb5`
- Manifest fingerprint: `3419f1a4b5228ddee6d554ccda68d0bfd44075fa9e4e5657f4e0d550e6006fa5`
- Manifest file SHA-256: `544333c68cbca408fc8e39eb9e8d42d670e57a3a65cd94ddaca02dd57f7a3f6b`
- Input-fingerprint-lock SHA-256: `9de8c28b3fc100bdb77ed3baad912e0d3863050df50534f7c67483eb43e87739`
- Report JSON SHA-256: `637752258ebd8fb2befdf4da3992e843e5e85237c6fd0802364cd0c1b5d05024`
- Complete 224-file inventory SHA-256: `45718e68049fccc8855850135f723046e85171e9fb5126fc8756a8ef25ca8d92`
- Model: `ministral-3:8b`, digest `1922accd5827ebe6829e536369195db25eaf664528dc66206d646ea3bb386b71`
- Prompt: `ai-shadow-prompt-v4-issuer-resolution`
- Output contract: `ai-shadow-output-v4-issuer-resolution`
- Deterministic policy: `event-asset-resolution-v1`

The model-facing input path does not read this document. It must never be added to prompts, aliases, deterministic mappings, or corrective-retry hints.

## Notation

- `D(name)` = accepted `DIRECT`; `P(exposure)` = accepted `PROXY`; `U` = accepted `UNRESOLVED`.
- `X[P(name,NONE) -> P(NONE)]` = the first response was contract-invalid and the corrective retry remained invalid.
- `R[P(name,NONE) -> P(exposure)]` = the corrective retry made the response contract-valid, but did not necessarily make it semantically correct.
- `x3` means the identity fields and outcome were the same in all three repetitions. Case 022 varied only in explanatory wording, not in the identity fields shown here.
- Resolution shows the deterministic result after the model classification. `n/a` means validation correctly prevented resolution.

## Complete case taxonomy

| Case | Adjudicated semantic target | Model output by repetition | Validity / retry | Semantic / resolution | Primary taxonomy and secondary evidence | Deterministic fix legitimate? |
|---|---|---|---|---|---|---|
| 001 | `D(Apple Inc.) -> AAPL` | R1 `P(SP500_NAMED_INDEX)->SPY`; R2-R3 `P(SEMICONDUCTOR_GROUP)->SOXX` | `R[P(Apple,NONE) -> P(exposure)]` each run | wrong / wrong | Proxy/substitute; output-contract recovery discarded a correctly named issuer; exposure variation | No. Mapping either proxy to Apple would be answer leakage. |
| 002 | `D(Microsoft Corporation) -> MSFT` | `D(Microsoft)->MSFT` x3 | first-pass valid | correct / correct | Correct direct issuer | n/a |
| 003 | `D(NVIDIA Corporation) -> NVDA` | `X[P(NVIDIA,NONE) -> P(NONE)]` x3 | retry failed | invalid / n/a | Output-contract and corrective-retry failure; issuer text was initially recoverable | No. Validator rejection is correct; coercion would weaken the contract. |
| 004 | `D(Tesla, Inc.) -> TSLA` | `D(Tesla)->TSLA` x3 | first-pass valid | correct / correct | Correct direct issuer | n/a |
| 005 | `D(Amazon.com, Inc.) -> AMZN` | `X[P(Amazon,NONE) -> P(NONE)]` x3 | retry failed | invalid / n/a | Output-contract and corrective-retry failure; issuer text was initially recoverable | No. Coercing `PROXY/NONE` to `DIRECT` would reinterpret model output. |
| 006 | `D(Exxon Mobil Corporation) -> XOM` | `X[P(Exxon Mobil,NONE) -> P(NONE)]` x3 | retry failed | invalid / n/a | Output-contract and corrective-retry failure; issuer text was initially recoverable | No. Existing alias resolves the issuer when validly classified. |
| 007 | `P(OIL_CATEGORY) -> XLE` | `P(OIL_CATEGORY)->XLE` x3 | first-pass valid | correct / correct | Correct bounded proxy | n/a |
| 008 | `P(FEDERAL_RESERVE_OFFICIAL) -> TLT` | `P(US_RATES_CATEGORY)->TLT` x3 | first-pass valid | wrong / correct ticker | Other: exposure-provenance mismatch; same current ticker does not make the semantic label correct | No. The resolver correctly applied the exposure it received. |
| 009 | `P(GOLD_NAMED_MARKET) -> GLD` | `P(GOLD_NAMED_MARKET)->GLD` x3 | first-pass valid | correct / correct | Correct bounded proxy | n/a |
| 010 | `P(SEMICONDUCTOR_GROUP) -> SOXX` | `P(SEMICONDUCTOR_GROUP)->SOXX` x3 | first-pass valid | correct / correct | Correct bounded proxy | n/a |
| 011 | `P(SP500_NAMED_INDEX) -> SPY` | `P(SP500_NAMED_INDEX)->SPY` x3 | first-pass valid | correct / correct | Correct bounded proxy | n/a |
| 012 | `P(US_RATES_CATEGORY) -> TLT` | `P(US_RATES_CATEGORY)->TLT` x3 | first-pass valid | correct / correct | Correct bounded proxy | n/a |
| 013 | `U -> unresolved` | `U->unresolved` x3 | first-pass valid | correct / correct | Correct unknown/unresolved | n/a |
| 014 | `U -> unresolved` | `X[P(Meta,NONE) -> P(NONE)]` x3 | retry failed | invalid / n/a | Output-contract, corrective-retry, and ambiguity-handling failure | No. Resolving Meta would erase the adjudicated causal ambiguity. |
| 015 | `U -> unresolved` | `U->unresolved` x3 | first-pass valid | correct / correct | Correct unknown/unresolved | n/a |
| 016 | `D(Alphabet Inc.) -> ambiguous` | `P(SP500_NAMED_INDEX)->SPY` x3 | `R[P(Alphabet,NONE) -> P(SP500_NAMED_INDEX)]` | wrong / wrong | Proxy/substitute and ambiguity-handling failure | No. The resolver already returns share-class ambiguity for valid `D(Alphabet Inc.)`. |
| 017 | `U -> unresolved` | `U->unresolved` x3 | first-pass valid | correct / correct | Correct unknown/unresolved | n/a |
| 018 | `U -> unresolved` | `X[P(NONE) -> P(NONE)]` x3 | retry failed | invalid / n/a | Output-contract, corrective-retry, and ambiguity-handling failure | No. `PROXY/NONE` must remain invalid, not be silently retyped. |
| 019 | `U -> unresolved` | `X[P(Apple, Microsoft,NONE) -> P(NONE)]` x3 | retry failed | invalid / n/a | Output-contract, corrective-retry, and multi-issuer failure | No. Selecting either issuer would be benchmark leakage. |
| 020 | `U -> unresolved` | `P(OIL_CATEGORY)->XLE` x3 | first-pass valid | wrong / wrong | Proxy/substitute and multi-issuer failure | No. The resolver correctly obeyed the wrong proxy classification. |
| 021 | `U -> unresolved` | `P(SEMICONDUCTOR_GROUP)->SOXX` x3 | first-pass valid | wrong / wrong | Proxy/substitute, multi-issuer failure, and ticker-token violation (`AMD` in free text) | No. Mapping the proxy to unresolved would corrupt a valid policy mapping. |
| 022 | `U -> unresolved` | `P(US_RATES_CATEGORY)->TLT` x3 | `R[P(Amazon,US_RATES_CATEGORY) -> P(US_RATES_CATEGORY)]` | wrong / wrong | Proxy/substitute and multi-issuer failure; retry repaired only the cross-field contract | No. The wrong rates proxy is a model choice. |
| 023 | `U -> unresolved` | `P(US_RATES_CATEGORY)->TLT` x3 | first-pass valid | wrong / wrong | Proxy/substitute and multi-issuer failure | No. No deterministic rule can infer that this valid proxy output should be ignored only here. |
| 024 | `U -> unresolved` | `P(GOLD_NAMED_MARKET)->GLD` x3 | first-pass valid | wrong / wrong | Proxy/substitute and multi-issuer failure | No. Case-specific suppression would be leakage. |
| 025 | `D(Corning Incorporated) -> GLW` | `D(Corning)->GLW` x3 | first-pass valid | correct / correct | Correct direct issuer and alias resolution | n/a |
| 026 | `D(Humana Inc.) -> HUM` | `P(SP500_NAMED_INDEX)->SPY` x3 | `R[P(Humana,NONE) -> P(SP500_NAMED_INDEX)]` | wrong / wrong | Proxy/substitute; retry discarded a correctly named issuer | No. Existing Humana identity resolves correctly. |
| 027 | `D(Yum! Brands, Inc.) -> YUM` | `P(SP500_NAMED_INDEX)->SPY` x3 | `R[P(Yum Brands,NONE) -> P(SP500_NAMED_INDEX)]` | wrong / wrong | Proxy/substitute; retry discarded a correctly named issuer | No. Existing Yum Brands identity resolves correctly. |
| 028 | `D(Rivian Automotive, Inc.) -> RIVN` | `X[P(Rivian,NONE) -> P(NONE)]` x3 | retry failed | invalid / n/a | Output-contract and corrective-retry failure; issuer text was initially recoverable | No. Existing Rivian identity resolves correctly. |
| 029 | `D(eBay Inc.) -> EBAY` | `X[P(eBay,NONE) -> P(NONE)]` x3 | retry failed | invalid / n/a | Output-contract and corrective-retry failure; issuer text was initially recoverable | No. Existing eBay identity resolves correctly. |
| 030 | `D(Novo Nordisk A/S) -> NVO` | `X[P(Novo Nordisk,NONE) -> P(NONE)]` x3 | retry failed | invalid / n/a | Output-contract and corrective-retry failure; issuer text was initially recoverable | No. Existing Novo Nordisk identity resolves correctly. |
| 031 | `D(Meta Platforms, Inc.) -> META` | `P(SP500_NAMED_INDEX)->SPY` x3 | `R[P(Meta Platforms,NONE) -> P(SP500_NAMED_INDEX)]` | wrong / wrong | Proxy/substitute; retry discarded a correctly named issuer | No. Existing canonical identity resolves correctly at the receipt-time anchor. |
| 032 | `D(Ford Motor Company) -> F` | `D(Ford Motor)->F` x3 | first-pass valid | correct / correct | Correct direct issuer and alias resolution | n/a |
| 033 | `D(Visa Inc.) -> V` | `P(SP500_NAMED_INDEX)->SPY` x3 | `R[P(Visa Inc.,NONE) -> P(SP500_NAMED_INDEX)]` | wrong / wrong | Proxy/substitute; retry discarded a correctly named issuer | No. Existing canonical identity resolves correctly. |
| 034 | `D(NIKE, Inc.) -> NKE` | `D(NIKE)->NKE` x3 | first-pass valid | correct / correct | Correct direct issuer and alias resolution | n/a |
| 035 | `D(Reddit, Inc.) -> RDDT` | `P(SP500_NAMED_INDEX)->SPY` x3 | `R[P(Reddit Inc.,NONE) -> P(SP500_NAMED_INDEX)]` | wrong / wrong | Proxy/substitute; retry discarded a correctly named issuer | No. Existing canonical identity resolves correctly. |
| 036 | `D(Amazon.com, Inc.) -> AMZN` | `D(Amazon.com)->AMZN` x3 | first-pass valid | correct / correct | Correct direct issuer and alias resolution | n/a |
| 037 | `U -> unresolved` | `P(GOLD_NAMED_MARKET)->GLD` x3 | first-pass valid | wrong / wrong | Proxy/substitute; company mention was not causal | No. The model selected an exposure the resolver must consistently honor. |
| 038 | `U -> unresolved` | `P(GOLD_NAMED_MARKET)->GLD` x3 | first-pass valid | wrong / wrong | Proxy/substitute; company mention was not causal | No. A case-specific rejection would be leakage. |
| 039 | `U -> unresolved` | `D(Tesla)->TSLA` x3 | first-pass valid | wrong / wrong | Issuer-recognition semantic failure / false issuer; incidental mention treated as causal | No. Tesla is a genuine alias and must resolve when validly supplied. |
| 040 | `U -> unresolved` | `U->unresolved` x3 | first-pass valid | correct / correct | Correct unknown/unresolved | n/a |
| 041 | `U -> unresolved` | `X[P(NVIDIA,NONE) -> P(NONE)]` x3 | retry failed | invalid / n/a | Output-contract and corrective-retry failure; company mention was not causal | No. Resolving NVIDIA would hide the causal-classification failure. |
| 042 | `U -> unresolved` | `X[P(Ford,NONE) -> P(NONE)]` x3 | retry failed | invalid / n/a | Output-contract and corrective-retry failure; company mention was not causal | No. Resolving Ford would hide the causal-classification failure. |
| 043 | `D(Palantir Technologies Inc.) -> unresolved` | `P(SP500_NAMED_INDEX)->SPY` x3 | `R[P(Palantir,NONE) -> P(SP500_NAMED_INDEX)]` | wrong / wrong | Proxy/substitute; retry discarded the issuer | No. The benchmark intentionally expects a recognized but unsupported issuer to remain unresolved; adding an alias would change the target. |
| 044 | `D(Arm Holdings plc) -> unresolved` | `P(SP500_NAMED_INDEX)->SPY` x3 | `R[P(Arm Holdings,NONE) -> P(SP500_NAMED_INDEX)]` | wrong / wrong | Proxy/substitute; retry discarded the issuer | No. Adding an alias would contradict the frozen expected unresolved status. |
| 045 | `D(Ferrari N.V.) -> unresolved` | `P(FEDERAL_RESERVE_OFFICIAL)->TLT` x3 | `R[P(Ferrari,NONE) -> P(FEDERAL_RESERVE_OFFICIAL)]` | wrong / wrong | Proxy/substitute; retry invented an unrelated macro exposure | No. Alias addition or proxy remapping would be benchmark leakage. |
| 046 | `D(Spotify Technology S.A.) -> unresolved` | `P(SP500_NAMED_INDEX)->SPY` x3 | `R[P(Spotify,NONE) -> P(SP500_NAMED_INDEX)]` | wrong / wrong | Proxy/substitute; retry discarded the issuer | No. The frozen target deliberately tests recognized-but-unsupported identity. |
| 047 | `D(MercadoLibre, Inc.) -> unresolved` | `X[P(MercadoLibre,NONE) -> P(NONE)]` x3 | retry failed | invalid / n/a | Output-contract and corrective-retry failure; issuer text was initially recoverable | No. Adding an alias would change the frozen expected resolution status. |
| 048 | `D(Adyen N.V.) -> unresolved` | `X[P(Adyen,NONE) -> P(NONE)]` x3 | retry failed | invalid / n/a | Output-contract and corrective-retry failure; issuer text was initially recoverable | No. Adding an alias would change the frozen expected resolution status. |

## Counts and dominant root causes

The following primary categories are mutually exclusive and cover all 48 cases (144 executions):

| Primary outcome | Unique cases | Executions |
|---|---:|---:|
| Correct direct issuer | 6 | 18 |
| Correct bounded proxy | 5 | 15 |
| Correct unknown/unresolved | 4 | 12 |
| Proxy/substitute failure | 18 | 54 |
| Final output-contract plus corrective-retry failure | 13 | 39 |
| False direct issuer | 1 | 3 |
| Other: wrong exposure provenance with coincidentally correct ticker | 1 | 3 |

Cross-cutting evidence:

- Initial output-contract failures: 25 cases per repetition, 75 attempts. Of these, 23 cases per repetition had both a non-empty `direct_issuer` and `proxy_exposure=NONE`; one had only the missing bounded exposure; one had only the non-empty issuer violation.
- Corrective retries: 75. Thirty-six became contract-valid but remained benchmark-wrong; 39 remained invalid. A successful schema repair was therefore not evidence of semantic recovery.
- Proxy/substitute failures: 11 expected-DIRECT cases and 7 expected-UNRESOLVED cases, repeated consistently (54 executions).
- Ambiguity-handling failures: cases 014, 016, and 018 (9 executions). Cases 013, 015, and 017 were correctly unresolved. Case 016 should have reached deterministic share-class ambiguity but the model substituted SPY.
- Multi-issuer failures: all six cases 019-024 (18 executions); one remained contract-invalid and five selected unsupported proxies.
- Mentioned-but-not-causal failures: cases 037-039 and 041-042 (15 executions); case 040 was correctly unresolved.
- Unsupported-issuer failures: all six cases 043-048 (18 executions). The required behavior was `DIRECT` recognition followed by conservative deterministic unresolved, not alias expansion.
- Ticker-token violations: case 021 in each repetition (3 outputs), where `AMD` appeared in free text. No prohibited ticker/symbol JSON field appeared.
- Deterministic resolver failures after a semantically correct classification: **0**.
- Alias/canonicalization gaps demonstrated by the run: **0**.
- Model-induced deterministic-resolution mismatches among accepted outputs: **57**. The resolver consistently applied a wrong-but-contract-valid model classification.

## Legitimate deterministic hardening candidates

There are no issuer alias, canonicalization, ordering, ambiguity, or deterministic mapping changes justified by the frozen failures.

The only legitimate implementation candidates are generic diagnostic-analysis defects that do not change model behavior or benchmark answers:

1. Attribute resolution mismatches separately when they follow a semantically correct classification versus a model semantic mismatch. The v1 report's `incorrect_deterministic_ticker_resolutions` count is an end-to-end mismatch and is not proof that the resolver itself failed.
2. Label proxy-exposure and deterministic-ticker variations explicitly. The v1 variation classifier called case 001 a harmless wording variation even though its proxy exposure and resolved ticker changed.
3. Scan free-text ticker tokens in every structurally decodable attempt, including rejected first attempts and rejected final attempts, rather than only the final accepted classification.

These changes improve audit accuracy only. They do not alter the prompt, output validity rules, corrective retry, deterministic policy, aliases, model settings, manifest, labels, order, fingerprints, or the accepted WP-00.02 artifacts.

## Model/prompt capability failures that remain

Cases 001, 003, 005, 006, 014, 016, 018-024, 026-031, 033, 035, 037-039, and 041-048 remain model/output-contract capability failures. Deterministic mappings cannot legitimately repair them.

The dominant behavior is a model tendency to emit `PROXY` even when it names a direct issuer, followed by a corrective retry that deletes the issuer and either invents a bounded proxy or leaves `NONE`. Passing Phase 00 therefore appears to require an architecture decision about model/prompt/contract capability in a later authorized package; it cannot be achieved honestly by deterministic alias or fallback injection in WP-00.03.
