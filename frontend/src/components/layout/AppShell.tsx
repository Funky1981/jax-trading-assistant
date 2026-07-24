import { useEffect, useMemo, useState } from 'react';
import { NavLink, Outlet, useLocation } from 'react-router-dom';
import {
  Activity,
  BarChart3,
  Bell,
  BookOpen,
  Bot,
  Briefcase,
  CheckSquare,
  ChevronDown,
  ChevronRight,
  ClipboardPenLine,
  FileText,
  FlaskConical,
  Globe,
  Inbox,
  LayoutDashboard,
  LogOut,
  Menu,
  Server,
  Settings,
  ShieldCheck,
  TrendingUp,
  X,
} from 'lucide-react';
import { cn } from '@/lib/utils';
import { Button } from '@/components/ui/button';
import JaxLogo from '@/images/jax_ai_trader.svg';
import { useAuth } from '@/contexts/AuthContext';
import { BeginnerModeToggle } from '@/components/trading/BeginnerModeToggle';

const primaryNavItems = [
  { label: 'Home', path: '/', icon: LayoutDashboard, end: true },
  { label: 'Guide', path: '/guide', icon: BookOpen },
  { label: 'Evidence Inbox', path: '/monitor/inbox', icon: Inbox },
  { label: 'Candidates', path: '/etf/approvals', icon: CheckSquare },
  { label: 'Outcomes', path: '/outcomes', icon: BarChart3 },
  { label: 'System Safety', path: '/system', icon: ShieldCheck },
] as const;

const reviewNavItems = [
  { label: 'AI Trading', path: '/ai-trading', icon: Bot },
  { label: 'Manual Trading', path: '/manual-trading', icon: ClipboardPenLine },
  { label: 'Swing Trading', path: '/swing-trading', icon: TrendingUp },
  { label: 'Research', path: '/research', icon: FlaskConical },
  { label: 'Macro Events', path: '/macro/events', icon: Globe },
  { label: 'Analysis', path: '/analysis', icon: BarChart3 },
  { label: 'Notifications', path: '/notifications', icon: Bell },
  { label: 'Settings', path: '/settings', icon: Settings },
  { label: 'Testing', path: '/testing', icon: ShieldCheck },
  { label: 'Paper Trading Test Plan', path: '/testing/plan', icon: CheckSquare },
  { label: 'Mobile Approval Harness', path: '/testing/mobile-approval-harness', icon: CheckSquare },
  { label: 'E2E Tests', path: '/e2e-tests', icon: Server },
  { label: 'Portfolio', path: '/portfolio', icon: Briefcase },
  { label: 'Blotter', path: '/blotter', icon: FileText },
  { label: 'Assistant', path: '/assistant', icon: Bot },
  { label: 'Choose Workflow', path: '/modules', icon: TrendingUp },
  { label: 'Equity manual trading', path: '/equity-alpha/trading', icon: TrendingUp },
  { label: 'Equity guide', path: '/equity-alpha/guide', icon: BookOpen },
  { label: 'Equity strategies', path: '/equity-alpha/strategies', icon: BarChart3 },
  { label: 'Equity timeline', path: '/equity-alpha/timeline', icon: Globe },
  { label: 'Equity trading modes', path: '/equity-alpha/trading-modes', icon: Settings },
  { label: 'Approval-gated ETF trading', path: '/etf/trading', icon: TrendingUp },
  { label: 'ETF guide', path: '/etf/guide', icon: BookOpen },
  { label: 'ETF universe', path: '/etf/universe', icon: Globe },
  { label: 'ETF strategies', path: '/etf/strategies', icon: BarChart3 },
  { label: 'ETF timeline', path: '/etf/timeline', icon: Globe },
  { label: 'ETF trading modes', path: '/etf/trading-modes', icon: Settings },
] as const;

const matches = (pathname: string, path: string) =>
  pathname === path || pathname.startsWith(`${path}/`);

export function AppShell() {
  const [sidebarOpen, setSidebarOpen] = useState(false);
  const { user, authRequired, logout } = useAuth();
  const location = useLocation();
  const reviewRouteActive = useMemo(
    () => reviewNavItems.some((item) => matches(location.pathname, item.path)),
    [location.pathname],
  );
  const [reviewExpanded, setReviewExpanded] = useState(reviewRouteActive);

  useEffect(() => {
    if (reviewRouteActive) setReviewExpanded(true);
  }, [reviewRouteActive]);

  const navLinkClass = ({ isActive }: { isActive: boolean }) =>
    cn(
      'flex items-center gap-3 rounded-md px-3 py-2 text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring',
      isActive
        ? 'bg-accent text-accent-foreground'
        : 'text-muted-foreground hover:bg-muted hover:text-foreground',
    );

  return (
    <div className="min-h-screen bg-background text-foreground">
      {sidebarOpen && (
        <button
          type="button"
          aria-label="Close navigation"
          className="fixed inset-0 z-40 bg-black/50 md:hidden"
          onClick={() => setSidebarOpen(false)}
        />
      )}
      <aside
        className={cn(
          'fixed inset-y-0 left-0 z-50 w-64 transform border-r border-border bg-card transition-transform duration-200 md:translate-x-0',
          sidebarOpen ? 'translate-x-0' : '-translate-x-full',
        )}
      >
        <div className="flex h-full flex-col">
          <div className="flex h-14 items-center gap-3 border-b border-border px-4">
            <img src={JaxLogo} alt="Jax Logo" className="h-8 w-auto" />
            <span className="text-lg font-semibold">Jax</span>
            <Button
              variant="ghost"
              size="icon"
              className="ml-auto md:hidden"
              aria-label="Close sidebar"
              onClick={() => setSidebarOpen(false)}
            >
              <X className="h-5 w-5" />
            </Button>
          </div>
          <nav className="flex-1 overflow-y-auto p-3" aria-label="Application navigation">
            <div aria-label="Primary navigation" className="space-y-1">
              {primaryNavItems.map((item) => (
                <NavLink
                  key={item.path}
                  to={item.path}
                  end={item.path === '/'}
                  onClick={() => setSidebarOpen(false)}
                  className={navLinkClass}
                >
                  <item.icon className="h-4 w-4 shrink-0" aria-hidden="true" />
                  {item.label}
                </NavLink>
              ))}
            </div>
            <div className="mt-3 border-t border-border pt-3">
              <button
                type="button"
                onClick={() => setReviewExpanded((value) => !value)}
                className={cn(
                  'flex w-full items-center gap-3 rounded-md px-3 py-2 text-left text-sm font-medium focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring',
                  reviewRouteActive
                    ? 'bg-accent text-accent-foreground'
                    : 'text-muted-foreground hover:bg-muted hover:text-foreground',
                )}
                aria-expanded={reviewExpanded}
                aria-controls="review-navigation"
              >
                <ShieldCheck className="h-4 w-4" aria-hidden="true" />
                <span className="flex-1">Review</span>
                {reviewExpanded ? (
                  <ChevronDown className="h-4 w-4" />
                ) : (
                  <ChevronRight className="h-4 w-4" />
                )}
              </button>
              {reviewExpanded && (
                <div
                  id="review-navigation"
                  aria-label="Review navigation"
                  className="ml-4 mt-1 space-y-1 border-l border-border pl-2"
                >
                  {reviewNavItems.map((item) => (
                    <NavLink
                      key={item.path}
                      to={item.path}
                      end
                      onClick={() => setSidebarOpen(false)}
                      className={navLinkClass}
                    >
                      <item.icon className="h-4 w-4 shrink-0" aria-hidden="true" />
                      {item.label}
                    </NavLink>
                  ))}
                </div>
              )}
            </div>
          </nav>
          <div className="space-y-2 border-t border-border p-3">
            {authRequired && user && !user.anonymous && (
              <div className="flex items-center justify-between">
                <span className="truncate text-xs text-muted-foreground">{user.username}</span>
                <Button
                  variant="ghost"
                  size="icon"
                  className="h-7 w-7"
                  aria-label="Sign out"
                  onClick={logout}
                >
                  <LogOut className="h-3 w-3" />
                </Button>
              </div>
            )}
            <div className="flex items-center gap-2 text-xs text-muted-foreground">
              <Activity className="h-3 w-3 text-success" aria-hidden="true" />
              <span>Session active</span>
            </div>
          </div>
        </div>
      </aside>
      <div className="md:pl-64">
        <header className="sticky top-0 z-30 flex h-14 items-center gap-3 border-b border-border bg-background/95 px-3 backdrop-blur sm:px-4">
          <Button
            variant="ghost"
            size="icon"
            className="md:hidden"
            aria-label="Open sidebar"
            onClick={() => setSidebarOpen(true)}
          >
            <Menu className="h-5 w-5" />
          </Button>
          <NavLink to="/" className="flex items-center gap-2 md:hidden">
            <img src={JaxLogo} alt="Jax Logo" className="h-7 w-auto" />
            <span className="font-semibold">Jax</span>
          </NavLink>
          <div className="ml-auto flex min-w-0 items-center gap-2">
            <BeginnerModeToggle />
            {authRequired && user && !user.anonymous && (
              <Button variant="ghost" size="sm" onClick={logout} className="hidden gap-1.5 sm:flex">
                <LogOut className="h-3.5 w-3.5" />
                Sign out
              </Button>
            )}
          </div>
        </header>
        <main className="p-4 md:p-6 lg:p-8">
          {reviewRouteActive && (
            <div
              role="note"
              className="mb-4 rounded-md border border-border bg-muted/40 px-4 py-3 text-sm text-muted-foreground"
            >
              <strong className="text-foreground">Review page</strong> — this area has not yet been
              redesigned for the current Jax workflow.
            </div>
          )}
          <Outlet />
        </main>
      </div>
    </div>
  );
}
