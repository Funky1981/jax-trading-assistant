import { Activity } from 'lucide-react';
import { useHealth } from '@/hooks/useHealth';
import { CollapsiblePanel, StatusDot } from './CollapsiblePanel';
import { Badge } from '@/components/ui/badge';
import { formatTime } from '@/lib/utils';

interface HealthPanelProps {
  isOpen: boolean;
  onToggle: () => void;
}

export function HealthPanel({ isOpen, onToggle }: HealthPanelProps) {
  const { data, isLoading } = useHealth();

  const summary = data && data.services && data.services.length > 0 ? (
    <div className="flex items-center gap-2">
      {data.services.slice(0, 3).map((service) => (
        <StatusDot key={service.name} status={service.status} />
      ))}
      <span className="text-xs">
        {data.services.filter((s) => s.status === 'healthy').length}/{data.services.length} healthy
      </span>
    </div>
  ) : null;

  return (
    <CollapsiblePanel
      title="System Health"
      icon={<Activity className="h-4 w-4" />}
      summary={summary}
      isOpen={isOpen}
      onToggle={onToggle}
      isLoading={isLoading}
    >
      <div className="space-y-3">
        {data?.services && data.services.length > 0 ? data.services.map((service) => (
          <div
            key={service.name}
            className="flex items-center justify-between rounded-md border border-border bg-muted/30 p-3"
          >
            <div className="flex items-center gap-3">
              <StatusDot status={service.status} className="h-3 w-3" />
              <div>
                <p className="text-sm font-medium">{service.name}</p>
                {service.message && (
                  <p className="text-xs text-muted-foreground">{service.message}</p>
                )}
              </div>
            </div>
            <div className="flex items-center gap-3">
              {service.latency && (
                <span className="text-xs text-muted-foreground">
                  {service.latency}ms
                </span>
              )}
              <Badge
                variant={
                  service.status === 'healthy'
                    ? 'success'
                    : service.status === 'degraded'
                    ? 'warning'
                    : 'destructive'
                }
              >
                {service.status}
              </Badge>
            </div>
          </div>
        )) : (
          <div className="text-sm text-muted-foreground text-center py-4">
            No service health data available
          </div>
        )}
        {data && data.services && data.services.length > 0 && (
          <p className="text-xs text-muted-foreground text-right">
            Last checked: {formatTime(data.services[0]?.lastCheck || Date.now())}
          </p>
        )}
      </div>
    </CollapsiblePanel>
  );
}
