import { formatPrice } from '../../domain/market';
import type { Position } from '../../domain/models';
import { calculateUnrealizedPnl } from '../../domain/calculations';
import { PnLIndicator } from './PnLIndicator';
import { Card, CardContent } from '@/components/ui/card';

interface PositionCardProps {
  position: Position;
}

export function PositionCard({ position }: PositionCardProps) {
  const pnl = calculateUnrealizedPnl(position);

  return (
    <Card>
      <CardContent>
        <div className="space-y-2">
          <h3 className="text-sm font-medium">{position.symbol}</h3>
          <p className="text-sm text-muted-foreground">
            Qty {position.quantity} @ {formatPrice(position.avgPrice)}
          </p>
          <div className="flex items-center gap-2">
            <span className="text-sm">Mkt {formatPrice(position.marketPrice)}</span>
            <PnLIndicator value={pnl} />
          </div>
        </div>
      </CardContent>
    </Card>
  );
}
