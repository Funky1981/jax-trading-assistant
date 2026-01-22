import { Box, LinearProgress, Stack, Typography } from '@mui/material';
import type { RiskLimits } from '../../domain/models';
import { tokens } from '../../styles/tokens';

interface RiskSummaryProps {
  exposure: number;
  pnl: number;
  limits: RiskLimits;
}

export function RiskSummary({ exposure, pnl, limits }: RiskSummaryProps) {
  const exposureRatio = Math.min(exposure / limits.maxPositionValue, 1);
  const lossRatio = Math.min(Math.abs(pnl) / limits.maxDailyLoss, 1);
  const pnlLabel = pnl >= 0 ? `+${pnl.toFixed(2)}` : pnl.toFixed(2);

  return (
    <Box
      sx={{
        padding: tokens.spacing.lg,
        borderRadius: tokens.radius.md,
        border: `1px solid ${tokens.colors.border}`,
        backgroundColor: tokens.colors.surface,
      }}
    >
      <Stack spacing={2}>
        <Stack>
          <Typography variant="subtitle2">Exposure</Typography>
          <Typography variant="body2" color="text.secondary">
            {exposure.toFixed(0)} / {limits.maxPositionValue.toFixed(0)}
          </Typography>
          <LinearProgress variant="determinate" value={exposureRatio * 100} />
        </Stack>
        <Stack>
          <Typography variant="subtitle2">Daily PnL</Typography>
          <Typography variant="body2" color="text.secondary">
            {pnlLabel} / -{limits.maxDailyLoss.toFixed(0)}
          </Typography>
          <LinearProgress
            variant="determinate"
            value={lossRatio * 100}
            color={pnl >= 0 ? 'success' : 'warning'}
          />
        </Stack>
      </Stack>
    </Box>
  );
}
