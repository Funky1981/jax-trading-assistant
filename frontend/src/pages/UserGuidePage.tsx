import { Link } from 'react-router-dom';
import { AlertTriangle, ArrowRight, CheckCircle2, Circle, Clock } from 'lucide-react';
import { useOperatorEvidenceOverview } from '@/hooks/useOperatorEvidenceOverview';
import { isPaperSafe } from '@/lib/operator-safety';
import { beginnerGlossary } from '@/data/beginner-glossary';
import { PageIntro, SafetyBanner, TechnicalDetailsDisclosure } from '@/components/ui/beginner-help';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';

type Task = { label: string; status: string; path: string };

export function UserGuidePage() {
  const overview = useOperatorEvidenceOverview();
  const safe = isPaperSafe(overview.data);
  const tasks: Task[] = overview.data
    ? [
        {
          label: 'Confirm paper-safe mode',
          status: safe ? 'Done' : 'Needs attention',
          path: '/system',
        },
        {
          label: 'Review new evidence',
          status: overview.data.genuineEvents > 0 ? 'Ready' : 'No data yet',
          path: '/monitor/inbox',
        },
        {
          label: 'Review a candidate when one exists',
          status: overview.data.candidates > 0 ? 'Ready' : 'Waiting',
          path: '/etf/approvals',
        },
        {
          label: 'Review hypothetical outcomes',
          status:
            overview.data.completedCheckpoints > 0 || overview.data.pendingCheckpoints > 0
              ? 'Ready'
              : 'No data yet',
          path: '/outcomes',
        },
        {
          label: 'Check System Safety if anything looks wrong',
          status: safe ? 'Ready' : 'Needs attention',
          path: '/system',
        },
      ]
    : [
        {
          label: 'Confirm paper-safe mode',
          status: overview.isPending ? 'Waiting' : 'Needs attention',
          path: '/system',
        },
        { label: 'Review new evidence', status: 'No data yet', path: '/monitor/inbox' },
        {
          label: 'Review a candidate when one exists',
          status: 'No data yet',
          path: '/etf/approvals',
        },
        { label: 'Review hypothetical outcomes', status: 'No data yet', path: '/outcomes' },
        { label: 'Check System Safety if anything looks wrong', status: 'Ready', path: '/system' },
      ];
  const next = !safe
    ? { label: 'Open System Safety', path: '/system' }
    : overview.data && overview.data.genuineEvents > 0
      ? { label: 'Open Evidence Inbox', path: '/monitor/inbox' }
      : overview.data && overview.data.candidates > 0
        ? { label: 'Open Candidates', path: '/etf/approvals' }
        : overview.data &&
            (overview.data.completedCheckpoints > 0 || overview.data.pendingCheckpoints > 0)
          ? { label: 'Open Outcomes', path: '/outcomes' }
          : { label: 'Open Evidence Inbox', path: '/monitor/inbox' };

  return (
    <div className="mx-auto max-w-5xl space-y-6">
      <PageIntro
        eyebrow="Guide"
        title="Start with the current Jax workflow"
        description="Jax reviews evidence and tracks hypothetical paper plans. It is not a live trading terminal."
      />
      <SafetyBanner safe={safe} loading={overview.isPending} />
      <div className="grid gap-4 md:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle>What Jax does today</CardTitle>
          </CardHeader>
          <CardContent>
            <ul className="list-disc space-y-2 pl-5 text-sm text-muted-foreground">
              <li>Collect genuine news evidence and store its provenance.</li>
              <li>Review events and create structured candidates when evidence supports one.</li>
              <li>Require human approval before a candidate can progress.</li>
              <li>Track hypothetical paper outcomes.</li>
              <li>Show the current runtime safety state.</li>
            </ul>
          </CardContent>
        </Card>
        <Card>
          <CardHeader>
            <CardTitle>What Jax cannot do</CardTitle>
          </CardHeader>
          <CardContent>
            <ul className="list-disc space-y-2 pl-5 text-sm text-muted-foreground">
              <li>Place a real trade: Jax cannot trade live, place broker orders or fill positions in this workflow.</li>
              <li>Guarantee profit or automatically approve candidates.</li>
              <li>Use leverage above the configured safety limit.</li>
              <li>Treat hypothetical outcomes as real returns.</li>
            </ul>
          </CardContent>
        </Card>
      </div>
      <Card>
        <CardHeader>
          <CardTitle>Your current workflow</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          <ol className="space-y-2">
            {tasks.map((task, index) => (
              <li
                key={task.label}
                className="flex flex-col gap-2 rounded-md border p-3 sm:flex-row sm:items-center"
              >
                <span className="flex flex-1 items-center gap-3">
                  {task.status === 'Done' ? (
                    <CheckCircle2 className="h-5 w-5 text-success" />
                  ) : task.status === 'Needs attention' ? (
                    <AlertTriangle className="h-5 w-5 text-destructive" />
                  ) : task.status === 'Waiting' ? (
                    <Clock className="h-5 w-5 text-muted-foreground" />
                  ) : (
                    <Circle className="h-5 w-5 text-primary" />
                  )}
                  <span>
                    <span className="mr-2 text-muted-foreground">{index + 1}.</span>
                    {task.label}
                  </span>
                </span>
                <span className="text-sm font-medium">{task.status}</span>
              </li>
            ))}
          </ol>
          <Button asChild>
            <Link to={next.path}>
              {next.label}
              <ArrowRight className="ml-2 h-4 w-4" />
            </Link>
          </Button>
        </CardContent>
      </Card>
      <Card id="evidence-inbox" className="scroll-mt-4">
        <CardHeader>
          <CardTitle>How to review the Evidence Inbox</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3 text-sm text-muted-foreground">
          <p>
            Filter the list, open one item, then read its source, three separate timestamps,
            analysis method and event journey. Opening evidence is read-only.
          </p>
          <p>
            Awaiting processing is different from research only. Unknown assets means Jax has no
            truthful persisted mapping; it does not insert a fallback symbol.
          </p>
          <p>
            A deterministic decision finishes as NO_TRADE when evidence does not justify continued
            review, WATCH when a material event still needs evidence or a truthful asset mapping,
            and CANDIDATE only when the complete persisted candidate, evidence, trust and risk
            contract passes. These outcomes do not approve or execute a trade.
          </p>
          <Button asChild variant="outline" size="sm">
            <Link to="/monitor/inbox">Open Evidence Inbox</Link>
          </Button>
        </CardContent>
      </Card>
      <Card>
        <CardHeader>
          <CardTitle>Key terms</CardTitle>
        </CardHeader>
        <CardContent>
          <dl className="grid gap-4 sm:grid-cols-2">
            {Object.entries(beginnerGlossary).map(([term, definition]) => (
              <div key={term}>
                <dt className="font-semibold">{term}</dt>
                <dd className="text-sm text-muted-foreground">{definition}</dd>
              </div>
            ))}
          </dl>
        </CardContent>
      </Card>
      <Card>
        <CardHeader>
          <CardTitle>Troubleshooting</CardTitle>
        </CardHeader>
        <CardContent className="space-y-2 text-sm text-muted-foreground">
          <p>
            If safety is unknown or warns, open System Safety before relying on any other
            information.
          </p>
          <p>
            If Evidence Inbox is empty, Jax may be waiting for a genuine event. An API error is
            different from an empty inbox and should be investigated.
          </p>
          <nav aria-label="Guide destinations" className="flex flex-wrap gap-2 pt-2">
            <Button asChild variant="outline" size="sm">
              <Link to="/monitor/inbox">Evidence Inbox</Link>
            </Button>
            <Button asChild variant="outline" size="sm">
              <Link to="/etf/approvals">Candidates</Link>
            </Button>
            <Button asChild variant="outline" size="sm">
              <Link to="/outcomes">Outcomes</Link>
            </Button>
            <Button asChild variant="outline" size="sm">
              <Link to="/system">System Safety</Link>
            </Button>
          </nav>
        </CardContent>
      </Card>
      <TechnicalDetailsDisclosure>
        <div className="space-y-3">
          <p>
            The operator-evidence overview is the source for the workflow statuses above. Unknown
            data never becomes a completed status.
          </p>
          <p>
            Dataset-backed research, diagnostics, manual trading, specialist guides and testing
            tools remain available in Review. Those areas have not yet been redesigned for the
            current operator workflow.
          </p>
          <p>
            World Monitor input is research context, not an order instruction. Candidate promotion
            remains a separate evidence-gated process.
          </p>
        </div>
      </TechnicalDetailsDisclosure>
    </div>
  );
}
