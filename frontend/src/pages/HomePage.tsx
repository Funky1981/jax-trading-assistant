import { useEffect } from 'react';
import { Link } from 'react-router-dom';
import { ArrowRight, BarChart3, Bot, ClipboardPenLine } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { useBeginnerMode } from '@/context/BeginnerUXContextValue';
import { emitAnalyticsEvent } from '@/lib/analytics';

const startingActions = [
  {
    title: 'Find AI opportunities',
    description: 'Review scanner output, evidence, and the right route for each idea.',
    path: '/ai-trading',
    icon: Bot,
  },
  {
    title: 'Place a manual trade',
    description: 'Use the manual path for instruments that do not require approval routing.',
    path: '/manual-trading',
    icon: ClipboardPenLine,
  },
  {
    title: 'Test a strategy',
    description: 'Run research workflows and compare historical strategy performance.',
    path: '/research',
    icon: BarChart3,
  },
];

export function HomePage() {
  const { mode } = useBeginnerMode();
  const isSimple = mode === 'simple';

  useEffect(() => {
    emitAnalyticsEvent('page_viewed', { source_surface: 'home' });
  }, []);

  return (
    <div className="mx-auto flex w-full max-w-6xl flex-col gap-6">
      <section className="space-y-3">
        <p className="text-xs font-semibold uppercase tracking-widest text-primary">Home</p>
        <div className="max-w-3xl space-y-2">
          <h1 className="text-2xl font-bold md:text-3xl">{isSimple ? 'Start Here' : 'Home'}</h1>
          <p className="text-base text-muted-foreground">
            {isSimple
              ? 'Jax is a paper-trading assistant. It helps you find ideas, check evidence, and keep every action behind safety steps.'
              : 'Jax helps you find AI-backed trading opportunities, review the evidence, and act through the right safety-controlled workflow.'}
          </p>
        </div>
      </section>

      {isSimple && (
        <Card>
          <CardHeader>
            <CardTitle>How The App Fits Together</CardTitle>
            <CardDescription>Use this as the mental model before clicking into tools.</CardDescription>
          </CardHeader>
          <CardContent className="grid gap-3 text-sm md:grid-cols-4">
            <div>
              <p className="font-semibold text-foreground">Research</p>
              <p className="text-muted-foreground">Test ideas before trading them.</p>
            </div>
            <div>
              <p className="font-semibold text-foreground">AI Trading</p>
              <p className="text-muted-foreground">Review ideas Jax has found.</p>
            </div>
            <div>
              <p className="font-semibold text-foreground">Approvals</p>
              <p className="text-muted-foreground">Decide whether a candidate is allowed.</p>
            </div>
            <div>
              <p className="font-semibold text-foreground">Manual Trading</p>
              <p className="text-muted-foreground">Place or manage paper orders after review.</p>
            </div>
          </CardContent>
        </Card>
      )}

      <section className="grid gap-4 md:grid-cols-3" aria-label="Start a workflow">
        {startingActions.map((action) => (
          <Card key={action.path} className="flex h-full flex-col">
            <CardHeader>
              <div className="mb-3 flex h-10 w-10 items-center justify-center rounded-md border border-border bg-muted text-primary">
                <action.icon className="h-5 w-5" />
              </div>
              <CardTitle>{action.title}</CardTitle>
              <CardDescription>{action.description}</CardDescription>
            </CardHeader>
            <CardContent className="mt-auto">
              <Button asChild className="w-full justify-between">
                <Link to={action.path}>
                  {action.title}
                  <ArrowRight className="h-4 w-4" />
                </Link>
              </Button>
            </CardContent>
          </Card>
        ))}
      </section>
    </div>
  );
}
