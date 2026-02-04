import { CheckCircle, AlertCircle, HelpCircle } from 'lucide-react';
import type { ReactNode } from 'react';
import { Badge } from '@/components/ui/badge';
import { cn } from '@/lib/utils';

type StatusType = 'success' | 'error' | 'warning' | 'info';

interface StatusCardProps {
  title: string;
  status: StatusType;
  statusLabel?: string;
  icon?: ReactNode;
  description?: string;
  compact?: boolean;
}

export function StatusCard({ 
  title, 
  status, 
  statusLabel,
  icon,
  description,
  compact = false,
}: StatusCardProps) {
  const getStatusIcon = () => {
    if (icon) return icon;
    
    switch (status) {
      case 'success':
        return <CheckCircle className="h-5 w-5" />;
      case 'error':
        return <AlertCircle className="h-5 w-5" />;
      case 'warning':
        return <HelpCircle className="h-5 w-5" />;
      default:
        return <HelpCircle className="h-5 w-5" />;
    }
  };

  const getDefaultLabel = () => {
    if (statusLabel) return statusLabel;
    
    switch (status) {
      case 'success':
        return 'Healthy';
      case 'error':
        return 'Down';
      case 'warning':
        return 'Unknown';
      default:
        return 'Unknown';
    }
  };

  const statusColors = {
    success: 'text-emerald-600 border-emerald-600',
    error: 'text-red-600 border-red-600',
    warning: 'text-yellow-600 border-yellow-600',
    info: 'text-blue-600 border-blue-600',
  };

  const badgeVariant = status === 'success' ? 'success' : status === 'error' ? 'destructive' : 'warning';

  return (
    <div className={cn(
      'rounded-lg border-2 bg-muted/50 transition-all hover:bg-muted',
      statusColors[status],
      compact ? 'p-2' : 'p-3'
    )}>
      <div className="flex items-center justify-between gap-3">
        <div className="flex items-center gap-2 min-w-0">
          <div className={cn('flex-shrink-0', statusColors[status])}>
            {getStatusIcon()}
          </div>
          <p className={cn('font-medium truncate', compact ? 'text-sm' : 'text-base')}>
            {title}
          </p>
        </div>
        <Badge variant={badgeVariant} className="flex-shrink-0">
          {getDefaultLabel()}
        </Badge>
      </div>
      {description && (
        <p className={cn('text-muted-foreground mt-1 ml-7', compact ? 'text-xs' : 'text-sm')}>
          {description}
        </p>
      )}
    </div>
  );
}
