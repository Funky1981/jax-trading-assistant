import { useState } from 'react';
import { NavLink, Outlet, useLocation } from 'react-router-dom';
import { 
  Menu, 
  X, 
  LayoutDashboard, 
  FileText, 
  Briefcase, 
  Settings,
  Receipt,
  Activity,
  TrendingUp,
  Server,
  LogOut,
} from 'lucide-react';
import { cn } from '@/lib/utils';
import { Button } from '@/components/ui/button';
import { Separator } from '@/components/ui/separator';
import JaxLogo from '@/images/jax_ai_trader.svg';
import { useAuth } from '@/contexts/AuthContext';

const navItems = [
  { label: 'Dashboard', path: '/', icon: LayoutDashboard, end: true },
  { label: 'Trading', path: '/trading', icon: TrendingUp },
  { label: 'System', path: '/system', icon: Server },
  { label: 'Portfolio', path: '/portfolio', icon: Briefcase },
  { label: 'Blotter', path: '/blotter', icon: FileText },
  { label: 'Settings', path: '/settings', icon: Settings },
];

export function AppShell() {
  const [sidebarOpen, setSidebarOpen] = useState(false);
  const location = useLocation();
  const { user, authRequired, logout } = useAuth();

  console.log('AppShell rendering, current path:', location.pathname);

  return (
    <div className="min-h-screen bg-background text-foreground">
      {/* Mobile overlay */}
      {sidebarOpen && (
        <div
          className="fixed inset-0 z-40 bg-black/50 md:hidden"
          onClick={() => setSidebarOpen(false)}
        />
      )}

      {/* Sidebar */}
      <aside
        className={cn(
          'fixed inset-y-0 left-0 z-50 w-64 transform border-r border-border bg-card transition-transform duration-200 md:translate-x-0',
          sidebarOpen ? 'translate-x-0' : '-translate-x-full'
        )}
      >
        <div className="flex h-full flex-col">
          {/* Sidebar Header */}
          <div className="flex h-16 items-center gap-3 border-b border-border px-4">
            <img
              src={JaxLogo}
              alt="Jax Logo"
              className="h-8 w-auto"
            />
            <span className="text-lg font-semibold">Jax Trader</span>
            <Button
              variant="ghost"
              size="icon"
              className="ml-auto md:hidden"
              onClick={() => setSidebarOpen(false)}
            >
              <X className="h-5 w-5" />
            </Button>
          </div>

          {/* Navigation */}
          <nav className="flex-1 space-y-1 p-4">
            <p className="mb-3 text-xs font-semibold uppercase tracking-wider text-muted-foreground">
              Navigation
            </p>
            {navItems.map((item) => (
              <NavLink
                key={item.path}
                to={item.path}
                end={item.end}
                onClick={(e) => {
                  console.log(`🔗 NavLink clicked: ${item.label} -> ${item.path}`);
                  setSidebarOpen(false);
                }}
                className={({ isActive }) =>
                  cn(
                    'flex items-center gap-3 rounded-md px-3 py-2 text-sm font-medium transition-colors',
                    isActive
                      ? 'bg-accent text-accent-foreground'
                      : 'text-muted-foreground hover:bg-muted hover:text-foreground'
                  )
                }
              >
                <item.icon className="h-4 w-4" />
                {item.label}
              </NavLink>
            ))}
          </nav>

          {/* Sidebar Footer */}
          <div className="border-t border-border p-4 space-y-2">
            {authRequired && user && !user.anonymous && (
              <div className="flex items-center justify-between">
                <span className="text-xs text-muted-foreground truncate">{user.username}</span>
                <Button
                  variant="ghost"
                  size="icon"
                  className="h-6 w-6 shrink-0"
                  title="Sign out"
                  onClick={logout}
                >
                  <LogOut className="h-3 w-3" />
                </Button>
              </div>
            )}
            <div className="flex items-center gap-2 text-xs text-muted-foreground">
              <Activity className="h-3 w-3 text-success" />
              <span>System Online</span>
            </div>
          </div>
        </div>
      </aside>

      {/* Main Content Area */}
      <div className="md:pl-64">
        {/* Header */}
        <header className="sticky top-0 z-30 flex h-16 items-center gap-4 border-b border-border bg-background/95 px-4 backdrop-blur supports-[backdrop-filter]:bg-background/60">
          <Button
            variant="ghost"
            size="icon"
            className="md:hidden"
            onClick={() => setSidebarOpen(true)}
          >
            <Menu className="h-5 w-5" />
          </Button>

          <NavLink to="/" className="flex items-center gap-3 md:hidden">
            <img
              src={JaxLogo}
              alt="Jax Logo"
              className="h-8 w-auto"
            />
            <span className="text-lg font-semibold">Jax Trader</span>
          </NavLink>

          <div className="ml-auto flex items-center gap-4">
            <div className="flex items-center gap-2">
              <div className="h-2 w-2 rounded-full bg-success" />
              <span className="text-sm text-muted-foreground">
                Live Session
              </span>
            </div>
            {authRequired && user && !user.anonymous && (
              <Button variant="ghost" size="sm" onClick={logout} className="gap-1.5">
                <LogOut className="h-3.5 w-3.5" />
                Sign out
              </Button>
            )}
          </div>
        </header>

        {/* Page Content */}
        <main className="p-4 md:p-6 lg:p-8">
          <Outlet key={location.pathname} />
        </main>
      </div>
    </div>
  );
}
