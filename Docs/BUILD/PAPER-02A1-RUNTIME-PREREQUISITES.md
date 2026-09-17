# PAPER-02A1 — Runtime Prerequisite Closure

Status: **IMPLEMENTED / EXTERNAL REVIEW REQUIRED**.

PAPER-02 remains **READY_FOR_EXTERNAL_REVIEW / NOT ACTIVE**. This package does
not create a pilot identity, admit an opportunity, replay an event, or activate
the pilot.

## Why activation stopped

The activation preflight could not resolve four real-runtime prerequisites:

1. `PAPER_SESSION_CALENDAR_JSON` was absent.
2. Genuine prospective event intake was disabled and had no configured endpoint.
3. The configured PAPER symbols had no versioned eligible-universe identity.
4. Risk and entry identities existed only at lower contract boundaries, not as
   freezeable pilot-level identities.

## Implemented closure

- `config/paper-02/session-calendar-us-equities-2026-v2.json` is a bounded,
  timezone-aware `US_EQUITIES_REGULAR` calendar with explicit coverage,
  weekends, holidays, DST-aware `America/New_York` sessions, and an early-close
  window. Coverage outside the manifest fails closed.
- `config/paper-02/eligible-universe-us-equities-2026-v1.json` is derived from
  the PAPER Compose symbol set and gives each supported instrument an explicit
  exchange identity, effective date, PAPER applicability, and deterministic
  content hash.
- `config/paper-02/entry-policy-v1.json` binds the existing candidate,
  evidence, risk, human-approval, thesis, PAPER-only, and simulated-venue
  requirements without creating a second entry engine.
- The risk identity is derived at runtime from `config/risk-constraints.json`
  and the configured PAPER position percentage; its version is the repository's
  loaded policy version and its hash is a canonical SHA-256 content hash.
- `AssessPaper02Readiness` is a pure, read-only contract. The protected
  `/api/v1/exploratory-paper/pilot-readiness` route exposes READY/BLOCKED state
  and blocking reasons; it has no activation side effect.
- Pilot identity payloads and the Postgres schema can retain eligible-universe,
  risk-policy, and entry-policy hashes. The new migration protects those
  fields as immutable identity.

The resolved checked-in identities are:

- Calendar: `us-equities-regular-2026-09-to-12-v2`, hash
  `sha256:e3cd198d1b80b267999e6250ad4b1440d872274cce5299e90aa2b31176c7071b`.
- Eligible universe: `paper-02-us-equities-2026-v1`, hash
  `sha256:52d98a46bdd7fc8cc90de179ca0e51f1878517ce4d47b34b756e02a818a3f569`.
- Risk: version `v269d7a37cb3e`, hash
  `sha256:3979fe3455aa129166bd38d242ff1f1956f2024c317004000a2575ab6f22bba6`.
- Entry: `paper-02-entry-policy-v1`, hash
  `sha256:e440471e04426e14210fabeb6184389ac90f7235e4ce2773a7bbfb6ed729c8c8`.

## Current readiness

The checked-in runtime configuration is safety-compatible, but the default
World Monitor pull is disabled. Therefore the current honest readiness result
is **BLOCKED** until an operator configures a genuine prospective source using
`WORLD_MONITOR_PULL_ENABLED=true` and an absolute `WORLD_MONITOR_EVENTS_URL`
(plus any source credentials required by that external service). No endpoint,
credential, or historical replay is invented by this package.

The event source identity is `jax-genuine-event-pull-v1`. When enabled, the
existing World Monitor pull worker supplies source/event identity, first-seen
and ingestion timestamps, provenance, persistence, candidate compatibility,
and duplicate delivery handling. When disabled or malformed, readiness fails
closed.

## Safety boundary

Runtime readiness requires `PAPER`, `ExecutionAuthority=NONE`,
`BrokerExecutionAllowed=false`, and leverage in `(0, 1]`. The PAPER-01/PAPER-02
path remains connected only to the isolated simulated paper venue. No broker,
IB, live order, live fill, real position, historical validation, strategy
tuning, formal evidence, Phase 13 work, pilot observation, or PAPER-02
activation occurred.

## Remaining external gate

External review must confirm the bounded calendar/universe/policy identities
and provide/configure a genuine prospective event source before activation.
PAPER-02 stays **READY_FOR_EXTERNAL_REVIEW / NOT ACTIVE**, with sample `0 / 50`.

## Calendar correction

External review identified that October 12 and November 11 had incorrectly been
classified as equity-market holidays. The corrected v2 manifest models exchange
sessions rather than generic federal/bank holidays: both dates are regular open
sessions; November 26 is closed for Thanksgiving; and November 27 is a
09:30–13:00 America/New_York early-close session.

The correction was cross-checked against the official [NYSE 2026 trading
calendar](https://www.nyse.com/publicdocs/nyse/ICE_NYSE_2026_Yearly_Trading_Calendar.pdf),
the [NYSE hours and holidays page](https://www.nyse.com/markets/hours-calendars),
and the [Nasdaq 2026 equity holiday schedule](https://nasdaqtrader.com/Trader.aspx?id=Calendar).

Readiness now requires the proposed maximum pilot deadline to remain within
calendar coverage: `PilotStartTimestamp + MaximumDurationDays <= CalendarCoverageEnd`.
For the 90-day PAPER-02 maximum, a proposed start that would exceed the frozen
coverage end fails closed before activation.
