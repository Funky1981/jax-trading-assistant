import { Activity, AlertCircle, WifiOff } from 'lucide-react';
import { cn } from '@/lib/utils';

interface DataSourceBadgeProps {
  isLive: boolean;
  isError?: boolean;
  className?: string;
}

export function DataSourceBadge({ isLive, isError, className }: DataSourceBadgeProps) {
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
  
  if (isLive) {
    return (
      <div className={cn(
        "inline-flex items-center gap-1 px-2 py-0.5 rounded-md text-xs font-medium",
        "bg-emerald-500/10 text-emerald-500 border border-emerald-500/20",
        className
      )}>
        <Activity className="h-3 w-3 animate-pulse" />
        <span>LIVE DATA</span>
      </div>
    );
  }
  
  return (
    <div className={cn(
      "inline-flex items-center gap-1 px-2 py-0.5 rounded-md text-xs font-medium",
      "bg-yellow-500/10 text-yellow-500 border border-yellow-500/20",
      className
    )}>
      <AlertCircle className="h-3 w-3" />
      <span>SIMULATED</span>
    </div>
  );
}
