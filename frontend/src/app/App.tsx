import { createBrowserRouter, Navigate, RouterProvider, useLocation } from 'react-router-dom';
import { AppShell } from '@/components/layout/AppShell';
import { DashboardPage } from '@/pages/DashboardPage';
import { TradingPage } from '@/pages/TradingPage';
import { SystemPage } from '@/pages/SystemPage';
import { LoginPage } from '@/pages/LoginPage';
import { AuthProvider, useAuth } from '@/contexts/AuthContext';

// Placeholder pages - to be implemented
function PlaceholderPage({ title }: { title: string }) {
  return (
    <div className="flex items-center justify-center h-64">
      <div className="text-center">
        <h2 className="text-xl font-semibold mb-2">{title}</h2>
        <p className="text-muted-foreground">Coming soon...</p>
      </div>
    </div>
  );
}

// ── Protected route ────────────────────────────────────────────────────────────
// If auth is required and the user has no valid token, redirect to /login.
// If auth is disabled (dev mode), renders children immediately.

function ProtectedRoute({ children }: { children: React.ReactNode }) {
  const { isAuthenticated, isLoading, authRequired } = useAuth();
  const location = useLocation();

  if (isLoading) {
    // Spinner while /auth/status is resolving — keeps layout stable
    return (
      <div className="min-h-screen flex items-center justify-center bg-background">
        <p className="text-muted-foreground text-sm">Loading…</p>
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
      { path: 'system', element: <SystemPage /> },
      { path: 'blotter', element: <PlaceholderPage title="Trade Blotter" /> },
      { path: 'portfolio', element: <PlaceholderPage title="Portfolio" /> },
      { path: 'settings', element: <PlaceholderPage title="Settings" /> },
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
      <RouterProvider router={router} />
    </AuthProvider>
  );
}
