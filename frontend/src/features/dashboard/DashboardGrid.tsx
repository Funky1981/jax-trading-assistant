import { TrendingUp, BarChart3, DollarSign, Gauge, List as ListIcon } from 'lucide-react';
import { Card, CardContent } from '@/components/ui/card';
import { cn } from '@/lib/utils';
import {
  DataTable,
  OrderTicket,
  PositionCard,
  RiskSummary,
  PnLIndicator,
  EmptyState,
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
      if (ticks.length === 0) {
        return (
          <EmptyState
            icon={<TrendingUp className="h-10 w-10" />}
            title="No market data"
            description="Watchlist will populate when market data arrives."
            compact
          />
        );
      }
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
          <EmptyState
            icon={<BarChart3 className="h-10 w-10" />}
            title="No positions"
            description="Positions will appear here after your first fills."
            compact
          />
        );
      }
      return (
        <div className="space-y-2">
          {positions.map((position) => (
            <PositionCard key={position.symbol} position={position} />
          ))}
        </div>
      );
    case 'risk-summary':
      if (positions.length === 0) {
        return (
          <EmptyState
            icon={<Gauge className="h-10 w-10" />}
            title="No risk data"
            description="Risk metrics will appear after your first fills."
            compact
          />
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
      if (orders.length === 0) {
        return (
          <EmptyState
            icon={<ListIcon className="h-10 w-10" />}
            title="No orders"
            description="Your order history will appear here."
            compact
          />
        );
      }
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
        <div className="space-y-2">
          <div className="flex items-center gap-2">
            <Gauge className="h-5 w-5 text-emerald-500" />
            <p className="text-sm font-medium">Market feed: healthy</p>
          </div>
          <div className="pl-6">
            <p className="text-sm text-muted-foreground">
              Latency: 12ms
            </p>
            <p className="text-sm text-muted-foreground">
              Last refresh: {new Date().toLocaleTimeString()}
            </p>
          </div>
        </div>
      );
    default:
      return (
        <EmptyState
          title="Data pending"
          description="Waiting for data stream to connect..."
          compact
        />
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
    <div
      className="grid grid-cols-12 gap-6"
      style={{ gridAutoRows: `${tokens.layout.gridRowHeight}px` }}
    >
      {layout.widgets.map((widget) => {
        const definition = getWidgetById(widget.id);
        return (
          <Card
            key={widget.id}
            className="transition-all duration-200 hover:shadow-lg hover:-translate-y-0.5"
            style={{
              gridColumn: `${widget.x + 1} / span ${widget.w}`,
              gridRow: `${widget.y + 1} / span ${widget.h}`,
            }}
          >
            <CardContent className="p-5 h-full flex flex-col">
              <div className="mb-3 pb-2 border-b">
                <h3 className="text-base font-semibold">
                  {definition?.title ?? widget.id}
                </h3>
              </div>
              <div className="flex-1 overflow-auto">
                {renderWidget(widget, {
                  positions,
                  orders,
                  ticks,
                  riskLimits,
                  onOrderSubmit,
                })}
              </div>
            </CardContent>
          </Card>
        );
      })}
    </div>
  );
}
