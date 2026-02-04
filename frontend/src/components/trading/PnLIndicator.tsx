import { Badge } from '@/components/ui/badge';
import { tokens } from '../../styles/tokens';

interface PnLIndicatorProps {
  value: number;
  suffix?: string;
}

export function PnLIndicator({ value, suffix = '' }: PnLIndicatorProps) {
  const isPositive = value >= 0;
  const label = `${isPositive ? '+' : ''}${value.toFixed(2)}${suffix}`;

  return (
    <Badge
      variant="outline"
      className={`font-semibold ${
        isPositive ? 'text-green-500 border-green-500' : 'text-red-500 border-red-500'
      }`}
    >
      {label}
    </Badge>
  );
}
