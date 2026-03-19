# Frontend Completion Plan

## Current confirmed state
Routed:
- `/`
- `/trading`
- `/system`
- login
- placeholders for portfolio / blotter / settings

## Required new pages
- `ResearchPage.tsx`
- `AnalysisPage.tsx`
- `TestingPage.tsx`
- `ApprovalsPage.tsx`
- `AssistantPage.tsx`

## Required supporting components
- candidate trade table / cards
- approval queue cards
- gate dashboard cards
- assistant chat panel
- event stream hook
- strategy instance forms
- run detail / timeline viewer

## Required hooks
- `useStrategyInstances`
- `useCandidates`
- `useApprovals`
- `useGateStatus`
- `useChat`
- `useEventStream`

## Required app-shell updates
Add nav items:
- Research
- Analysis
- Testing
- Approvals
- Assistant
