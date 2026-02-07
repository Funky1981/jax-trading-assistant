import { HashRouter, Route, Routes } from 'react-router-dom';
import { AppShell } from '@/components/layout/AppShell';
import { DashboardPage } from '@/pages/DashboardPage';
import { TradingPage } from '@/pages/TradingPage';
import { SystemPage } from '@/pages/SystemPage';

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

export function AppRoutes() {
  return (
    <Routes>
      <Route element={<AppShell />}>
        <Route key="dashboard" path="/" element={<DashboardPage />} />
        <Route key="trading" path="/trading" element={<TradingPage />} />
        <Route key="system" path="/system" element={<SystemPage />} />
        <Route key="blotter" path="/blotter" element={<PlaceholderPage title="Trade Blotter" />} />
        <Route key="portfolio" path="/portfolio" element={<PlaceholderPage title="Portfolio" />} />
        <Route key="settings" path="/settings" element={<PlaceholderPage title="Settings" />} />
      </Route>
    </Routes>
  );
}

export default function App() {
  return (
    <HashRouter
      future={{
        v7_startTransition: true,
        v7_relativeSplatPath: true,
      }}
    >
      <AppRoutes />
    </HashRouter>
  );
}
