import { createBrowserRouter, Navigate, RouterProvider, useLocation } from 'react-router-dom';
import { AppShell } from '@/components/layout/AppShell';
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
import { LoginPage } from '@/pages/LoginPage';
import { AuthProvider, useAuth } from '@/contexts/AuthContext';

// ── Protected route ────────────────────────────────────────────────────────────
// If auth is required and the user has no valid token, redirect to /login.
// If auth is disabled (dev mode), renders children immediately.

function ProtectedRoute({ children }: { children: React.ReactNode }) {
  const { isAuthenticated, isLoading, authRequired } = useAuth();
  const location = useLocation();

  if (isLoading) {
    // Spinner while /auth/status is resolving - keeps layout stable
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
        <AppShell />
      </ProtectedRoute>
    ),
    children: [
      { index: true, element: <DashboardPage /> },
      { path: 'trading', element: <TradingPage /> },
      { path: 'order-ticket', element: <OrderTicketPage /> },
      { path: 'system', element: <SystemPage /> },
      { path: 'research', element: <ResearchPage /> },
      { path: 'analysis', element: <AnalysisPage /> },
      { path: 'testing', element: <TestingPage /> },
      { path: 'blotter', element: <BlotterPage /> },
      { path: 'portfolio', element: <PortfolioPage /> },
      { path: 'settings', element: <SettingsPage /> },
      { path: 'e2e-tests', element: <E2ETestingPage /> },
      { path: 'guide', element: <UserGuidePage /> },
    ],
  },
];

export const router = createBrowserRouter(
  routes,
  {
    basename: import.meta.env.BASE_URL,
    future: {
      v7_relativeSplatPath: true,
    },
  }
);

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

