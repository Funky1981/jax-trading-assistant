import { useEffect } from 'react';
import { Link } from 'react-router-dom';
import { ArrowRight, BookOpen, Inbox } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { PageIntro, SafetyBanner, TechnicalDetailsDisclosure } from '@/components/ui/beginner-help';
import { useOperatorEvidenceOverview } from '@/hooks/useOperatorEvidenceOverview';
import { isPaperSafe } from '@/lib/operator-safety';
import { emitAnalyticsEvent } from '@/lib/analytics';

export function HomePage() {
  const overview = useOperatorEvidenceOverview();
  const safe = isPaperSafe(overview.data);
  useEffect(() => {
    emitAnalyticsEvent('page_viewed', { source_surface: 'home' });
  }, []);

  const cards = [
    {
      title: 'New evidence',
      value: overview.data ? overview.data.genuineEvents : undefined,
      description: 'Genuine evidence currently stored for review.',
      path: '/monitor/inbox',
      link: 'Open Evidence Inbox',
    },
    {
      title: 'Candidates',
      value: overview.data ? overview.data.candidates : undefined,
      description: 'Structured ideas that still require human judgement.',
      path: '/etf/approvals',
      link: 'Open Candidates',
    },
    {
      title: 'Hypothetical outcomes',
      value: overview.data ? overview.data.completedCheckpoints : undefined,
      description: 'Completed retrospective checkpoints. These are not fills or realised returns.',
      path: '/outcomes',
      link: 'Open Outcomes',
    },
    {
      title: 'System safety',
      value: overview.isPending ? undefined : safe ? 'Paper-safe' : 'Needs attention',
      description: 'The current runtime and execution safety state.',
      path: '/system',
      link: 'Open System Safety',
    },
  ];

  return (
    <div className="mx-auto flex w-full max-w-6xl flex-col gap-6">
      <PageIntro
        eyebrow="Home"
        title="Jax overview"
        description="Review what Jax has received, what it decided and what happened to hypothetical paper plans."
      />
      <SafetyBanner safe={safe} loading={overview.isPending} />
      {overview.isError && (
        <p
          role="alert"
          className="rounded-md border border-destructive/50 p-3 text-sm text-destructive"
        >
          Operator evidence is unavailable. Counts and safety cannot be confirmed.
        </p>
      )}
      <section aria-label="Current activity" className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        {cards.map((card) => (
          <Card key={card.title} className="flex flex-col">
            <CardHeader>
              <CardTitle className="text-base">{card.title}</CardTitle>
            </CardHeader>
            <CardContent className="flex flex-1 flex-col">
              <p className="text-2xl font-bold">
                {card.value ?? (overview.isError ? 'Unavailable' : 'Checking')}
              </p>
              <p className="mt-2 flex-1 text-sm text-muted-foreground">{card.description}</p>
              <Button asChild variant="link" className="mt-3 h-auto justify-start p-0">
                <Link to={card.path}>
                  {card.link}
                  <ArrowRight className="ml-1 h-4 w-4" />
                </Link>
              </Button>
            </CardContent>
          </Card>
        ))}
      </section>
      <section aria-label="Next actions" className="flex flex-col gap-3 sm:flex-row">
        <Button asChild>
          <Link to="/guide">
            <BookOpen className="mr-2 h-4 w-4" />
            Start the guide
          </Link>
        </Button>
        <Button asChild variant="outline">
          <Link to="/monitor/inbox">
            <Inbox className="mr-2 h-4 w-4" />
            Open Evidence Inbox
          </Link>
        </Button>
      </section>
      <TechnicalDetailsDisclosure>
        <p>
          Counts and safety come from the authenticated operator-evidence overview. Home is
          read-only and cannot approve candidates, create orders or change runtime settings.
        </p>
      </TechnicalDetailsDisclosure>
    </div>
  );
}
