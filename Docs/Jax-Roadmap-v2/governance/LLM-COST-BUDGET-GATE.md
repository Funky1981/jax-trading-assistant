# Governance — LLM Cost and Budget Gate

## Purpose

Define minimum review controls for any Jax work package that introduces or changes a paid hosted-model execution path. This supplements existing GO/NO-GO governance and does not replace safety or trading/execution controls.

## Before paid execution

The handover/preflight must establish:

1. exact provider and requested model;
2. API endpoint/contract;
3. reasoning/tool/structured-output mode;
4. current official pricing checked immediately before execution;
5. configurable pricing values used by Jax;
6. conservative maximum token/request liability;
7. hard experiment/run ceiling;
8. credential presence without exposure;
9. expected request count/retry policy;
10. expected cache semantics;
11. evidence path;
12. mutation/safety state appropriate to the experiment;
13. explicit hosted-inference authorization gate.

If pricing is contradictory, missing or materially changed, stop for review.

## Pricing rule

Provider prices are execution-time facts, not permanent source-code truth. Record effective date/time, source/provider, model/tier, input/cached/cache-write/output rates, separate reasoning billing if any, batch/priority tier and currency.

## Budget rule

Every experimental hosted run requires a hard ceiling derived from frozen case/request count, actual/maximum prompt size, maximum output, permitted corrective retries, provider billing semantics and an ambiguity buffer.

Do not use a large generic ceiling merely because the account has available credit.

## Fail-closed conditions

Stop before or during paid inference when provider/model or execution shape differs from approval; pricing cannot be verified; required usage is missing/inconsistent; cost becomes too ambiguous; the hard budget would be exceeded; authorization is unsafe; credentials could be exposed; or provider behaviour materially changes the experiment.

## Post-run evidence

Report planned vs actual requests, retries, token categories, cache usage, exact/estimated cost, ambiguous liability, budget remaining, model/fingerprint identity, errors/timeouts, artifact hashes, credential-redaction proof and repository/safety state.

## Production extension

Before hosted inference becomes routine production research, additionally define task budgets, daily/monthly alerts/hard ceilings as appropriate, model routing/escalation policy, retention/privacy classification, cost telemetry/dashboarding and incident behaviour for provider pricing/availability changes.

## Benchmark exception

For capability benchmarks, application-level result caching is normally disabled so every case measures the model. Natural provider-side prompt caching may be observed and billed at the provider-reported rate; do not rewrite a frozen benchmark merely to manufacture cache hits.
