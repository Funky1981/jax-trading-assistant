import { Activity, AlertCircle, Snowflake, WifiOff } from 'lucide-react';
import { cn } from '@/lib/utils';
import {
  getMarketDataBadgeText,
  getMarketDataTone,
  normalizeMarketDataMode,
} from '@/lib/market-data';

interface DataSourceBadgeProps {
  marketDataMode?: string | null;
  paperTrading?: boolean;
  isError?: boolean;
  className?: string;
}

export function DataSourceBadge({
  marketDataMode,
  paperTrading,
  isError,
  className,
}: DataSourceBadgeProps) {
  if (isError) {
    return (
      <div className={cn(
        "inline-flex items-center gap-1 px-2 py-0.5 rounded-md text-xs font-medium",
        "bg-destructive/10 text-destructive border border-destructive/20",
        className
      )}>
        <WifiOff className="h-3 w-3" />
        <span>UNAVAILABLE</span>
      </div>
    );
  }

  const normalizedMode = normalizeMarketDataMode(marketDataMode);
  const tone = getMarketDataTone(normalizedMode);
  const label = getMarketDataBadgeText(normalizedMode, paperTrading);
  const Icon = normalizedMode === 'live'
    ? Activity
    : normalizedMode === 'frozen'
    ? Snowflake
    : AlertCircle;

  return (
    <div className={cn(
      "inline-flex items-center gap-1 px-2 py-0.5 rounded-md text-xs font-medium",
      tone,
      className
    )}>
      <Icon className={cn("h-3 w-3", normalizedMode === 'live' && "animate-pulse")} />
      <span>{label}</span>
    </div>
  );
}
