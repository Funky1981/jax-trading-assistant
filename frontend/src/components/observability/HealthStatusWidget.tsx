import { Box, Card, CardContent, Typography, Chip, Stack, CircularProgress } from '@mui/material';
import { CheckCircle, Error as ErrorIcon } from '@mui/icons-material';
import { useAPIHealth, useMemoryHealth } from '../../hooks/useObservability';

export function HealthStatusWidget() {
  const { data: apiHealth, isLoading: apiLoading } = useAPIHealth();
  const { data: memoryHealth, isLoading: memoryLoading } = useMemoryHealth();

  const renderServiceStatus = (
    name: string,
    isLoading: boolean,
    isHealthy?: boolean,
    timestamp?: string,
  ) => {
    if (isLoading) {
      return (
        <Stack direction="row" spacing={1} alignItems="center">
          <CircularProgress size={16} />
          <Typography variant="body2">{name}</Typography>
        </Stack>
      );
    }

    const healthy = isHealthy ?? false;
    return (
      <Stack direction="row" spacing={1} alignItems="center" justifyContent="space-between">
        <Stack direction="row" spacing={1} alignItems="center">
          {healthy ? (
            <CheckCircle sx={{ color: 'success.main', fontSize: 20 }} />
          ) : (
            <ErrorIcon sx={{ color: 'error.main', fontSize: 20 }} />
          )}
          <Typography variant="body2">{name}</Typography>
        </Stack>
        <Chip
          label={healthy ? 'Healthy' : 'Unhealthy'}
          color={healthy ? 'success' : 'error'}
          size="small"
        />
      </Stack>
    );
  };

  return (
    <Card>
      <CardContent>
        <Typography variant="h6" gutterBottom>
          Backend Health
        </Typography>
        <Stack spacing={2}>
          {renderServiceStatus('JAX API', apiLoading, apiHealth?.healthy, apiHealth?.timestamp)}
          {renderServiceStatus(
            'Memory Service',
            memoryLoading,
            memoryHealth?.healthy,
            memoryHealth?.timestamp,
          )}
        </Stack>
        {(apiHealth?.timestamp || memoryHealth?.timestamp) && (
          <Typography variant="caption" color="text.secondary" sx={{ mt: 2, display: 'block' }}>
            Last check: {new Date(apiHealth?.timestamp || memoryHealth?.timestamp || '').toLocaleTimeString()}
          </Typography>
        )}
      </CardContent>
    </Card>
  );
}
