import { PositionCard, RiskSummary } from '../components';
import { calculateTotalExposure, calculateTotalUnrealizedPnl } from '../domain/calculations';
import { useDomain } from '../domain/store';
import { selectPositions } from '../domain/selectors';

export function PortfolioPage() {
  const { state } = useDomain();
  const positions = selectPositions(state);
  const exposure = calculateTotalExposure(positions);
  const pnl = calculateTotalUnrealizedPnl(positions);

  return (
    <div className="space-y-4">
      <h1 className="text-3xl font-semibold">Portfolio & Risk</h1>
      <p className="text-sm text-muted-foreground">
        Positions, exposure, and risk thresholds.
      </p>
      <RiskSummary exposure={exposure} pnl={pnl} limits={state.riskLimits} />
      <div className="flex flex-col md:flex-row gap-4">
        {positions.map((position) => (
          <PositionCard key={position.symbol} position={position} />
        ))}
      </div>
    </div>
  );
}
