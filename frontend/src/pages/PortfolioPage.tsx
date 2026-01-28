import { Stack, Typography } from '@mui/material';
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
    <Stack spacing={2}>
      <Typography variant="h4">Portfolio & Risk</Typography>
      <Typography variant="body2" color="text.secondary">
        Positions, exposure, and risk thresholds.
      </Typography>
      <RiskSummary exposure={exposure} pnl={pnl} limits={state.riskLimits} />
      <Stack direction={{ xs: 'column', md: 'row' }} spacing={2}>
        {positions.map((position) => (
          <PositionCard key={position.symbol} position={position} />
        ))}
      </Stack>
    </Stack>
  );
}
