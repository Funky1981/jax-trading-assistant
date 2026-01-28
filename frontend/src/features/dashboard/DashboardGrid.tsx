import { Box, Paper, Stack, Typography } from '@mui/material';
import {
  DataTable,
  OrderTicket,
  PositionCard,
  RiskSummary,
  PnLIndicator,
} from '../../components';
import { calculateTotalExposure, calculateTotalUnrealizedPnl } from '../../domain/calculations';
import type { Order, OrderDraft, Position, RiskLimits } from '../../domain/models';
import { formatPrice } from '../../domain/market';
import { tokens } from '../../styles/tokens';
import type { DashboardLayout, WidgetLayout } from './layouts';
import { getWidgetById } from './registry';
import type { MarketTick } from '../../data/types';

function renderWidget(
  widget: WidgetLayout,
  {
    positions,
    orders,
    ticks,
    riskLimits,
    onOrderSubmit,
  }: {
    positions: Position[];
    orders: Order[];
    ticks: MarketTick[];
    riskLimits: RiskLimits;
    onOrderSubmit?: (draft: OrderDraft) => void;
  }
) {
  switch (widget.id) {
    case 'order-ticket': {
      const primary = ticks.find((tick) => tick.symbol === 'AAPL') ?? ticks[0];
      return (
        <OrderTicket
          symbol={primary?.symbol ?? 'AAPL'}
          defaultPrice={primary?.price}
          onSubmit={onOrderSubmit}
        />
      );
    }
    case 'watchlist':
      return (
        <DataTable
          columns={[
            { key: 'symbol', label: 'Symbol' },
            {
              key: 'price',
              label: 'Last',
              align: 'right',
              render: (row) => formatPrice(row.price),
            },
            {
              key: 'changePct',
              label: 'Change',
              align: 'right',
              render: (row) => <PnLIndicator value={row.changePct} suffix="%" />,
            },
          ]}
          rows={ticks}
          getRowId={(row) => row.symbol}
        />
      );
    case 'positions':
      if (positions.length === 0) {
        return (
          <Typography variant="body2" color="text.secondary">
            No positions yet.
          </Typography>
        );
      }
      return (
        <Stack spacing={2}>
          {positions.map((position) => (
            <PositionCard key={position.symbol} position={position} />
          ))}
        </Stack>
      );
    case 'risk-summary':
      if (positions.length === 0) {
        return (
          <Typography variant="body2" color="text.secondary">
            Risk metrics will appear after your first fills.
          </Typography>
        );
      }
      return (
        <RiskSummary
          exposure={calculateTotalExposure(positions)}
          pnl={calculateTotalUnrealizedPnl(positions)}
          limits={riskLimits}
        />
      );
    case 'blotter':
      return (
        <DataTable
          columns={[
            { key: 'symbol', label: 'Symbol' },
            { key: 'side', label: 'Side' },
            { key: 'quantity', label: 'Qty', align: 'right' },
            {
              key: 'price',
              label: 'Price',
              align: 'right',
              render: (row) => formatPrice(row.price),
            },
            { key: 'status', label: 'Status' },
          ]}
          rows={orders}
          getRowId={(row) => row.id}
        />
      );
    case 'system-status':
      return (
        <Stack spacing={1}>
          <Typography variant="body2">Market feed: healthy</Typography>
          <Typography variant="body2" color="text.secondary">
            Latency: 12ms
          </Typography>
          <Typography variant="body2" color="text.secondary">
            Last refresh: {new Date().toLocaleTimeString()}
          </Typography>
        </Stack>
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
  positions: Position[];
  orders: Order[];
  ticks: MarketTick[];
  riskLimits: RiskLimits;
  onOrderSubmit?: (draft: OrderDraft) => void;
}

export function DashboardGrid({
  layout,
  positions,
  orders,
  ticks,
  riskLimits,
  onOrderSubmit,
}: DashboardGridProps) {
  return (
    <Box
      sx={{
        display: 'grid',
        gridTemplateColumns: 'repeat(12, minmax(0, 1fr))',
        gridAutoRows: `${tokens.layout.gridRowHeight}px`,
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
              gridColumn: `${widget.x + 1} / span ${widget.w}`,
              gridRow: `${widget.y + 1} / span ${widget.h}`,
              padding: tokens.spacing.md,
              backgroundColor: tokens.colors.surface,
              borderColor: tokens.colors.border,
            }}
          >
            <Typography variant="subtitle2" sx={{ marginBottom: tokens.spacing.sm }}>
              {definition?.title ?? widget.id}
            </Typography>
            {renderWidget(widget, {
              positions,
              orders,
              ticks,
              riskLimits,
              onOrderSubmit,
            })}
          </Paper>
        );
      })}
    </Box>
  );
}
