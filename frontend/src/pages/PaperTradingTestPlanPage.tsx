import { useEffect, useMemo, useState } from 'react';
import { Link } from 'react-router-dom';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';

type PlanTask = {
  id: string;
  title: string;
  detail: string;
};

type PlanSection = {
  id: string;
  title: string;
  description: string;
  tasks: PlanTask[];
};

const STORAGE_KEY = 'jax.paperTradingTestPlan.v1';

const sections: PlanSection[] = [
  {
    id: 'paper-trades',
    title: 'Paper Trading Workflow',
    description: 'Validate end-to-end paper execution flow from idea to managed open position and closeout.',
    tasks: [
      {
        id: 'paper-1',
        title: 'Switch to paper mode',
        detail: 'Confirm environment/mode indicates paper trading and no live endpoint is active.',
      },
      {
        id: 'paper-2',
        title: 'Create candidate from module flow',
        detail: 'Open either ETF or Equity Alpha workflow and create a candidate with visible evidence.',
      },
      {
        id: 'paper-3',
        title: 'Submit order from ticket',
        detail: 'Place a paper order with defined stop, target, and size constraints.',
      },
      {
        id: 'paper-4',
        title: 'Verify blotter updates',
        detail: 'Check working/filled states and ensure order lifecycle transitions are visible.',
      },
      {
        id: 'paper-5',
        title: 'Confirm portfolio impact',
        detail: 'Validate position sizing, unrealized P/L, and risk metrics update on portfolio view.',
      },
      {
        id: 'paper-6',
        title: 'Document closeout path',
        detail: 'Exit position and verify realized results and audit timeline entries.',
      },
    ],
  },
  {
    id: 'research',
    title: 'Research Workflow',
    description: 'Validate research tools that support planning, hypothesis checks, and trade decisions.',
    tasks: [
      {
        id: 'research-1',
        title: 'Open strategy cards and select a setup',
        detail: 'Review available setup guidance and choose one candidate strategy for the session.',
      },
      {
        id: 'research-2',
        title: 'Use timeline to review context',
        detail: 'Confirm timeline view shows recent signals, catalysts, and supporting notes.',
      },
      {
        id: 'research-3',
        title: 'Validate instrument universe data',
        detail: 'Check symbol metadata, eligibility status, and category filters as expected.',
      },
      {
        id: 'research-4',
        title: 'Capture a research note',
        detail: 'Add a note or annotation that can be referenced during execution decisions.',
      },
      {
        id: 'research-5',
        title: 'Verify evidence handoff',
        detail: 'Ensure research context appears in candidate evidence or approval views.',
      },
    ],
  },
  {
    id: 'ai-scanning',
    title: 'AI News and Chart Scanning Workflow',
    description: 'Validate how the AI assistant continuously scans news and chart context for opportunities.',
    tasks: [
      {
        id: 'ai-1',
        title: 'Run AI scan prompt for current market session',
        detail: 'Use assistant flow to request opportunities based on today\'s news and intraday chart structure.',
      },
      {
        id: 'ai-2',
        title: 'Confirm source transparency',
        detail: 'Check that AI output references the news/chart signals used in each suggestion.',
      },
      {
        id: 'ai-3',
        title: 'Validate opportunity ranking',
        detail: 'Confirm opportunities are prioritized by strength, confidence, or explicit criteria.',
      },
      {
        id: 'ai-4',
        title: 'Promote one idea into candidate review',
        detail: 'Take one AI suggestion into the relevant module workflow and verify continuity.',
      },
      {
        id: 'ai-5',
        title: 'Repeat scan after market update',
        detail: 'Re-run the AI scan after a context change and verify stale ideas are replaced or re-ranked.',
      },
      {
        id: 'ai-6',
        title: 'Record decision outcome',
        detail: 'Log whether the AI suggestion was accepted, modified, or rejected and why.',
      },
    ],
  },
];

function loadInitialState() {
  if (typeof window === 'undefined') {
    return {} as Record<string, boolean>;
  }

  try {
    const raw = window.localStorage.getItem(STORAGE_KEY);
    if (!raw) {
      return {} as Record<string, boolean>;
    }
    return JSON.parse(raw) as Record<string, boolean>;
  } catch {
    return {} as Record<string, boolean>;
  }
}

export function PaperTradingTestPlanPage() {
  const [checked, setChecked] = useState<Record<string, boolean>>(() => loadInitialState());

  useEffect(() => {
    window.localStorage.setItem(STORAGE_KEY, JSON.stringify(checked));
  }, [checked]);

  const allTasks = useMemo(() => sections.flatMap((section) => section.tasks), []);
  const completedCount = useMemo(
    () => allTasks.filter((task) => checked[task.id]).length,
    [allTasks, checked]
  );
  const percentComplete = useMemo(() => {
    if (allTasks.length === 0) {
      return 0;
    }
    return Math.round((completedCount / allTasks.length) * 100);
  }, [allTasks.length, completedCount]);
  const progressWidthClass = percentComplete >= 95
    ? 'w-full'
    : percentComplete >= 75
      ? 'w-3/4'
      : percentComplete >= 50
        ? 'w-1/2'
        : percentComplete >= 25
          ? 'w-1/4'
          : 'w-0';

  return (
    <div className="space-y-6">
      <div>
        <p className="text-xs font-semibold uppercase tracking-widest text-primary mb-1">EXECUTION READINESS</p>
        <h1 className="text-2xl font-bold md:text-3xl">Paper Trading UI Test Plan</h1>
        <p className="mt-1 text-muted-foreground">
          Use this checklist to track completed UI tests across paper trading, research, and AI opportunity scanning.
        </p>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Progress</CardTitle>
          <CardDescription>
            Completion: {completedCount}/{allTasks.length} tasks ({percentComplete}%)
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-3">
          <div className="space-y-2" aria-label="Checklist completion">
            <div className="h-3 w-full overflow-hidden rounded-full bg-muted">
              <div
                className={`h-full rounded-full bg-success transition-all duration-300 ${progressWidthClass}`}
              />
            </div>
            <div className="grid gap-2 sm:grid-cols-3">
              {sections.map((section) => {
                const sectionCompleted = section.tasks.filter((task) => checked[task.id]).length;
                return (
                  <div key={section.id} className="rounded-md border border-border bg-muted/30 px-2.5 py-2 text-xs">
                    <div className="font-medium">{section.title}</div>
                    <div className="text-muted-foreground">
                      {sectionCompleted}/{section.tasks.length} complete
                    </div>
                  </div>
                );
              })}
            </div>
          </div>
          <div className="flex flex-wrap gap-2">
            <Button
              variant="outline"
              onClick={() => {
                const next: Record<string, boolean> = {};
                for (const task of allTasks) {
                  next[task.id] = true;
                }
                setChecked(next);
              }}
            >
              Mark All Complete
            </Button>
            <Button
              variant="secondary"
              onClick={() => {
                setChecked({});
              }}
            >
              Clear All
            </Button>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Guided Quick Start</CardTitle>
          <CardDescription>Use this sequence to learn how AI suggestions flow into strategy setup, backtesting, and approvals.</CardDescription>
        </CardHeader>
        <CardContent className="space-y-2 text-sm">
          <div className="rounded-md border border-border bg-muted/30 p-3">
            1. Open <Link className="font-medium text-primary underline" to="/etf/guide">ETF Guide</Link> to understand the workflow.
          </div>
          <div className="rounded-md border border-border bg-muted/30 p-3">
            2. Open <Link className="font-medium text-primary underline" to="/etf/strategies">ETF Strategies</Link> and choose a setup to prefill Research.
          </div>
          <div className="rounded-md border border-border bg-muted/30 p-3">
            3. In <Link className="font-medium text-primary underline" to="/research">Research</Link>, pick Strategy Type, save instance, then run backtest.
          </div>
          <div className="rounded-md border border-border bg-muted/30 p-3">
            4. Review outcomes in <Link className="font-medium text-primary underline" to="/analysis">Analysis</Link> and promote a candidate to <Link className="font-medium text-primary underline" to="/etf/approvals">ETF Approvals</Link>.
          </div>
          <div className="rounded-md border border-border bg-muted/30 p-3">
            5. Ask <Link className="font-medium text-primary underline" to="/assistant">Assistant</Link> for "top ETF opportunities with cited news + chart context" and compare with your strategy run.
          </div>
        </CardContent>
      </Card>

      {sections.map((section) => (
        <Card key={section.id}>
          <CardHeader>
            <CardTitle>{section.title}</CardTitle>
            <CardDescription>{section.description}</CardDescription>
          </CardHeader>
          <CardContent className="space-y-3">
            {section.tasks.map((task) => (
              <label
                key={task.id}
                className="flex cursor-pointer items-start gap-3 rounded-md border border-border p-3 hover:bg-muted/40"
              >
                <input
                  type="checkbox"
                  className="mt-1 h-4 w-4"
                  checked={Boolean(checked[task.id])}
                  onChange={(event) => {
                    const isChecked = event.target.checked;
                    setChecked((prev) => ({ ...prev, [task.id]: isChecked }));
                  }}
                />
                <span>
                  <span className="block font-medium text-foreground">{task.title}</span>
                  <span className="block text-sm text-muted-foreground">{task.detail}</span>
                </span>
              </label>
            ))}
          </CardContent>
        </Card>
      ))}
    </div>
  );
}
