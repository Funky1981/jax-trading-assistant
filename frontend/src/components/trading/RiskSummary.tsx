import { TrendingUp, Landmark } from 'lucide-react';
import type { RiskLimits } from '../../domain/models';
import { Card, CardContent } from '@/components/ui/card';
import { Progress } from '@/components/ui/progress';

interface RiskSummaryProps {
  exposure: number;
  pnl: number;
  limits: RiskLimits;
}

export function RiskSummary({ exposure, pnl, limits }: RiskSummaryProps) {
  const exposureRatio = Math.min(exposure / limits.maxPositionValue, 1);
  const lossRatio = Math.min(Math.abs(pnl) / limits.maxDailyLoss, 1);
  const pnlLabel = pnl >= 0 ? `+$${pnl.toFixed(2)}` : `-$${Math.abs(pnl).toFixed(2)}`;

  const getExposureColor = () => {
    if (exposureRatio > 0.9) return 'bg-red-500';
    if (exposureRatio > 0.7) return 'bg-yellow-500';
    return 'bg-blue-500';
  };

  const getPnLColor = () => {
    if (pnl >= 0) return 'bg-green-500';
    if (lossRatio > 0.8) return 'bg-red-500';
    return 'bg-yellow-500';
  };

  return (
    <Card>
      <CardContent className="pt-6">
        <div className="space-y-6">
          <div className="space-y-4">
            <div className="flex items-center gap-2">
              <Landmark className="h-5 w-5 text-muted-foreground" />
              <h3 className="text-sm font-semibold">Exposure</h3>
            </div>
            <div>
              <div className="flex justify-between mb-2">
                <span className="text-xl font-semibold">
                  ${exposure.toFixed(0)}
                </span>
                <span className="text-sm text-muted-foreground">
                  of ${limits.maxPositionValue.toFixed(0)}
                </span>
              </div>
              <Progress 
                value={exposureRatio * 100} 
                className={`h-2 ${getExposureColor()}`}
              />
              <p className="text-xs text-muted-foreground mt-1">
                {(exposureRatio * 100).toFixed(1)}% utilized
              </p>
            </div>
          </div>

          <div className="space-y-4">
            <div className="flex items-center gap-2">
              <TrendingUp className={`h-5 w-5 ${pnl >= 0 ? 'text-green-500' : 'text-red-500'}`} />
              <h3 className="text-sm font-semibold">Daily P&L</h3>
            </div>
            <div>
              <div className="flex justify-between mb-2">
                <span 
                  className={`text-xl font-semibold ${pnl >= 0 ? 'text-green-500' : 'text-red-500'}`}
                >
                  {pnlLabel}
                </span>
                <span className="text-sm text-muted-foreground">
                  limit ${limits.maxDailyLoss.toFixed(0)}
                </span>
              </div>
              <Progress
                value={lossRatio * 100}
                className={`h-2 ${getPnLColor()}`}
              />
              <p className="text-xs text-muted-foreground mt-1">
                {pnl >= 0 ? 'Profitable' : `${(lossRatio * 100).toFixed(1)}% of loss limit`}
              </p>
            </div>
          </div>
        </div>
      </CardContent>
    </Card>
  );
}
