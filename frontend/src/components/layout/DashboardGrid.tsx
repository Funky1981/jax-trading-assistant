import { ReactNode } from 'react';
import { cn } from '@/lib/utils';

interface DashboardGridProps {
  children: ReactNode;
  className?: string;
}

export function DashboardGrid({ children, className }: DashboardGridProps) {
  return (
    <div
      className={cn(
        'grid gap-4 md:gap-6',
        'grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4',
        className
      )}
    >
      {children}
    </div>
  );
}

interface DashboardPanelProps {
  children: ReactNode;
  className?: string;
  /** Number of columns to span (1-4) */
  colSpan?: 1 | 2 | 3 | 4;
  /** Number of rows to span */
  rowSpan?: 1 | 2;
}

export function DashboardPanel({
  children,
  className,
  colSpan = 1,
  rowSpan = 1,
}: DashboardPanelProps) {
  return (
    <div
      className={cn(
        'min-h-0',
        colSpan === 2 && 'md:col-span-2',
        colSpan === 3 && 'md:col-span-2 lg:col-span-3',
        colSpan === 4 && 'md:col-span-2 lg:col-span-3 xl:col-span-4',
        rowSpan === 2 && 'row-span-2',
        className
      )}
    >
      {children}
    </div>
  );
}
