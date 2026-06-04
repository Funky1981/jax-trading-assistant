import type { SentimentEvidence } from '@/data/types';
import { Badge } from '@/components/ui/badge';

function pct(value?: number) {
  if (typeof value !== 'number' || Number.isNaN(value)) return 'Not scored';
  return `${Math.round(value * 100)}%`;
}

function signed(value?: number) {
  if (typeof value !== 'number' || Number.isNaN(value)) return 'Unavailable';
  return value > 0 ? `+${value.toFixed(2)}` : value.toFixed(2);
}

const stateLabel: Record<SentimentEvidence['state'], string> = {
  available: 'Available',
  disabled: 'Disabled',
  missing: 'Missing',
  sparse: 'Sparse',
  low_confidence: 'Low confidence',
  degraded: 'Degraded',
  error: 'Error',
};

export function SentimentEvidencePanel({ sentiment, compact = false }: { sentiment?: SentimentEvidence; compact?: boolean }) {
  if (!sentiment) {
    return (
      <div className="rounded-md border border-border bg-muted/20 px-3 py-2 text-sm text-muted-foreground">
        Sentiment evidence is unavailable for this opportunity. Review strategy, policy, price, and risk evidence before acting.
      </div>
    );
  }

  return (
    <div className="rounded-md border border-border bg-muted/20 px-3 py-2 text-sm">
      <div className="flex flex-wrap items-center gap-2">
        <span className="font-medium">Sentiment evidence</span>
        <Badge variant={sentiment.state === 'available' ? 'secondary' : 'outline'}>{stateLabel[sentiment.state]}</Badge>
        <Badge variant="outline">{sentiment.label}</Badge>
        {!compact && <span className="text-muted-foreground">{sentiment.window ?? 'window unknown'}</span>}
      </div>
      <p className="mt-1 text-muted-foreground">
        {sentiment.summary ??
          `${sentiment.label} tone with score ${signed(sentiment.score)}, ${pct(sentiment.confidence)} confidence, and ${
            sentiment.sourceCount ?? 0
          } source(s).`}
      </p>
      {!compact && (
        <>
          <div className="mt-2 grid gap-2 md:grid-cols-3">
            <span>Score: {signed(sentiment.score)}</span>
            <span>Confidence: {pct(sentiment.confidence)}</span>
            <span>Sources: {sentiment.sourceCount ?? 0}</span>
          </div>
          {sentiment.topDrivers && sentiment.topDrivers.length > 0 && (
            <p className="mt-2 text-muted-foreground">Drivers: {sentiment.topDrivers.join(', ')}</p>
          )}
          <p className="mt-2 text-xs text-muted-foreground">
            {sentiment.intendedUse ?? 'Use sentiment as supporting evidence; it does not approve trades or override policy.'}
          </p>
          {(sentiment.limitations ?? []).map((limitation) => (
            <p key={limitation} className="mt-1 text-xs text-muted-foreground">
              {limitation}
            </p>
          ))}
        </>
      )}
    </div>
  );
}
