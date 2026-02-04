import { useState } from 'react';
import { Timeline, CheckCircle, XCircle, Info } from 'lucide-react';
import { useRecentMetrics, useRunMetrics } from '../../hooks/useObservability';
import type { MetricEvent } from '../../data/types';
import { Card, CardContent } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Skeleton } from '@/components/ui/skeleton';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';

export function MetricsDashboard() {
  const [selectedRunId, setSelectedRunId] = useState<string | null>(null);
  const { data: recentMetrics, isLoading: recentLoading, error: recentError } = useRecentMetrics();
  const { data: runMetrics, isLoading: runLoading } = useRunMetrics(selectedRunId || null);

  if (recentLoading) {
    return (
      <Card>
        <CardContent className="pt-6">
          <div className="flex items-center gap-2 mb-5">
            <Timeline className="h-5 w-5 text-primary" />
            <h2 className="text-lg font-semibold">
              Recent Metrics
            </h2>
          </div>
          <div className="space-y-2">
            {[...Array(5)].map((_, i) => (
              <div key={i} className="flex gap-4 items-center">
                <Skeleton className="h-5 w-5 rounded-full" />
                <Skeleton className="h-4 w-[30%]" />
                <Skeleton className="h-6 w-20 rounded-md" />
                <Skeleton className="h-4 w-[15%]" />
                <Skeleton className="h-4 w-[10%]" />
              </div>
            ))}
          </div>
        </CardContent>
      </Card>
    );
  }

  if (recentError) {
    return (
      <Card>
        <CardContent className="pt-6">
          <div className="flex items-center gap-2 mb-4">
            <Timeline className="h-5 w-5 text-primary" />
            <h2 className="text-lg font-semibold">
              Recent Metrics
            </h2>
          </div>
          <div className="rounded-md border border-destructive bg-destructive/10 p-4">
            <p className="text-sm text-destructive">
              Failed to load metrics. Please check your connection.
            </p>
          </div>
        </CardContent>
      </Card>
    );
  }

  const getEventIcon = (event: string) => {
    if (event.includes('success') || event.includes('completed')) {
      return <CheckCircle className="h-[18px] w-[18px] text-green-500" />;
    }
    if (event.includes('error') || event.includes('failed')) {
      return <XCircle className="h-[18px] w-[18px] text-red-500" />;
    }
    return <Info className="h-[18px] w-[18px] text-blue-500" />;
  };

  return (
    <Card>
      <CardContent className="pt-6">
        <div className="flex items-center gap-2 mb-5">
          <Timeline className="h-5 w-5 text-primary" />
          <h2 className="text-lg font-semibold">
            Recent Metrics
          </h2>
        </div>
        {recentMetrics && recentMetrics.length > 0 ? (
          <div className="max-h-[400px] overflow-auto">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Event</TableHead>
                  <TableHead>Source</TableHead>
                  <TableHead>Run ID</TableHead>
                  <TableHead>Duration</TableHead>
                  <TableHead>Time</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {recentMetrics.map((metric: MetricEvent, idx: number) => (
                  <TableRow
                    key={idx}
                    className={metric.run_id ? 'cursor-pointer' : ''}
                    onClick={() => metric.run_id && setSelectedRunId(metric.run_id)}
                  >
                    <TableCell>
                      <div className="flex items-center gap-2">
                        {getEventIcon(metric.event)}
                        <span className="text-sm">{metric.event}</span>
                      </div>
                    </TableCell>
                    <TableCell>
                      <Badge variant="outline">{metric.source}</Badge>
                    </TableCell>
                    <TableCell>
                      <span className="font-mono text-xs">
                        {metric.run_id?.slice(0, 8) || '-'}
                      </span>
                    </TableCell>
                    <TableCell>
                      {metric.latency_ms ? `${metric.latency_ms}ms` : '-'}
                    </TableCell>
                    <TableCell>
                      <span className="text-xs">
                        {new Date(metric.timestamp).toLocaleTimeString()}
                      </span>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        ) : (
          <div className="py-16 text-center">
            <p className="text-muted-foreground">No recent metrics available</p>
          </div>
        )}

        {selectedRunId && runLoading && (
          <div className="mt-4">
            <div className="h-5 w-5 animate-spin rounded-full border-2 border-primary border-t-transparent" />
          </div>
        )}

        {selectedRunId && runMetrics && (
          <div className="mt-4">
            <h3 className="text-sm font-medium">Run Details: {selectedRunId.slice(0, 8)}</h3>
            <p className="text-xs text-muted-foreground">
              {runMetrics.length} events
            </p>
          </div>
        )}
      </CardContent>
    </Card>
  );
}
