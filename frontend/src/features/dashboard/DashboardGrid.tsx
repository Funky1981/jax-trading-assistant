import { Box, Paper, Stack, Typography } from '@mui/material';
import { DataTable, OrderTicket, PositionCard, RiskSummary } from '../../components';
import { calculateTotalExposure, calculateTotalUnrealizedPnl } from '../../domain/calculations';
import { defaultRiskLimits } from '../../domain/state';
import type { Position } from '../../domain/models';
import { tokens } from '../../styles/tokens';
import type { DashboardLayout, WidgetLayout } from './layouts';
import { getWidgetById } from './registry';

const mockPositions: Position[] = [
  { symbol: 'AAPL', quantity: 250, avgPrice: 231.12, marketPrice: 249.42 },
  { symbol: 'MSFT', quantity: 120, avgPrice: 402.55, marketPrice: 413.1 },
];

function renderWidget(widget: WidgetLayout) {
  switch (widget.id) {
    case 'order-ticket':
      return <OrderTicket symbol="AAPL" />;
    case 'positions':
      return (
        <Stack spacing={2}>
          {mockPositions.map((position) => (
            <PositionCard key={position.symbol} position={position} />
          ))}
        </Stack>
      );
    case 'risk-summary':
      return (
        <RiskSummary
          exposure={calculateTotalExposure(mockPositions)}
          pnl={calculateTotalUnrealizedPnl(mockPositions)}
          limits={defaultRiskLimits}
        />
      );
    case 'blotter':
      return (
        <DataTable
          columns={[
            { key: 'symbol', label: 'Symbol' },
            { key: 'quantity', label: 'Qty', align: 'right' },
            { key: 'marketPrice', label: 'Last', align: 'right' },
          ]}
          rows={mockPositions}
          getRowId={(row) => row.symbol}
        />
      );
    default:
      return (
        <Typography variant="body2" color="text.secondary">
          Data stream pending.
        </Typography>
      );
  }
}

interface DashboardGridProps {
  layout: DashboardLayout;
}

export function DashboardGrid({ layout }: DashboardGridProps) {
  return (
    <Box
      sx={{
        display: 'grid',
        gridTemplateColumns: 'repeat(12, minmax(0, 1fr))',
        gap: tokens.spacing.md,
      }}
    >
      {layout.widgets.map((widget) => {
        const definition = getWidgetById(widget.id);
        return (
          <Paper
            key={widget.id}
            variant="outlined"
            sx={{
              gridColumn: `span ${widget.w}`,
              gridRow: `span ${widget.h}`,
              padding: tokens.spacing.md,
              backgroundColor: tokens.colors.surface,
              borderColor: tokens.colors.border,
            }}
          >
            <Typography variant="subtitle2" sx={{ marginBottom: tokens.spacing.sm }}>
              {definition?.title ?? widget.id}
            </Typography>
            {renderWidget(widget)}
          </Paper>
        );
      })}
    </Box>
  );
}
