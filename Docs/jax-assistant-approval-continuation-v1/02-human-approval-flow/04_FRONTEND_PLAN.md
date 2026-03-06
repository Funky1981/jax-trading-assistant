# Frontend Plan

## New page
- `/approvals`

## Page sections
1. Approval queue table/cards
2. Candidate detail drawer
3. AI/explanation summary
4. Risk block summary
5. Approve / reject / snooze / reanalyze controls

## Existing page integration
- `TradingPage`
  - show pending approvals count
  - quick link to candidate details
- future `AnalysisPage`
  - show approval decision in timeline

## Nav updates needed
Modify:
- `frontend/src/app/App.tsx`
- `frontend/src/components/layout/AppShell.tsx`

Add nav item:
- `Approvals`
