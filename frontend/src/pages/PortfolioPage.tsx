import { Stack, Typography } from '@mui/material';
import { PositionCard, RiskSummary } from '../components';
import type { Position } from '../domain/models';
import { calculateTotalExposure, calculateTotalUnrealizedPnl } from '../domain/calculations';
import { defaultRiskLimits } from '../domain/state';

const positions: Position[] = [
  { symbol: 'AAPL', quantity: 250, avgPrice: 231.12, marketPrice: 249.42 },
  { symbol: 'MSFT', quantity: 120, avgPrice: 402.55, marketPrice: 413.1 },
];

const exposure = calculateTotalExposure(positions);
const pnl = calculateTotalUnrealizedPnl(positions);

export function PortfolioPage() {
  return (
    <Stack spacing={2}>
      <Typography variant="h4">Portfolio & Risk</Typography>
      <Typography variant="body2" color="text.secondary">
        Positions, exposure, and risk thresholds.
      </Typography>
      <RiskSummary exposure={exposure} pnl={pnl} limits={defaultRiskLimits} />
      <Stack direction={{ xs: 'column', md: 'row' }} spacing={2}>
        {positions.map((position) => (
          <PositionCard key={position.symbol} position={position} />
        ))}
      </Stack>
    </Stack>
  );
}
