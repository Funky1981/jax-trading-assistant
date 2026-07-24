import { Link } from 'react-router-dom';
import { ArrowRight } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { PageIntro } from '@/components/ui/beginner-help';
import { useOperatorEvidenceOverview } from '@/hooks/useOperatorEvidenceOverview';

export function OutcomesPage() {
  const overview = useOperatorEvidenceOverview();
  const rows = overview.data
    ? [
        ['Completed', overview.data.completedCheckpoints],
        ['Pending', overview.data.pendingCheckpoints],
        ['Missing data', overview.data.missingDataCheckpoints],
        ['Ambiguous', overview.data.ambiguousCheckpoints],
      ]
    : [];
  return (
    <div className="mx-auto max-w-4xl space-y-6">
      <PageIntro
        eyebrow="Outcomes"
        title="Hypothetical outcomes"
        description="Review retrospective paper-plan checkpoints. These calculations are not orders, fills or realised profit and loss."
      />
      <Card>
        <CardHeader>
          <CardTitle>Checkpoint summary</CardTitle>
        </CardHeader>
        <CardContent>
          {overview.isPending ? (
            <p className="text-muted-foreground">Loading persisted outcomes…</p>
          ) : overview.isError || !overview.data ? (
            <p role="alert" className="text-destructive">
              Outcome data is unavailable.
            </p>
          ) : (
            <div className="grid gap-3 sm:grid-cols-2">
              {rows.map(([label, value]) => (
                <div className="rounded-md border p-4" key={label}>
                  <p className="text-sm text-muted-foreground">{label}</p>
                  <p className="text-2xl font-bold">{value}</p>
                </div>
              ))}
            </div>
          )}
        </CardContent>
      </Card>
      <p className="text-sm text-muted-foreground">
        Open a candidate from Candidates to inspect its individual hypothetical checkpoints and
        persisted evidence.
      </p>
      <Button asChild variant="outline">
        <Link to="/etf/approvals">
          Open Candidates
          <ArrowRight className="ml-2 h-4 w-4" />
        </Link>
      </Button>
    </div>
  );
}
