# PAPER-01A Runtime Audit

Date: 2026-09-17
Scope: repository-backed audit of the PAPER-01 exploratory layer. This audit
does not authorize PAPER-02 or a prospective pilot.

## Findings

1. **TradeThesis creation**

   `TradeThesis` is defined and validated in
   `internal/modules/exploratorypaper/contracts.go`. Repository construction is
   presently in test fixtures: `fixtureThesis` in
   `internal/modules/exploratorypaper/contracts_test.go` and the integration
   proof in `internal/modules/exploratorypaper/integration_test.go`. No runtime
   caller constructs `exploratorypaper.TradeThesis`.

2. **TradeThesis durability**

   No exploratorypaper repository, SQL store, migration, or runtime persistence
   adapter was found. The thesis is not durably persisted.

3. **EntryBinding creation and durability**

   `EntryBinding` is defined and validated in
   `internal/modules/exploratorypaper/contracts.go`. It is created only by the
   exploratorypaper tests and integration proof. No runtime creator, SQL store,
   migration, or durable adapter was found.

4. **Exploratory Position persistence**

   `Position` is defined in `contracts.go` and created through
   `OpenApprovedPosition`. It is held by the caller in memory; there is no
   exploratorypaper position store or migration.

5. **Thesis transitions**

   `Position.Reassess` appends `Transition` values to the in-memory
   `Position.Transitions` slice. No runtime event sink or durable transition
   store is wired to this contract.

6. **Scheduled review orchestration**

   `BuildReviewSchedule` in
   `internal/modules/exploratorypaper/lifecycle.go` calculates a bounded
   one-review-per-trading-session schedule. Repository search found no runtime
   scheduler or job caller for this exploratory schedule.

7. **Outcome creation and persistence**

   `Outcome` and `Outcome.Validate` are defined in
   `internal/modules/exploratorypaper/outcomes.go`. PAPER-01A's integration
   proof constructs and validates an outcome in memory. No exploratorypaper
   outcome repository, migration, or runtime persistence path was found.

8. **API and operator UI**

   Existing workflow, approval, paper-ticket, paper-venue, and outcome read
   models are exposed through the existing trader/operator surfaces. The
   `exploratorypaper` types themselves are not currently exposed by a dedicated
   API route or UI screen. Existing `frontend/src/pages/OutcomesPage.tsx` and
   the operator evidence handlers expose existing persisted paper/read-model
   data, not durable `TradeThesis`, `EntryBinding`, exploratory `Position`, or
   exploratory `Outcome` records.

9. **Domain-contract/test-only boundary**

   The new event-driven exploratory layer is presently a domain-contract and
   test proof. The existing `internal/modules/workflow` store, paper venue, and
   paper ledger are separate in-memory domains with snapshot/restore helpers:
   `workflow.Store.ExportSnapshot` / `RestoreSnapshot`,
   `papertrading.PaperVenue.Snapshot` / `RestorePaperVenue`, and
   `papertrading.PaperLedger.Snapshot` / `RestorePaperLedger`. Those helpers do
   not persist the exploratorypaper records or connect them to a restartable
   runtime service.

10. **Process-restart conclusion**

    A prospective PAPER-02 pilot could not currently survive a process restart
    with complete thesis, approval-binding, exploratory-position, transition,
    and outcome provenance. The workflow, simulated venue, and ledger have
    independently testable snapshot/restore contracts, but the exploratory
    records and their cross-domain identity chain are not durably stored and
    restored as one unit.

11. **Implementation still required before a prospective pilot**

    Before any future pilot authorization, the repository would need a reviewed
    persistence and restart design for the exploratory thesis, binding,
    position, transitions, and outcomes; an atomic link to the existing approved
    workflow and paper-intent records; scheduled review orchestration; operator
    read models/API coverage; and restart/reconciliation evidence. That work is
    outside PAPER-01A. No large persistence or orchestration subsystem is
    introduced by this remediation.

## Audit conclusion

PAPER-01A hardens the in-memory approval boundary and makes the current runtime
integration limitation explicit. It does not claim that PAPER-01 is a durable
prospective pilot, and it does not authorize PAPER-02.
