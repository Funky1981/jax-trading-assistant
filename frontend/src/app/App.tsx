import { createBrowserRouter, Navigate, RouterProvider, useLocation } from 'react-router-dom';
import { AppShell } from '@/components/layout/AppShell';
import { HomePage } from '@/pages/HomePage';
import { AiTradingPage } from '@/pages/AiTradingPage';
import { DashboardPage } from '@/pages/DashboardPage';
import { TradingPage } from '@/pages/TradingPage';
import { SystemPage } from '@/pages/SystemPage';
import { ResearchPage } from '@/pages/ResearchPage';
import { AnalysisPage } from '@/pages/AnalysisPage';
import { TestingPage } from '@/pages/TestingPage';
import { E2ETestingPage } from '@/pages/E2ETestingPage';
import { BlotterPage } from '@/pages/BlotterPage';
import { PortfolioPage } from '@/pages/PortfolioPage';
import { OrderTicketPage } from '@/pages/OrderTicketPage';
import { SettingsPage } from '@/pages/SettingsPage';
import { UserGuidePage } from '@/pages/UserGuidePage';
import { ApprovalsPage } from '@/pages/ApprovalsPage';
import { AssistantPage } from '@/pages/AssistantPage';
import { LoginPage } from '@/pages/LoginPage';
import { AuthProvider, useAuth } from '@/contexts/AuthContext';
import { BeginnerUXProvider } from '@/context/BeginnerUXContext';
import { ETFUniversePage } from '@/pages/ETFUniversePage';
import { StrategyCardsPage } from '@/pages/StrategyCardsPage';
import { ResearchTimelinePage } from '@/pages/ResearchTimelinePage';
import { TradingModesPage } from '@/pages/TradingModesPage';
import { CandidateEvidencePage } from '@/pages/CandidateEvidencePage';
import { TradingModulesPage } from '@/pages/TradingModulesPage';
import { EquityAlphaGuidePage } from '@/pages/EquityAlphaGuidePage';
import { ETFGuidePage } from '@/pages/ETFGuidePage';
import { PaperTradingTestPlanPage } from '@/pages/PaperTradingTestPlanPage';
import { MobileApprovalHarnessPage } from '@/pages/MobileApprovalHarnessPage';
import { NotificationCentrePage } from '@/pages/NotificationCentrePage';
import { MacroEventsPage } from '@/pages/MacroEventsPage';
import { MonitorInboxPage } from '@/pages/MonitorInboxPage';
import { OutcomesPage } from '@/pages/OutcomesPage';

function ProtectedRoute({ children }: { children: React.ReactNode }) {
  const { isAuthenticated, isLoading, authRequired } = useAuth();
  const location = useLocation();

  if (isLoading) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-background">
        <p className="text-muted-foreground text-sm">Loading...</p>
      </div>
    );
  }

  if (authRequired && !isAuthenticated) {
    return <Navigate to="/login" state={{ from: location.pathname }} replace />;
  }

  return <>{children}</>;
}

export const routes = [
  {
    path: '/login',
    element: <LoginPage />,
  },
  {
    path: '/',
    element: (
      <ProtectedRoute>
        <BeginnerUXProvider>
          <AppShell />
        </BeginnerUXProvider>
      </ProtectedRoute>
    ),
    children: [
      { index: true, element: <HomePage /> },
      { path: 'dashboard', element: <DashboardPage /> },
      { path: 'ai-trading', element: <AiTradingPage /> },
      { path: 'manual-trading', element: <TradingPage /> },
      { path: 'swing-trading', element: <TradingModesPage /> },
      { path: 'notifications', element: <NotificationCentrePage /> },
      { path: 'macro/events', element: <MacroEventsPage /> },
      { path: 'monitor/inbox', element: <MonitorInboxPage /> },
      { path: 'outcomes', element: <OutcomesPage /> },
      { path: 'modules', element: <TradingModulesPage /> },

      { path: 'trading', element: <Navigate to="/equity-alpha/trading" replace /> },
      { path: 'order-ticket', element: <OrderTicketPage /> },
      { path: 'approvals', element: <Navigate to="/etf/approvals" replace /> },
      { path: 'etf-universe', element: <Navigate to="/etf/universe" replace /> },
      { path: 'strategies', element: <Navigate to="/etf/strategies" replace /> },
      { path: 'timeline', element: <Navigate to="/etf/timeline" replace /> },
      { path: 'trading-modes', element: <Navigate to="/etf/trading-modes" replace /> },

      { path: 'legacy/trading', element: <Navigate to="/equity-alpha/trading" replace /> },
      {
        path: 'legacy/order-ticket',
        element: <Navigate to="/equity-alpha/order-ticket" replace />,
      },
      { path: 'legacy/guide', element: <Navigate to="/equity-alpha/guide" replace /> },

      { path: 'equity-alpha/trading', element: <TradingPage /> },
      { path: 'equity-alpha/order-ticket', element: <OrderTicketPage /> },
      { path: 'equity-alpha/guide', element: <EquityAlphaGuidePage /> },
      { path: 'equity-alpha/strategies', element: <StrategyCardsPage /> },
      { path: 'equity-alpha/timeline', element: <ResearchTimelinePage /> },
      { path: 'equity-alpha/trading-modes', element: <TradingModesPage /> },
      { path: 'equity-alpha/candidates/:candidateId/evidence', element: <CandidateEvidencePage /> },

      { path: 'etf/trading', element: <TradingPage /> },
      { path: 'etf/guide', element: <ETFGuidePage /> },
      { path: 'etf/approvals', element: <ApprovalsPage /> },
      { path: 'etf/universe', element: <ETFUniversePage /> },
      { path: 'etf/strategies', element: <StrategyCardsPage /> },
      { path: 'etf/timeline', element: <ResearchTimelinePage /> },
      { path: 'etf/trading-modes', element: <TradingModesPage /> },
      { path: 'etf/candidates/:candidateId/evidence', element: <CandidateEvidencePage /> },

      { path: 'system', element: <SystemPage /> },
      { path: 'research', element: <ResearchPage /> },
      { path: 'analysis', element: <AnalysisPage /> },
      { path: 'testing', element: <TestingPage /> },
      { path: 'testing/plan', element: <PaperTradingTestPlanPage /> },
      { path: 'testing/mobile-approval-harness', element: <MobileApprovalHarnessPage /> },
      { path: 'blotter', element: <BlotterPage /> },
      { path: 'portfolio', element: <PortfolioPage /> },
      { path: 'settings', element: <SettingsPage /> },
      { path: 'e2e-tests', element: <E2ETestingPage /> },
      { path: 'guide', element: <UserGuidePage /> },
      { path: 'assistant', element: <AssistantPage /> },
      { path: 'candidates/:candidateId/evidence', element: <CandidateEvidencePage /> },
    ],
  },
];

export const router = createBrowserRouter(routes, {
  basename: import.meta.env.BASE_URL,
  future: {
    v7_relativeSplatPath: true,
  },
});

export default function App() {
  return (
    <AuthProvider>
      <RouterProvider
        router={router}
        future={{
          v7_startTransition: true,
        }}
      />
    </AuthProvider>
  );
}
