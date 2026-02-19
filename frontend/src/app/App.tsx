import { createBrowserRouter, RouterProvider } from 'react-router-dom';
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

export const routes = [
  {
    path: '/',
    element: <AppShell />,
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
      // @ts-expect-error v7_startTransition is valid at runtime but missing from v6.26.x type definitions
      v7_startTransition: true,
    },
  }
);

export default function App() {
  return <RouterProvider router={router} />;
}
