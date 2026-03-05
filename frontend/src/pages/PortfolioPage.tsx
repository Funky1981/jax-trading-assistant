import { PositionsPanel, RiskSummaryPanel } from '@/components/dashboard';
import { HelpHint } from '@/components/ui/help-hint';

export function PortfolioPage() {
  return (
    <div className="space-y-4">
      <h1 className="flex items-center gap-2 text-3xl font-semibold">
        Portfolio & Risk
        <HelpHint text="Review exposure, unrealized P/L, and risk thresholds." />
      </h1>
      <p className="text-sm text-muted-foreground">
        Snapshot of positions, exposure, and risk limits.
      </p>
      <div className="space-y-4">
        <RiskSummaryPanel isOpen onToggle={() => undefined} />
        <PositionsPanel isOpen onToggle={() => undefined} />
      </div>
    </div>
  );
}
