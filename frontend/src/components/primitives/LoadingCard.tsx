import { Skeleton } from '@/components/ui/skeleton';
import { Card, CardContent } from '@/components/ui/card';

interface LoadingCardProps {
  rows?: number;
  variant?: 'default' | 'table' | 'metric';
}

export function LoadingCard({ rows = 3, variant = 'default' }: LoadingCardProps) {
  if (variant === 'table') {
    return (
      <Card>
        <CardContent className="pt-6">
          <Skeleton className="h-8 w-2/5 mb-4" />
          <div className="space-y-2">
            {[...Array(rows)].map((_, i) => (
              <div key={i} className="flex gap-3 items-center">
                <Skeleton className="h-5 w-5 rounded-full flex-shrink-0" />
                <Skeleton className="h-4 w-1/3" />
                <Skeleton className="h-6 w-20" />
                <Skeleton className="h-4 w-1/4" />
                <Skeleton className="h-4 w-1/6" />
              </div>
            ))}
          </div>
        </CardContent>
      </Card>
    );
  }

  if (variant === 'metric') {
    return (
      <Card>
        <CardContent className="pt-6">
          <Skeleton className="h-7 w-1/2 mb-4" />
          <div className="space-y-3">
            {[...Array(rows)].map((_, i) => (
              <div key={i}>
                <Skeleton className="h-6 w-3/5 mb-2" />
                <Skeleton className="h-2 w-full rounded-full" />
              </div>
            ))}
          </div>
        </CardContent>
      </Card>
    );
  }

  return (
    <Card>
      <CardContent className="pt-6">
        <Skeleton className="h-8 w-2/5 mb-4" />
        <div className="space-y-2">
          {[...Array(rows)].map((_, i) => (
            <Skeleton key={i} className="h-4 w-full" />
          ))}
        </div>
      </CardContent>
    </Card>
  );
}
