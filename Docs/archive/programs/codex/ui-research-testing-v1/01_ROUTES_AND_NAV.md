# Step 1: Add Routes and Navigation

## Files to modify
- `frontend/src/app/App.tsx`
- `frontend/src/components/layout/AppShell.tsx`

## App.tsx (routing)
Add imports:
- `ResearchPage` from `@/pages/ResearchPage`
- `AnalysisPage` from `@/pages/AnalysisPage`
- `TestingPage` from `@/pages/TestingPage`

Add routes under AppShell children:
- `{ path: 'research', element: <ResearchPage /> }`
- `{ path: 'analysis', element: <AnalysisPage /> }`
- `{ path: 'testing', element: <TestingPage /> }`

Remove placeholders only if you are ready; otherwise keep them and just add new routes.

## AppShell.tsx (sidebar nav)
Add navItems:
- `{ label: 'Research', path: '/research', icon: <choose icon> }`
- `{ label: 'Analysis', path: '/analysis', icon: <choose icon> }`
- `{ label: 'Testing', path: '/testing', icon: <choose icon> }`

Recommended lucide icons:
- Research: `FlaskConical` or `BookOpen`
- Analysis: `BarChart3`
- Testing: `ShieldCheck` or `CheckCircle2`

Acceptance criteria:
- Clicking sidebar items routes correctly.
- Pages render without console errors.
