# Jax Harness Evaluation Specification

Status: **DESIGNED / HARNESS-02 FOUNDATION FIXTURES IMPLEMENTED / EXTERNAL
REVIEW REQUIRED**.

This specification defines how the engineering harness and the separate
Jax runtime/research harness will be evaluated. It deliberately does not define
a single aggregate “harness score.” A component is justified only when its
measured benefit is material for the task class and worth its cost and
complexity.

## 1. Evaluation rules

- Use fixed, versioned scenario fixtures with known objective, constraints,
  evidence identities, memory identities, tool outcomes, and expected safety
  boundaries.
- Keep current evidence, historical memory, task state, and model context
  labelled separately.
- Test both successful and adversarial cases, including missing and conflicting
  information.
- Preserve information-state and time boundaries to prevent outcome leakage.
- Record model, prompt/schema, retriever, builder, evaluator, policy, and tool
  versions for every run.
- Compare components against a simplest viable baseline before adding
  specialist agents or extra retrieval stages.
- Treat `ABSTAIN` as a valid quality outcome when the evidence or state is not
  sufficient.
- Never interpret evaluator PASS as trading, broker, live, formal-paper, or
  human-approval authorization.

## 2. Harness laboratory scenarios

Each scenario has a fixture, an oracle, a run trace, a structured output, an
evaluator result, and a reconstruction report.

| Scenario | Required challenge | Minimum oracle |
| --- | --- | --- |
| Context overload | Many items, only a small subset relevant | Relevant items selected; low-value items omitted or deferred with references |
| Contradiction hiding | Strong counter-evidence buried among irrelevant material | Counter-evidence retrieved and represented; no support-only PASS |
| Long-horizon continuity | Work spans multiple context windows | Objective, constraints, completed work, and next action survive each transition |
| Forced compaction | Important constraints occur before multiple compactions | Constraints, contradictions, references, and uncertainty remain recoverable |
| Fresh-session resume | Restart with zero conversational history | New session resumes the correct stage without repeating or inventing work |
| Memory contamination | Superficially similar prior case is irrelevant | Contaminating memory is rejected, penalized, or visibly qualified |
| Regime mismatch | Historical memory is from a materially different regime | Regime mismatch is surfaced and does not dominate current evidence |
| Missing data | Model is tempted to invent unavailable information | Output marks unknown/missing and evaluator refuses unsupported completion |
| Premature completion | A required research step remains incomplete | Completion is blocked or evaluator returns REVISE/ABSTAIN |
| Falsification | Obvious thesis is wrong and counter-evidence exists | Counter-thesis retrieval finds the invalidating evidence |
| Provenance reconstruction | Rebuild what the model saw | Package bytes or canonical references and assembly rules reproduce the view |
| Tool failure | Required source/tool fails midway | Failure is recorded; retry/alternative/ABSTAIN is explicit |
| Budget reduction | Context budget is reduced | Quality, omissions, abstentions, cost, and latency degradation are measured |

The lab should also include engineering-harness fixtures for a zero-history
checkout, conflicting roadmap documents, an incomplete worktree, a failed test,
an unrelated pre-existing failure, a missing verification command, and a clean
exit/handoff review.

## 3. Metrics

Metrics are reported by scenario, objective class, model route, budget, and
ablation. Do not collapse them into one score.

### Quality and safety

- **ObjectiveRetentionRate** — required objective elements preserved after each
  checkpoint/compaction/resume.
- **ConstraintRetentionRate** — hard constraints and non-authorizations retained.
- **RelevantEvidenceRecall** — oracle-relevant evidence selected or retrieved
  on demand.
- **CounterEvidenceRecall** — known contradictory/invalidating evidence selected.
- **SourceAttributionAccuracy** — claims linked to the correct source identity,
  timestamp, and information-state boundary.
- **UnsupportedClaimRate** — material claims lacking sufficient source support.
- **MemoryContaminationRate** — outputs materially influenced by irrelevant,
  stale, regime-mismatched, or outcome-leaking memory.
- **PrematureCompletionRate** — tasks marked complete while required work or
  evidence remains incomplete.
- **ContextReconstructionAccuracy** — ability to reproduce the model-visible
  package and its canonical components.
- **CompactionRecoveryRate** — successful continuation after forced compaction.
- **FreshSessionRecoveryRate** — successful continuation from durable state with
  zero transcript history.
- **EvaluatorRevisionRate** — share of producer outputs correctly sent to
  REVISE by an independent evaluator.
- **EvaluatorAbstentionRate** — share correctly abstained on insufficiency,
  contradiction, policy violation, or missing data.
- **ContradictionPreservationRate** — unresolved conflicts retained through
  checkpoints and output.
- **UnknownHonestyRate** — missing data represented as unknown rather than
  invented or overconfidently inferred.
- **PolicyViolationRate** — attempted or completed operation outside declared
  permissions or constraints.

### Efficiency and operations

- **TokensPerTask** — input, output, and tool-token usage by class.
- **LatencyPerTask** — wall-clock and per-stage latency.
- **CostPerTask** — provider/model/tool cost where available.
- **RetrievalVolume** — candidate, selected, deferred, and on-demand items.
- **UnnecessaryContextRate** — model-visible material not needed for the oracle.
- **CheckpointFrequency** and **ResumeAttempts**.
- **ToolFailureRecoveryRate** and retry amplification.
- **RevisionCount** and revision convergence.
- **TraceCompletenessRate** — required provenance fields present.
- **DuplicatePersistenceRate** — duplicate raw payloads or audit records created
  where stable references should have been used.
- **EngineeringVerificationPrecision** — selected commands protect the changed
  scope without needlessly running unrelated expensive suites.
- **EngineeringCleanExitRate** — completed tasks leave required diff, tests,
  status, handoff, and known-failure evidence.

## 4. Baseline and ablations

The laboratory must compare at least:

1. **Harness baseline** — bounded context, durable state, source references,
   deterministic budget, and independent evaluator where the scenario requires.
2. **Minus contradiction retrieval** — measure support-only failure and
   counter-evidence recall.
3. **Minus memory** — measure whether memory improves relevant recall without
   contamination.
4. **Minus evaluator** — measure unsupported claims and premature completion.
5. **Minus structured checkpointing** — measure resume and compaction recovery.
6. **Minus compaction** — measure long-horizon cost and failure under reduced
   context windows.
7. **Minus context budgeting** — measure overload, cost, and unnecessary context.
8. **Single-controller versus specialist orchestration** — only after the
   baseline is stable and only for tasks that justify the comparison.

An ablation is useful only when fixtures, model route, budget, and randomness
are controlled. A component is removable when its quality/safety benefit is
not reproducible or does not justify added latency, token cost, operational
complexity, and failure surface.

## 5. Evaluation gates

Future implementation packages must provide:

- a scenario fixture and oracle for every new contract;
- repeatable runs across at least two model/provider configurations where
  practical, without relying on provider-specific quirks;
- positive and negative cases for every safety or sufficiency gate;
- deterministic reconstruction evidence for important model invocations;
- metric results by ablation, not only an aggregate narrative; and
- external review of any claim that a component materially improves quality.

No threshold in this document authorizes PAPER-02, formal paper, live trading,
or Phase 13. Thresholds for a future implementation package must be proposed,
reviewed, and versioned with that package.

## 6. Provenance reconstruction protocol

For a selected historical run:

1. Load `HarnessRun`, task-state version, package ID/hash, and evaluator result.
2. Resolve selected evidence, counter-evidence, memory, tool-result, policy,
   schema, prompt, builder, retriever, and model references.
3. Verify hashes and information-state timestamps.
4. Reassemble the model-visible bytes using the recorded deterministic assembly
   algorithm, or compare to the retained immutable package bytes.
5. Compare the reconstruction hash to `ContextPackage.ContentHash`.
6. Report any missing, changed, unavailable, or nondeterministic component.

The result is `RECONSTRUCTED`, `PARTIAL`, or `FAILED`; a later source fetch must
not silently upgrade a failed historical reconstruction.

## 7. Falsification and evaluator review

Every meaningful hypothesis fixture declares its expected invalidation signals,
counter-evidence sources, and minimum evidence required to continue. The
evaluator must inspect causal links, source independence, unsupported claims,
uncertainty, confidence calibration, missing data, and premature completion.

The harness should measure whether the evaluator catches producer errors and
whether repeated REVISE loops converge. A critic that merely restates the
producer is not independent enough for acceptance.

## 8. Laboratory non-scope

HARNESS-00 created the laboratory specification only. HARNESS-02 adds the
minimum deterministic offline fixture coverage for the foundation selection
seam in `internal/modules/contextbuilder`; it does not implement HARNESS-07,
a harness runner, metrics collector, evaluator runtime, memory service,
production retriever, model route, multi-agent route, or database schema.

The current fixture suite has 53 named behavioral cases covering context
overload, contradiction hiding, missing data, budget reduction, duplicate versus
independent corroboration, temporal/future leakage, deterministic ordering and
hashing, memory/evidence separation, deferred references, build reports,
retriever failure, and PAPER-02 import isolation. These are foundation tests,
not laboratory quality claims or trading evidence.
