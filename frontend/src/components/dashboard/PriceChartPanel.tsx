import { useEffect, useMemo, useRef, useState } from 'react';
import { LineChart } from 'lucide-react';
import { createChart, IChartApi, ISeriesApi, CandlestickData, Time } from 'lightweight-charts';
import { useQuery } from '@tanstack/react-query';
import { CollapsiblePanel } from './CollapsiblePanel';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { formatCurrency } from '@/lib/utils';
import { buildUrl } from '@/config/api';

interface PriceChartPanelProps {
  isOpen: boolean;
  onToggle: () => void;
}

const symbols = ['AAPL', 'MSFT', 'GOOGL', 'NVDA', 'TSLA', 'SPY', 'QQQ'];
const timeframes = [
  { label: '1m', value: '1' },
  { label: '5m', value: '5' },
  { label: '15m', value: '15' },
  { label: '1h', value: '60' },
  { label: '1d', value: '1D' },
];

interface RawCandle {
  timestamp: string;
  open: number;
  high: number;
  low: number;
  close: number;
}

async function fetchCandles(symbol: string, timeframe: string): Promise<CandlestickData[]> {
  const response = await fetch(
    buildUrl('IB_BRIDGE', `/candles/${encodeURIComponent(symbol)}?limit=100&timeframe=${encodeURIComponent(timeframe)}`)
  );
  if (!response.ok) {
    throw new Error(`Chart data unavailable (HTTP ${response.status})`);
  }

  const payload = (await response.json()) as { candles?: RawCandle[] };
  const candles = payload.candles ?? [];

  const mapped = candles
    .map((candle) => {
      const ts = Date.parse(candle.timestamp);
      if (!Number.isFinite(ts)) return null;
      return {
        time: Math.floor(ts / 1000) as Time,
        open: candle.open,
        high: candle.high,
        low: candle.low,
        close: candle.close,
      } satisfies CandlestickData;
    })
    .filter((value): value is CandlestickData => value !== null);

  if (mapped.length === 0) {
    throw new Error('Chart data unavailable (no valid candles)');
  }

  return mapped;
}

export function PriceChartPanel({ isOpen, onToggle }: PriceChartPanelProps) {
  const [symbol, setSymbol] = useState('AAPL');
  const [timeframe, setTimeframe] = useState('15');
  const chartContainerRef = useRef<HTMLDivElement>(null);
  const chartRef = useRef<IChartApi | null>(null);
  const seriesRef = useRef<ISeriesApi<'Candlestick'> | null>(null);

  const { data = [], isLoading, isError } = useQuery({
    queryKey: ['chart-candles', symbol, timeframe],
    queryFn: () => fetchCandles(symbol, timeframe),
    refetchInterval: (query) => (query.state.error ? false : 10_000),
    retry: false,
  });

  const currentPrice = data[data.length - 1]?.close || 0;
  const prevClose = data[data.length - 2]?.close || currentPrice;
  const priceChange = currentPrice - prevClose;
  const priceChangePercent = prevClose > 0 ? (priceChange / prevClose) * 100 : 0;

  const statusSummary = useMemo(() => {
    if (isLoading) return <span className="text-xs text-muted-foreground">Loading live candles...</span>;
    if (isError) return <span className="text-xs text-destructive">Chart feed unavailable</span>;
    return (
      <div className="flex items-center gap-2">
        <span className="font-mono font-semibold">{symbol}</span>
        <span className="font-mono">{formatCurrency(currentPrice)}</span>
        <span
          className={`font-mono text-xs ${
            priceChange >= 0 ? 'text-success' : 'text-destructive'
          }`}
        >
          {priceChange >= 0 ? '+' : ''}{priceChangePercent.toFixed(2)}%
        </span>
      </div>
    );
  }, [currentPrice, isError, isLoading, priceChange, priceChangePercent, symbol]);

  useEffect(() => {
    if (!chartContainerRef.current || !isOpen || isError || data.length === 0) return;

    // Create chart with actual colors (lightweight-charts doesn't support CSS variables)
    const chart = createChart(chartContainerRef.current, {
      layout: {
        background: { color: 'transparent' },
        textColor: '#94a3b8', // slate-400
      },
      grid: {
        vertLines: { color: '#1e293b' }, // slate-800
        horzLines: { color: '#1e293b' },
      },
      crosshair: {
        mode: 1,
      },
      rightPriceScale: {
        borderColor: '#1e293b',
      },
      timeScale: {
        borderColor: '#1e293b',
        timeVisible: true,
        secondsVisible: false,
      },
      width: chartContainerRef.current.clientWidth,
      height: 300,
    });

    chartRef.current = chart;

    // Add candlestick series
    const candlestickSeries = chart.addCandlestickSeries({
      upColor: '#22c55e', // green-500
      downColor: '#ef4444', // red-500
      borderDownColor: '#ef4444',
      borderUpColor: '#22c55e',
      wickDownColor: '#ef4444',
      wickUpColor: '#22c55e',
    });

    seriesRef.current = candlestickSeries;
    candlestickSeries.setData(data);
    chart.timeScale().fitContent();

    // Handle resize
    const handleResize = () => {
      if (chartContainerRef.current) {
        chart.applyOptions({ width: chartContainerRef.current.clientWidth });
      }
    };

    window.addEventListener('resize', handleResize);

    return () => {
      window.removeEventListener('resize', handleResize);
      chart.remove();
      chartRef.current = null;
      seriesRef.current = null;
    };
  }, [data, isError, isOpen]);

  // Update data when symbol/timeframe changes
  useEffect(() => {
    if (seriesRef.current && isOpen && data.length > 0) {
      seriesRef.current.setData(data);
      chartRef.current?.timeScale().fitContent();
    }
  }, [data, isOpen]);

  return (
    <CollapsiblePanel
      title="Price Chart"
      icon={<LineChart className="h-4 w-4" />}
      summary={statusSummary}
      isOpen={isOpen}
      onToggle={onToggle}
      isLoading={isLoading}
    >
      <div className="space-y-4">
        {/* Controls */}
        <div className="flex gap-4">
          <div className="space-y-1">
            <label className="text-xs text-muted-foreground">Symbol</label>
            <Select value={symbol} onValueChange={setSymbol}>
              <SelectTrigger className="w-28 h-8">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {symbols.map((s) => (
                  <SelectItem key={s} value={s}>
                    {s}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <div className="space-y-1">
            <label className="text-xs text-muted-foreground">Timeframe</label>
            <Select value={timeframe} onValueChange={setTimeframe}>
              <SelectTrigger className="w-20 h-8">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {timeframes.map((tf) => (
                  <SelectItem key={tf.value} value={tf.value}>
                    {tf.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <div className="ml-auto text-right">
            <p className="text-2xl font-mono font-bold">
              {isError ? '—' : formatCurrency(currentPrice)}
            </p>
            <p
              className={`text-sm font-mono ${
                priceChange >= 0 ? 'text-success' : 'text-destructive'
              }`}
            >
              {isError ? 'Unavailable' : `${priceChange >= 0 ? '+' : ''}${formatCurrency(priceChange)} (${priceChangePercent.toFixed(2)}%)`}
            </p>
          </div>
        </div>

        {/* Chart */}
        {isError ? (
          <div className="h-[300px] flex items-center justify-center rounded-md border border-dashed border-border text-sm text-muted-foreground">
            Live chart data is unavailable.
          </div>
        ) : (
          <div ref={chartContainerRef} className="w-full" />
        )}
      </div>
    </CollapsiblePanel>
  );
}
