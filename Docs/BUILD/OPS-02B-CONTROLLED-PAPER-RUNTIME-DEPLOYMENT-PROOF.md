# OPS-02B — Controlled PAPER Runtime Deployment Proof

## Status

OPS-02B is **IMPLEMENTED / CONTROLLED PROOF COMPLETE / EXTERNAL REVIEW
REQUIRED** on top of reviewed SHA
`2635d1a180827d9ae6ec09911781ad037b88f156`.

This record includes the narrowly authorised OPS-02B3 disablement of twelve
historical World Monitor integration-test strategy fixtures. The fixtures were
disabled, not deleted, and their dependent evidence was preserved.

PAPER-02-2026-01 remains `ABORTED / HISTORICALLY PRESERVED`, with one incident,
zero genuine prospective opportunities, and formal evidence disabled. No new
pilot, strategy validation, broker execution, live execution, or
`FORMAL_FORWARD_PAPER` work was performed.

## Operator authorisation and fixture boundary

The operator explicitly authorised one database mutation: set `enabled=false`
only for the twelve currently-enabled `strategy_instances` rows whose names
started with `wm-scanner-test-`, `wm-promoter-test-`, or
`wm-chart-block-test-`.

Before mutation the database proved:

- 19 total strategy instances;
- 12 enabled and 7 disabled;
- all 12 enabled rows matched an authorised prefix;
- no enabled row existed outside the authorised fixture set;
- all 12 used `etf_news_sector_momentum_v1`.

The mutation used one explicit transaction and committed exactly twelve
updates. No rows were deleted, and no dependent evidence was cleaned up.

The twelve preserved fixture rows were:

| ID | Name | Signals | Candidates | Candidate approvals | Trade approvals | Paper tickets | Execution instructions |
|---|---|---:|---:|---:|---:|---:|---:|
| `a2038d08-2d14-4399-bc0d-f4789c39c47d` | `wm-chart-block-test-0c7d85e6-49de-400d-9326-a946f3758ce5` | 0 | 0 | 0 | 0 | 0 | 0 |
| `3e4778fa-0b2c-4168-9a26-d492c454f076` | `wm-chart-block-test-0d766a40-5999-41e4-964d-719e837bcd13` | 0 | 0 | 0 | 0 | 0 | 0 |
| `f35ebdd0-14ab-472a-b134-8bd2b081d9a3` | `wm-chart-block-test-53a357c6-4dfc-43eb-9a8f-afab358dd8ff` | 3 | 2 | 0 | 0 | 0 | 0 |
| `e03bc380-a292-404a-b07a-637ee16f56f8` | `wm-chart-block-test-dde78447-fd8f-49d2-8793-6f46bc4cf7ab` | 0 | 0 | 0 | 0 | 0 | 0 |
| `59441827-2a84-4d03-9df8-d96d3799e83b` | `wm-chart-block-test-dede9529-c348-48f0-85b8-0d36ba864e40` | 0 | 0 | 0 | 0 | 0 | 0 |
| `4905e227-f6c2-4f64-a6d5-9559c4f31395` | `wm-chart-block-test-f393b146-a488-4d65-8a49-3adf0a55f57f` | 1 | 0 | 0 | 0 | 0 | 0 |
| `00b974a6-d337-4bff-80c3-09415dc523a7` | `wm-promoter-test-108872ec-e8be-455f-9eda-e20032013614` | 9 | 9 | 0 | 0 | 0 | 0 |
| `931ac0ce-f2d2-4f91-a587-268489b38b28` | `wm-promoter-test-35e7874d-9fdd-4650-a215-36761d138131` | 1 | 0 | 0 | 0 | 0 | 0 |
| `a066e9ff-88bf-4bd0-a5b9-7e9c025e9c10` | `wm-promoter-test-4eff6b8e-7743-402c-bed8-70dced433c27` | 0 | 0 | 0 | 0 | 0 | 0 |
| `91f58544-82bc-430d-87f6-4248b8ecefa8` | `wm-promoter-test-b856a415-34c6-41e3-9388-349591df1bdc` | 6 | 4 | 1 | 1 | 1 | 0 |
| `a396344c-aacb-4b70-8063-b0475471092e` | `wm-scanner-test-33d57ab9-a640-4175-b3ae-d4319795a2c3` | 0 | 0 | 0 | 0 | 0 | 0 |
| `8b9e90e3-c257-4405-ace8-2d8cb77fd6b7` | `wm-scanner-test-ccb5a236-768e-42cf-a13c-3436f9b63f6c` | 2 | 2 | 0 | 0 | 0 | 0 |

After mutation:

- 19 total strategy instances;
- 0 enabled and 19 disabled;
- all twelve fixture rows still existed, with their original type;
- all dependent counts above were unchanged;
- all five repository-managed strategy configuration rows remained disabled.

The five repository-managed JSON instances were not modified. The existing
loader may idempotently upsert them and update their timestamps; their
enablement and configuration remained unchanged.

## Provider gate result

No Polygon credential was supplied or added. After fixture disablement, the
normal PAPER startup query found zero enabled event-dependent strategy
instances. The existing provider gate passed naturally; it was not weakened or
bypassed.

## Exact runtime image and configuration

The trader image was freshly built from the reviewed checkout with:

- runtime SHA: `2635d1a180827d9ae6ec09911781ad037b88f156`;
- build time: `2026-09-25T14:52:30Z`;
- image ID: `sha256:03801d5ca26238ab927dcd3d27732fcc842b53ddba577252bd5b12ca8794a64c`.

The bounded run used:

- `JAX_RUNTIME_MODE=paper`;
- `JAX_TRADER_RUNTIME_MODE=paper`;
- `PAPER_ACCOUNT_ID=jax-paper-runtime-v1`;
- `ALLOW_LIVE_TRADING=false`;
- `BROKER_EXECUTION_ALLOWED=false`;
- `EXECUTION_ENABLED=false`;
- `EXECUTION_INSTRUCTION_WORKER_ENABLED=false`;
- `MAX_LEVERAGE=1`;
- all four process worker gates set to `false`;
- `WORLD_MONITOR_PULL_ENABLED=true`;
- the validated internal World Monitor endpoint;
- empty auth bootstrap username and password.

The runtime was started through the no-dependency Compose boundary. No
`ib-bridge`, frontend, research runtime, or additional database service was
started.

## Startup and worker proof

Startup logged the reviewed runtime SHA and passed without `POLYGON_API_KEY`.

`GET /health` returned `healthy`.

`GET /ready` returned `ready=true` with:

- runtime mode `paper`;
- execution disabled;
- broker `required=false`, `skipped=true`;
- opportunity scanner `false`;
- trade watcher `false`;
- market ingester `false`;
- mobile notification dispatcher `false`.

The exploratory PAPER entry and review workers both logged successful cycles.
No startup failure, account restore failure, invalid paper artifact, database
failure, or safety-policy failure was observed.

## Genuine World Monitor proof

Before runtime start:

- cursor: `17927`;
- pull pages: 13;
- inbox rows: 358;
- event decisions: 765.

The runtime completed three genuine pull cycles. Each cycle reported:

- fetched: 100;
- ingested: 100;
- duplicates: 0;
- decisions created: 100;
- decisions reused: 0;
- failures: 0.

After shutdown:

- cursor: `39394`;
- pull pages: 16;
- inbox rows: 658;
- event decisions: 1065.

One durable trace was:

`wm_680e77ccbf20eb1ced573234ed8d0519758208f7b3d936fee8513ae27ddde091`
(`cnbc-top-news`, native provider ID `108365360`, sequence `18186`)
→ World Monitor pull/raw page
→ inbox row `49422b16-5445-4db9-b03b-e8b8fcafbeb4`
→ normalized event `64ead050-b15e-4ff9-a964-23045124f092`
→ decision `a65aefc0-2619-4a90-a9ca-033c399a4501`.

The decision was `NO_TRADE`, with `decision_origin=live_origin` and
`decision_context=continuous_world_monitor_ingestion`.

## Economic-write reconciliation

Before and after runtime counts were unchanged:

- candidates: 21;
- candidate approvals: 2;
- trade approvals: 2;
- execution instructions: 1;
- trades: 0;
- `auth_users`: 1;
- `jax-paper-runtime-v1` account rows: 0;
- account-scoped paper orders: 0;
- account-scoped paper fills: 0;
- account-scoped paper ledger events: 0;
- exploratory lifecycles: 16;
- outcomes: 6.

The historical global paper totals remained 22 orders, 22 fills, and 22 ledger
events. No new economic row was created.

Unexpected economic writes: **0**.

Operational writes were limited to genuine World Monitor pull pages, cursor,
inbox, and decision state, plus the existing idempotent strategy bootstrap
upserts. No historical fixture evidence was deleted or altered.

## PAPER-02 preservation and shutdown

`PAPER-02-2026-01` remained:

- status: `ABORTED`;
- formal evidence eligible: `false`;
- incidents: 1;
- genuine prospective opportunities: 0.

The separate `pilot-restart-*` rows were not touched.

The one-off runtime received a shutdown signal and logged graceful shutdown of
both jax-trader and the frontend API server. The container exited and was
removed. No application runtime remained listening on ports 8100 or 8081.

## Validation and package boundary

The local validation performed for this documentation record passed:

- `go test ./...`;
- `golangci-lint run ./...`;
- `go list -mod=readonly -m all`;
- `docker compose config --quiet`;
- Golden Tests;
- Import Boundary Enforcement checks, including trader forbidden-import,
  research compilation, and import-cycle checks;
- isolated disposable PostgreSQL migrations and required integration packages;
- required frontend signal integration tests;
- required OPS-01 PostgreSQL proof;
- Linux `-race` execution for harness/lifecycle packages and the OPS-01 proof
  using Go `1.26.8`.

No Go files changed in this package. A repository-wide `gofmt -l` scan reports
pre-existing unrelated formatting drift in existing Go files; no unrelated
files were reformatted. `git diff --check` passed. Exact-final-SHA workflow
evidence is recorded in the final handover.

This document does not claim strategy validation, profitability, trading edge,
formal forward-paper readiness, broker execution, or live trading
authorization.

Package boundary: approved twelve-row fixture disablement plus bounded OPS-02B
PAPER runtime proof only. No next-stage work was executed.
