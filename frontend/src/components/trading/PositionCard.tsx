import { Card, CardContent, Stack, Typography } from '@mui/material';
import { formatPrice } from '../../domain/market';
import type { Position } from '../../domain/models';
import { calculateUnrealizedPnl } from '../../domain/calculations';
import { PnLIndicator } from './PnLIndicator';

interface PositionCardProps {
  position: Position;
}

export function PositionCard({ position }: PositionCardProps) {
  const pnl = calculateUnrealizedPnl(position);

  return (
    <Card variant="outlined">
      <CardContent>
        <Stack spacing={1}>
          <Typography variant="subtitle2">{position.symbol}</Typography>
          <Typography variant="body2" color="text.secondary">
            Qty {position.quantity} @ {formatPrice(position.avgPrice)}
          </Typography>
          <Stack direction="row" spacing={1} alignItems="center">
            <Typography variant="body2">Mkt {formatPrice(position.marketPrice)}</Typography>
            <PnLIndicator value={pnl} />
          </Stack>
        </Stack>
      </CardContent>
    </Card>
  );
}
