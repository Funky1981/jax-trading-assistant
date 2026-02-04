import { Inbox } from 'lucide-react';
import type { ReactNode } from 'react';
import { cn } from '@/lib/utils';

interface EmptyStateProps {
  icon?: ReactNode;
  title: string;
  description?: string;
  action?: ReactNode;
  compact?: boolean;
}

export function EmptyState({ 
  icon, 
  title, 
  description, 
  action,
  compact = false,
}: EmptyStateProps) {
  return (
    <div className={cn('flex flex-col items-center justify-center text-center', compact ? 'py-8 px-4' : 'py-12 px-4')}>
      <div className="flex items-center justify-center text-muted-foreground mb-4">
        {icon || <Inbox className={compact ? 'h-10 w-10' : 'h-12 w-12'} />}
      </div>
      <div className="space-y-1 mb-4">
        <h3 className={cn('font-semibold text-muted-foreground', compact ? 'text-base' : 'text-lg')}>
          {title}
        </h3>
        {description && (
          <p className="text-sm text-muted-foreground">
            {description}
          </p>
        )}
      </div>
      {action && <div className="pt-2">{action}</div>}
    </div>
  );
}
