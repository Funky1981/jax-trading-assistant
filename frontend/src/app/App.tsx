import { Box, Chip, Stack, Typography } from '@mui/material';

const stats = [
  { label: 'AAPL', value: '249.42', change: '+0.31%' },
  { label: 'MSFT', value: '413.10', change: '-0.18%' },
  { label: 'SPY', value: '541.22', change: '+0.05%' },
];

export default function App() {
  return (
    <Box
      sx={{
        minHeight: '100vh',
        backgroundColor: 'var(--color-bg)',
        color: 'var(--color-text)',
        padding: 3,
      }}
    >
      <Stack spacing={3}>
        <Stack spacing={1}>
          <Typography variant="overline" sx={{ letterSpacing: 2 }}>
            JAX TRADING UI
          </Typography>
          <Typography variant="h4" sx={{ fontWeight: 600 }}>
            Market Overview
          </Typography>
          <Typography variant="body2" color="text.secondary">
            Live pricing and portfolio snapshots will stream here.
          </Typography>
        </Stack>

        <Stack direction="row" spacing={1} flexWrap="wrap">
          {stats.map((item) => (
            <Chip
              key={item.label}
              label={`${item.label} ${item.value} (${item.change})`}
              size="small"
              sx={{
                backgroundColor: 'var(--color-surface)',
                color: 'var(--color-text)',
                border: '1px solid var(--color-border)',
              }}
            />
          ))}
        </Stack>

        <Box
          sx={{
            backgroundColor: 'var(--color-surface)',
            border: '1px solid var(--color-border)',
            borderRadius: 2,
            padding: 3,
          }}
        >
          <Typography variant="subtitle2" sx={{ marginBottom: 1 }}>
            Getting Started
          </Typography>
          <Typography variant="body2" color="text.secondary">
            This is a minimal app shell. Next steps: wire data streams, domain models,
            and the component library build-out.
          </Typography>
        </Box>
      </Stack>
    </Box>
  );
}
