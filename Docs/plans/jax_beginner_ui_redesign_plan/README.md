# Jax Beginner-First UI Redesign Plan

## Purpose

This plan turns the current Jax frontend from a feature-heavy developer console into a beginner-friendly operator product.

The redesign focuses only on the workflow currently being proven:

**Home → Guide → Evidence Inbox → Candidate Review → Outcomes → System Safety**

All other pages remain available but move into a collapsed **Review** section until there is a real need to redesign and promote them.

## Implementation order

1. `00_PRODUCT_DIRECTION.md`
2. `01_PHASE_NAV_HOME_GUIDE.md`
3. `02_PHASE_EVIDENCE_INBOX.md`
4. `03_PHASE_CANDIDATE_OUTCOMES.md`
5. `04_PHASE_SYSTEM_SAFETY_AND_HELP.md`
6. `05_REVIEW_SECTION_AND_PAGE_PROMOTION.md`
7. `06_TESTING_AND_ACCEPTANCE.md`
8. `07_DELIVERY_RULES.md`

## Core rule

Complete, test and review one phase before starting the next.

Do not attempt a whole-application visual rewrite. Existing routes should continue to resolve while the operator-facing workflow is simplified incrementally.

## Current design problem

The existing navigation exposes too many unrelated features at the same priority. A beginner must understand Jax's internal architecture before knowing what to do.

The new design must answer these questions immediately:

- Is Jax safe?
- What has Jax received?
- What did Jax do with it?
- Does anything need review?
- Was a candidate created?
- What happened to the hypothetical paper plan?
- Where can I get help?

## Repository material to inspect

Before implementing Phase 1, review:

- `frontend/src/components/layout/AppShell.tsx`
- `frontend/src/app/App.tsx`
- `frontend/src/pages/HomePage.tsx`
- `frontend/src/pages/UserGuidePage.tsx`
- `frontend/src/pages/MonitorInboxPage.tsx`
- `frontend/src/pages/CandidateEvidencePage.tsx`
- `frontend/src/pages/SystemPage.tsx`
- `frontend/src/context/BeginnerUXContext.tsx`
- `frontend/src/components/trading/BeginnerModeToggle.tsx`
- `frontend/src/components/ui/help-hint.tsx`
- every screenshot in `images/`

Use the screenshots as the visual baseline for identifying density, overflow, unclear hierarchy and developer-oriented terminology.

## Definition of success

A first-time user should be able to:

1. Open Home and understand that Jax is paper-safe.
2. Follow a short Guide without knowing trading-system terminology.
3. Open Evidence Inbox and understand one genuine event.
4. See whether Jax created a candidate.
5. Understand that outcomes are hypothetical and not fills.
6. Confirm execution and live trading are disabled.
7. Find help without leaving the page.
