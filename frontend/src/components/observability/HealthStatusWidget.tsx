import { CheckCircle, XCircle, HelpCircle, Computer } from 'lucide-react';
import { useAPIHealth, useMemoryHealth } from '../../hooks/useObservability';
import { Card, CardContent } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Skeleton } from '@/components/ui/skeleton';

export function HealthStatusWidget() {
  const { data: apiHealth, isLoading: apiLoading } = useAPIHealth();
  const { data: memoryHealth, isLoading: memoryLoading } = useMemoryHealth();

  const renderServiceStatus = (
    name: string,
    isLoading: boolean,
    isHealthy?: boolean,
  ) => {
    if (isLoading) {
      return (
        <div className="p-4 rounded-md bg-muted border border-border">
          <div className="flex items-center justify-between gap-4">
            <div className="flex items-center gap-3">
              <div className="h-5 w-5 animate-spin rounded-full border-2 border-primary border-t-transparent" />
              <span className="text-sm font-medium">{name}</span>
            </div>
            <Skeleton className="h-6 w-20" />
          </div>
        </div>
      );
    }

    // Handle three states: healthy, unhealthy, unknown
    let statusColor: 'default' | 'destructive' | 'secondary' = 'secondary';
    let statusLabel = 'Unknown';
    let statusIcon = <HelpCircle className="h-5 w-5" />;
    let borderColor = 'border-border';

    if (isHealthy === true) {
      statusColor = 'default';
      statusLabel = 'Healthy';
      statusIcon = <CheckCircle className="h-5 w-5 text-green-500" />;
      borderColor = 'border-green-500';
    } else if (isHealthy === false) {
      statusColor = 'destructive';
      statusLabel = 'Down';
      statusIcon = <XCircle className="h-5 w-5 text-red-500" />;
      borderColor = 'border-red-500';
    }

    return (
      <div
        className={`p-4 rounded-md bg-muted border-2 ${borderColor} transition-all hover:bg-accent`}
      >
        <div className="flex items-center justify-between gap-4">
          <div className="flex items-center gap-3">
            {statusIcon}
            <span className="text-sm font-medium">{name}</span>
          </div>
          <Badge variant={statusColor} className="min-w-[80px] justify-center font-semibold">
            {statusLabel}
          </Badge>
        </div>
      </div>
    );
  };

  return (
    <Card>
      <CardContent className="pt-6">
        <div className="flex items-center gap-2 mb-5">
          <Computer className="h-5 w-5 text-primary" />
          <h2 className="text-lg font-semibold">
            Backend Health
          </h2>
        </div>
        <div className="space-y-3">
          {renderServiceStatus('JAX API', apiLoading, apiHealth?.healthy)}
          {renderServiceStatus(
            'Memory Service',
            memoryLoading,
            memoryHealth?.healthy,
          )}
        </div>
        {(apiHealth?.timestamp || memoryHealth?.timestamp) && (
          <p className="text-xs text-muted-foreground mt-5">
            Last check: {new Date(apiHealth?.timestamp || memoryHealth?.timestamp || '').toLocaleTimeString()}
          </p>
        )}
      </CardContent>
    </Card>
  );
}
