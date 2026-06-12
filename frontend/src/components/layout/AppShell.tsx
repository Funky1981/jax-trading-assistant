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
  Clock,
  FileText,
  FlaskConical,
  Globe,
  Inbox,
  Layers,
  LayoutDashboard,
  ListChecks,
  LogOut,
  Menu,
  MonitorCheck,
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
import { useBeginnerMode } from '@/context/BeginnerUXContextValue';

const primaryNavItems = [
  { label: 'Home', path: '/', icon: LayoutDashboard, end: true },
  { label: 'Guide', path: '/guide', icon: BookOpen },
  { label: 'AI Trading', path: '/ai-trading', icon: Bot },
  { label: 'Manual Trading', path: '/manual-trading', icon: ClipboardPenLine },
  { label: 'Approvals', path: '/etf/approvals', icon: CheckSquare },
  { label: 'Research', path: '/research', icon: FlaskConical },
  { label: 'Macro Events', path: '/macro/events', icon: Globe },
  { label: 'Monitor Inbox', path: '/monitor/inbox', icon: Inbox },
  { label: 'Analysis', path: '/analysis', icon: BarChart3 },
  { label: 'Notifications', path: '/notifications', icon: Bell },
  { label: 'Settings', path: '/settings', icon: Settings },
];

const advancedSections = [
  {
    id: 'admin-qa',
    label: 'Admin and QA',
    icon: ShieldCheck,
    items: [
      { label: 'System', path: '/system', icon: Server },
      { label: 'Testing', path: '/testing', icon: ShieldCheck },
      { label: 'Paper Trading Test Plan', path: '/testing/plan', icon: CheckSquare },
      { label: 'Mobile Approval Harness', path: '/testing/mobile-approval-harness', icon: CheckSquare },
      { label: 'E2E Tests', path: '/e2e-tests', icon: MonitorCheck },
      { label: 'Portfolio', path: '/portfolio', icon: Briefcase },
      { label: 'Blotter', path: '/blotter', icon: FileText },
      { label: 'Assistant', path: '/assistant', icon: Bot },
    ],
  },
  {
    id: 'learn-legacy',
    label: 'Guides and legacy',
    icon: BookOpen,
    items: [
      { label: 'User Guide', path: '/guide', icon: BookOpen },
      { label: 'Choose Workflow', path: '/modules', icon: TrendingUp },
      { label: 'Equity manual trading', path: '/equity-alpha/trading', icon: TrendingUp },
      { label: 'Equity guide', path: '/equity-alpha/guide', icon: BookOpen },
      { label: 'Equity strategies', path: '/equity-alpha/strategies', icon: Layers },
      { label: 'Equity timeline', path: '/equity-alpha/timeline', icon: Clock },
      { label: 'Equity trading modes', path: '/equity-alpha/trading-modes', icon: ListChecks },
      { label: 'Approval-gated ETF trading', path: '/etf/trading', icon: TrendingUp },
      { label: 'ETF guide', path: '/etf/guide', icon: BookOpen },
      { label: 'ETF universe', path: '/etf/universe', icon: Globe },
      { label: 'ETF strategies', path: '/etf/strategies', icon: Layers },
      { label: 'ETF timeline', path: '/etf/timeline', icon: Clock },
      { label: 'ETF trading modes', path: '/etf/trading-modes', icon: ListChecks },
    ],
  },
];

export function AppShell() {
  const [sidebarOpen, setSidebarOpen] = useState(false);
  const { user, authRequired, logout } = useAuth();
  const { mode } = useBeginnerMode();
  const location = useLocation();
  const visibleAdvancedSections = useMemo(() => {
    if (mode !== 'simple') {
      return advancedSections;
    }

    return advancedSections.map((section) =>
      section.id === 'learn-legacy'
        ? {
            ...section,
            label: 'Legacy pages',
            items: section.items.filter((item) => item.path !== '/guide'),
          }
        : section
    );
  }, [mode]);

  const activeSectionIds = useMemo(() => {
    return visibleAdvancedSections
      .filter((section) => section.items.some((item) => location.pathname === item.path || location.pathname.startsWith(`${item.path}/`)))
      .map((section) => section.id);
  }, [location.pathname, visibleAdvancedSections]);

  const [expandedSections, setExpandedSections] = useState<Record<string, boolean>>(() =>
    Object.fromEntries(advancedSections.map((section) => [section.id, false]))
  );

  useEffect(() => {
    if (activeSectionIds.length === 0) {
      return;
    }

    setExpandedSections((current) => {
      const next = { ...current };
      let changed = false;

      for (const sectionId of activeSectionIds) {
        if (!next[sectionId]) {
          next[sectionId] = true;
          changed = true;
        }
      }

      return changed ? next : current;
    });
  }, [activeSectionIds]);

  const toggleSection = (sectionId: string) => {
    setExpandedSections((current) => ({
      ...current,
      [sectionId]: !current[sectionId],
    }));
  };

  const navLinkClass = ({ isActive }: { isActive: boolean }) =>
    cn(
      'flex items-center gap-3 rounded-md px-3 py-2 text-sm font-medium transition-colors',
      isActive
        ? 'bg-accent text-accent-foreground'
        : 'text-muted-foreground hover:bg-muted hover:text-foreground'
    );

  return (
    <div className="min-h-screen bg-background text-foreground">
      {sidebarOpen && (
        <div
          className="fixed inset-0 z-40 bg-black/50 md:hidden"
          onClick={() => setSidebarOpen(false)}
        />
      )}

      <aside
        className={cn(
          'fixed inset-y-0 left-0 z-50 w-64 transform border-r border-border bg-card transition-transform duration-200 md:translate-x-0',
          sidebarOpen ? 'translate-x-0' : '-translate-x-full'
        )}
      >
        <div className="flex h-full flex-col">
          <div className="flex h-16 items-center gap-3 border-b border-border px-4">
            <img src={JaxLogo} alt="Jax Logo" className="h-8 w-auto" />
            <span className="text-lg font-semibold">Jax Trader</span>
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

          <nav className="flex-1 overflow-y-auto p-4">
            <p className="mb-3 text-xs font-semibold uppercase tracking-wider text-muted-foreground">
              Navigation
            </p>

            <div aria-label="Primary navigation" className="space-y-1">
              {primaryNavItems.map((item) => (
                <NavLink
                  key={item.path}
                  to={item.path}
                  end={item.end}
                  onClick={() => setSidebarOpen(false)}
                  className={navLinkClass}
                >
                  <item.icon className="h-4 w-4" />
                  {item.label}
                </NavLink>
              ))}
            </div>

            <div className="mt-4 border-t border-border pt-4">
              <p className="mb-2 px-3 text-xs font-semibold uppercase tracking-wider text-muted-foreground">
                Advanced
              </p>

              {visibleAdvancedSections.map((section) => {
                const sectionActive = section.items.some(
                  (item) => location.pathname === item.path || location.pathname.startsWith(`${item.path}/`)
                );
                const expanded = expandedSections[section.id];

                return (
                  <div key={section.id} className="space-y-1">
                    <button
                      type="button"
                      onClick={() => toggleSection(section.id)}
                      className={cn(
                        'flex w-full items-center gap-3 rounded-md px-3 py-2 text-left text-sm font-medium transition-colors',
                        sectionActive
                          ? 'bg-accent text-accent-foreground'
                          : 'text-muted-foreground hover:bg-muted hover:text-foreground'
                      )}
                      aria-expanded={expanded ? 'true' : 'false'}
                      aria-controls={`${section.id}-nav-items`}
                      aria-label={`${section.label} navigation`}
                    >
                      <section.icon className="h-4 w-4" />
                      <span className="flex-1">{section.label}</span>
                      {expanded ? <ChevronDown className="h-4 w-4" /> : <ChevronRight className="h-4 w-4" />}
                    </button>

                    {expanded && (
                      <div id={`${section.id}-nav-items`} className="ml-4 space-y-1 border-l border-border pl-3">
                        {section.items.map((item) => (
                          <NavLink
                            key={item.path}
                            to={item.path}
                            end
                            onClick={() => setSidebarOpen(false)}
                            className={navLinkClass}
                          >
                            <item.icon className="h-4 w-4" />
                            {item.label}
                          </NavLink>
                        ))}
                      </div>
                    )}
                  </div>
                );
              })}
            </div>
          </nav>

          <div className="space-y-2 border-t border-border p-4">
            {authRequired && user && !user.anonymous && (
              <div className="flex items-center justify-between">
                <span className="truncate text-xs text-muted-foreground">{user.username}</span>
                <Button
                  variant="ghost"
                  size="icon"
                  className="h-6 w-6 shrink-0"
                  title="Sign out"
                  aria-label="Sign out"
                  onClick={logout}
                >
                  <LogOut className="h-3 w-3" />
                </Button>
              </div>
            )}
            <div className="flex items-center gap-2 text-xs text-muted-foreground">
              <Activity className="h-3 w-3 text-success" />
              <span>Services Online</span>
            </div>
          </div>
        </div>
      </aside>

      <div className="md:pl-64">
        <header className="sticky top-0 z-30 flex h-16 items-center gap-4 border-b border-border bg-background/95 px-4 backdrop-blur supports-[backdrop-filter]:bg-background/60">
          <Button
            variant="ghost"
            size="icon"
            className="md:hidden"
            aria-label="Open sidebar"
            onClick={() => setSidebarOpen(true)}
          >
            <Menu className="h-5 w-5" />
          </Button>

          <NavLink to="/" className="flex items-center gap-3 md:hidden">
            <img src={JaxLogo} alt="Jax Logo" className="h-8 w-auto" />
            <span className="text-lg font-semibold">Jax Trader</span>
          </NavLink>

          <div className="ml-auto flex items-center gap-4">
            <BeginnerModeToggle />
            <div className="flex items-center gap-2">
              <div className="h-2 w-2 rounded-full bg-success" />
              <span className="text-sm text-muted-foreground">Session Active</span>
            </div>
            {authRequired && user && !user.anonymous && (
              <Button variant="ghost" size="sm" onClick={logout} className="gap-1.5">
                <LogOut className="h-3.5 w-3.5" />
                Sign out
              </Button>
            )}
          </div>
        </header>

        <main className="p-4 md:p-6 lg:p-8">
          <Outlet />
        </main>
      </div>
    </div>
  );
}
