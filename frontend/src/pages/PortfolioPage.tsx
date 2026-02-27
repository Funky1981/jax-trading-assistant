import { PositionCard, RiskSummary } from '../components';
import { calculateTotalExposure, calculateTotalUnrealizedPnl } from '../domain/calculations';
import { useDomain } from '../domain/store';
import { selectPositions } from '../domain/selectors';
import { HelpHint } from '@/components/ui/help-hint';

export function PortfolioPage() {
  const { state } = useDomain();
  const positions = selectPositions(state);
  const exposure = calculateTotalExposure(positions);
  const pnl = calculateTotalUnrealizedPnl(positions);

  return (
    <div className="space-y-4">
      <h1 className="flex items-center gap-2 text-3xl font-semibold">
        Portfolio & Risk
        <HelpHint text="Review exposure, unrealized P/L, and risk thresholds." />
      </h1>
      <p className="text-sm text-muted-foreground">
        Snapshot of positions, exposure, and risk limits.
      </p>
      <RiskSummary exposure={exposure} pnl={pnl} limits={state.riskLimits} />
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        {positions.map((position) => (
          <PositionCard key={position.symbol} position={position} />
        ))}
      </div>
    </div>
  );
}
