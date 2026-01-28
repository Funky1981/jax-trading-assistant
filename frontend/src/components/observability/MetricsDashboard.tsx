import { useState } from 'react';
import {
  Box,
  Card,
  CardContent,
  Typography,
  Stack,
  Chip,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableRow,
  CircularProgress,
  Alert,
} from '@mui/material';
import { TrendingUp, CheckCircle, Error as ErrorIcon } from '@mui/icons-material';
import { useRecentMetrics, useRunMetrics } from '../../hooks/useObservability';
import type { MetricEvent } from '../../data/types';

export function MetricsDashboard() {
  const [selectedRunId, setSelectedRunId] = useState<string | null>(null);
  const { data: recentMetrics, isLoading: recentLoading, error: recentError } = useRecentMetrics();
  const { data: runMetrics, isLoading: runLoading } = useRunMetrics(selectedRunId || null);

  if (recentLoading) {
    return (
      <Card>
        <CardContent>
          <Stack alignItems="center" spacing={2}>
            <CircularProgress />
            <Typography>Loading metrics...</Typography>
          </Stack>
        </CardContent>
      </Card>
    );
  }

  if (recentError) {
    return (
      <Card>
        <CardContent>
          <Alert severity="error">Failed to load metrics</Alert>
        </CardContent>
      </Card>
    );
  }

  const getEventIcon = (event: string) => {
    if (event.includes('success') || event.includes('completed')) {
      return <CheckCircle sx={{ color: 'success.main', fontSize: 18 }} />;
    }
    if (event.includes('error') || event.includes('failed')) {
      return <ErrorIcon sx={{ color: 'error.main', fontSize: 18 }} />;
    }
    return <TrendingUp sx={{ color: 'info.main', fontSize: 18 }} />;
  };

  return (
    <Card>
      <CardContent>
        <Typography variant="h6" gutterBottom>
          Recent Metrics
        </Typography>
        {recentMetrics && recentMetrics.length > 0 ? (
          <Table size="small">
            <TableHead>
              <TableRow>
                <TableCell>Event</TableCell>
                <TableCell>Source</TableCell>
                <TableCell>Run ID</TableCell>
                <TableCell>Duration</TableCell>
                <TableCell>Time</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {recentMetrics.map((metric: MetricEvent, idx: number) => (
                <TableRow
                  key={idx}
                  hover
                  sx={{ cursor: metric.run_id ? 'pointer' : 'default' }}
                  onClick={() => metric.run_id && setSelectedRunId(metric.run_id)}
                >
                  <TableCell>
                    <Stack direction="row" spacing={1} alignItems="center">
                      {getEventIcon(metric.event)}
                      <Typography variant="body2">{metric.event}</Typography>
                    </Stack>
                  </TableCell>
                  <TableCell>
                    <Chip label={metric.source} size="small" variant="outlined" />
                  </TableCell>
                  <TableCell>
                    <Typography variant="caption" fontFamily="monospace">
                      {metric.run_id?.slice(0, 8) || '-'}
                    </Typography>
                  </TableCell>
                  <TableCell>
                    {metric.latency_ms ? `${metric.latency_ms}ms` : '-'}
                  </TableCell>
                  <TableCell>
                    <Typography variant="caption">
                      {new Date(metric.timestamp).toLocaleTimeString()}
                    </Typography>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        ) : (
          <Typography color="text.secondary">No recent metrics</Typography>
        )}

        {selectedRunId && runLoading && (
          <Box sx={{ mt: 2 }}>
            <CircularProgress size={20} />
          </Box>
        )}

        {selectedRunId && runMetrics && (
          <Box sx={{ mt: 2 }}>
            <Typography variant="subtitle2">Run Details: {selectedRunId.slice(0, 8)}</Typography>
            <Typography variant="caption" color="text.secondary">
              {runMetrics.length} events
            </Typography>
          </Box>
        )}
      </CardContent>
    </Card>
  );
}
