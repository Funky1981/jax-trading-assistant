import { ReactNode } from 'react';
import { ChevronDown, ChevronRight } from 'lucide-react';
import { cn } from '@/lib/utils';
import { Button } from '@/components/ui/button';
import { Card, CardHeader, CardTitle, CardContent } from '@/components/ui/card';
import { Collapsible, CollapsibleTrigger, CollapsibleContent } from '@/components/ui/collapsible';
import { Skeleton } from '@/components/ui/skeleton';

interface CollapsiblePanelProps {
  title: string;
  icon?: ReactNode;
  summary?: ReactNode;
  children: ReactNode;
  isOpen: boolean;
  onToggle: () => void;
  isLoading?: boolean;
  className?: string;
}

export function CollapsiblePanel({
  title,
  icon,
  summary,
  children,
  isOpen,
  onToggle,
  isLoading = false,
  className,
}: CollapsiblePanelProps) {
  return (
    <Collapsible open={isOpen} onOpenChange={onToggle}>
      <Card className={cn('overflow-hidden', className)}>
        <CollapsibleTrigger asChild>
          <CardHeader className="cursor-pointer select-none hover:bg-muted/50 transition-colors">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-3">
                {icon && <div className="text-muted-foreground">{icon}</div>}
                <CardTitle className="text-base">{title}</CardTitle>
              </div>
              <div className="flex items-center gap-3">
                {!isOpen && summary && (
                  <div className="text-sm text-muted-foreground">{summary}</div>
                )}
                <Button variant="ghost" size="icon" className="h-6 w-6">
                  {isOpen ? (
                    <ChevronDown className="h-4 w-4" />
                  ) : (
                    <ChevronRight className="h-4 w-4" />
                  )}
                </Button>
              </div>
            </div>
          </CardHeader>
        </CollapsibleTrigger>
        <CollapsibleContent>
          <CardContent className="pt-0">
            {isLoading ? (
              <div className="space-y-2">
                <Skeleton className="h-4 w-full" />
                <Skeleton className="h-4 w-3/4" />
                <Skeleton className="h-4 w-1/2" />
              </div>
            ) : (
              children
            )}
          </CardContent>
        </CollapsibleContent>
      </Card>
    </Collapsible>
  );
}

interface StatusDotProps {
  status: 'healthy' | 'degraded' | 'unhealthy' | 'success' | 'warning' | 'error';
  className?: string;
}

export function StatusDot({ status, className }: StatusDotProps) {
  const colorMap = {
    healthy: 'bg-success',
    success: 'bg-success',
    degraded: 'bg-warning',
    warning: 'bg-warning',
    unhealthy: 'bg-destructive',
    error: 'bg-destructive',
  };

  return (
    <div
      className={cn(
        'h-2 w-2 rounded-full',
        colorMap[status],
        className
      )}
    />
  );
}
