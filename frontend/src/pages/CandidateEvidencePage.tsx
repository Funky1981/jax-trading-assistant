import { useMemo } from 'react';
import { Link, useParams } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { ArrowLeft } from 'lucide-react';
import { candidatesService } from '@/data/approvals-service';
import { CandidateTradeSummary, type CandidateTradeData } from '@/components/trading/CandidateTradeSummary';
import { useBeginnerMode } from '@/context/BeginnerUXContext';
import { Button } from '@/components/ui/button';

function readMetadataNumber(metadata: Record<string, unknown> | undefined, keys: string[]): number | undefined {
  if (!metadata) return undefined;
  for (const key of keys) {
    const value = metadata[key];
    if (typeof value === 'number' && Number.isFinite(value)) {
      return value;
    }
  }
  return undefined;
}

function readMetadataString(metadata: Record<string, unknown> | undefined, keys: string[]): string | undefined {
  if (!metadata) return undefined;
  for (const key of keys) {
    const value = metadata[key];
    if (typeof value === 'string' && value.trim() !== '') {
      return value;
    }
  }
  return undefined;
}

function readMetadataStringArray(metadata: Record<string, unknown> | undefined, key: string): string[] | undefined {
  if (!metadata) return undefined;
  const value = metadata[key];
  if (!Array.isArray(value)) return undefined;
  const items = value.filter((item): item is string => typeof item === 'string' && item.trim() !== '');
  return items.length > 0 ? items : undefined;
}

function toPricedInPercent(value: number | undefined): number | undefined {
  if (value == null) return undefined;
  if (value <= 1 && value >= 0) return Math.round(value * 100);
  if (value >= 0 && value <= 100) return Math.round(value);
  return undefined;
}

export function CandidateEvidencePage() {
  const { candidateId } = useParams<{ candidateId: string }>();
  const { mode } = useBeginnerMode();

  const candidateQuery = useQuery({
    queryKey: ['candidate', candidateId],
    queryFn: () => candidatesService.get(candidateId ?? ''),
    enabled: Boolean(candidateId),
  });

  const data = useMemo<CandidateTradeData | null>(() => {
    const candidate = candidateQuery.data;
    if (!candidate) return null;

    const metadata = candidate.metadata;
    const pricedInRaw = readMetadataNumber(metadata, ['pricedInScore', 'priced_in_score', 'pricedIn']);

    return {
      candidate,
      newsHeadline: readMetadataString(metadata, ['newsHeadline', 'headline', 'eventTitle']),
      newsSource: readMetadataString(metadata, ['newsSource', 'source']),
      pricedInVerdictScore: toPricedInPercent(pricedInRaw),
      pricedInExplanation: readMetadataString(metadata, ['pricedInExplanation', 'pricedInReason', 'priced_in_reason']),
      confounders: readMetadataStringArray(metadata, 'confounders'),
      conflictingNews: readMetadataStringArray(metadata, 'conflictingNews'),
      riskControlsSummary: readMetadataString(metadata, ['riskControlsSummary', 'risk_summary']),
    };
  }, [candidateQuery.data]);

  if (!candidateId) {
    return <div className="p-6">Missing candidate id.</div>;
  }

  if (candidateQuery.isLoading) {
    return <div className="p-6">Loading candidate evidence...</div>;
  }

  if (candidateQuery.isError || !data) {
    return (
      <div className="space-y-4 p-6">
        <p className="text-sm text-destructive">Failed to load candidate evidence.</p>
        <Button asChild variant="outline">
          <Link to="/approvals">
            <ArrowLeft className="mr-2 h-4 w-4" /> Back to approvals
          </Link>
        </Button>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <Button asChild variant="outline" size="sm">
        <Link to="/approvals">
          <ArrowLeft className="mr-2 h-4 w-4" /> Back to approvals
        </Link>
      </Button>
      <CandidateTradeSummary data={data} mode={mode} />
    </div>
  );
}
