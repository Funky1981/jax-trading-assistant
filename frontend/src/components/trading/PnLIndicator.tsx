import { Chip } from '@mui/material';
import { tokens } from '../../styles/tokens';

interface PnLIndicatorProps {
  value: number;
}

export function PnLIndicator({ value }: PnLIndicatorProps) {
  const isPositive = value >= 0;
  const label = `${isPositive ? '+' : ''}${value.toFixed(2)}`;

  return (
    <Chip
      label={label}
      size="small"
      sx={{
        backgroundColor: 'transparent',
        border: `1px solid ${tokens.colors.border}`,
        color: isPositive ? tokens.colors.positive : tokens.colors.negative,
        fontWeight: tokens.typography.weight.semibold,
      }}
    />
  );
}
