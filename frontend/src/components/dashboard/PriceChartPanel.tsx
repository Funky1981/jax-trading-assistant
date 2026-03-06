import { useEffect, useMemo, useRef, useState } from 'react';
import { LineChart } from 'lucide-react';
import { createChart, IChartApi, ISeriesApi, CandlestickData, Time } from 'lightweight-charts';
import { useQuery } from '@tanstack/react-query';
import { CollapsiblePanel } from './CollapsiblePanel';
import { useWatchlist } from '@/hooks/useWatchlist';
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
  { label: '1m', value: '1m' },
  { label: '5m', value: '5m' },
  { label: '15m', value: '15m' },
  { label: '1h', value: '1h' },
  { label: '1d', value: '1d' },
];
const emptyCandles: CandlestickData[] = [];

interface RawCandle {
  timestamp: string;
  open: number;
  high: number;
  low: number;
  close: number;
}

interface ChartCandlesResponse {
  symbol?: string;
  timeframe?: string;
  requestedTimeframe?: string;
  degraded?: boolean;
  message?: string;
  marketDataMode?: string;
  paperTrading?: boolean;
  candles?: RawCandle[];
}

interface ChartCandlesResult {
  candles: CandlestickData[];
  timeframe: string;
  requestedTimeframe: string;
  degraded: boolean;
  message: string;
  marketDataMode: string;
  paperTrading: boolean;
}

async function fetchCandles(symbol: string, timeframe: string): Promise<ChartCandlesResult> {
  const response = await fetch(
    buildUrl(
      'JAX_API',
      `/api/v1/market/candles?symbol=${encodeURIComponent(symbol)}&limit=100&timeframe=${encodeURIComponent(timeframe)}`
    )
  );
  if (!response.ok) {
    throw new Error(`Chart data unavailable (HTTP ${response.status})`);
  }
  const contentType = response.headers.get('content-type') ?? '';
  if (!contentType.includes('application/json')) {
    throw new Error('Chart data unavailable (non-JSON response)');
  }

  const payload = (await response.json()) as ChartCandlesResponse;
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

  return {
    candles: mapped,
    timeframe: payload.timeframe ?? timeframe,
    requestedTimeframe: payload.requestedTimeframe ?? timeframe,
    degraded: payload.degraded === true,
    message: payload.message ?? '',
    marketDataMode: payload.marketDataMode ?? 'unknown',
    paperTrading: payload.paperTrading !== false,
  };
}

export function PriceChartPanel({ isOpen, onToggle }: PriceChartPanelProps) {
  const [symbol, setSymbol] = useState('AAPL');
  const [timeframe, setTimeframe] = useState('15m');
  const chartContainerRef = useRef<HTMLDivElement>(null);
  const chartRef = useRef<IChartApi | null>(null);
  const seriesRef = useRef<ISeriesApi<'Candlestick'> | null>(null);

  const { data, isLoading, isError } = useQuery({
    queryKey: ['chart-candles', symbol, timeframe],
    queryFn: () => fetchCandles(symbol, timeframe),
    refetchInterval: (query) => (query.state.error ? false : 10_000),
    retry: false,
  });
  const { data: watchlist = [] } = useWatchlist();
  const fallbackQuote = watchlist.find((item) => item.symbol === symbol);
  const candles = data?.candles ?? emptyCandles;
  const chartTimeframe = data?.timeframe ?? timeframe;
  const requestedTimeframe = data?.requestedTimeframe ?? timeframe;
  const isDegraded = data?.degraded === true;
  const degradedMessage = data?.message ?? '';
  const marketDataMode = data?.marketDataMode ?? 'unknown';
  const paperTrading = data?.paperTrading !== false;
  const marketDataLabel = marketDataMode === 'live' ? 'Live' : marketDataMode === 'delayed' ? 'Delayed' : marketDataMode === 'frozen' ? 'Frozen' : marketDataMode === 'delayed-frozen' ? 'Delayed Frozen' : 'Unknown';
  const marketDataTone = marketDataMode === 'live' ? 'text-success border-success/40 bg-success/10' : marketDataMode === 'delayed' || marketDataMode === 'delayed-frozen' ? 'text-warning border-warning/40 bg-warning/10' : 'text-muted-foreground border-border bg-muted/20';

  const currentPrice = candles[candles.length - 1]?.close || fallbackQuote?.price || 0;
  const prevClose = candles[candles.length - 2]?.close || currentPrice;
  const priceChange = currentPrice - prevClose;
  const priceChangePercent = prevClose > 0 ? (priceChange / prevClose) * 100 : 0;

  const statusSummary = useMemo(() => {
    if (isLoading) return <span className="text-xs text-muted-foreground">Loading live candles...</span>;
    if (isError && fallbackQuote) return <span className="text-xs text-warning">Live candles unavailable - delayed quote</span>;
    if (isError) return <span className="text-xs text-destructive">Chart feed unavailable</span>;
    if (isDegraded) {
      return (
        <span className="text-xs text-warning">
          {requestedTimeframe} unavailable - showing {chartTimeframe}
        </span>
      );
    }
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
  }, [
    chartTimeframe,
    currentPrice,
    fallbackQuote,
    isDegraded,
    isError,
    isLoading,
    priceChange,
    priceChangePercent,
    requestedTimeframe,
    symbol,
  ]);

  useEffect(() => {
    if (!chartContainerRef.current || !isOpen || isError || candles.length === 0) return;

    const chart = createChart(chartContainerRef.current, {
      layout: {
        background: { color: 'transparent' },
        textColor: '#94a3b8',
      },
      grid: {
        vertLines: { color: '#1e293b' },
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

    const candlestickSeries = chart.addCandlestickSeries({
      upColor: '#22c55e',
      downColor: '#ef4444',
      borderDownColor: '#ef4444',
      borderUpColor: '#22c55e',
      wickDownColor: '#ef4444',
      wickUpColor: '#22c55e',
    });

    seriesRef.current = candlestickSeries;
    candlestickSeries.setData(candles);
    chart.timeScale().fitContent();

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
  }, [candles, isError, isOpen]);

  useEffect(() => {
    if (seriesRef.current && isOpen && candles.length > 0) {
      seriesRef.current.setData(candles);
      chartRef.current?.timeScale().fitContent();
    }
  }, [candles, isOpen]);

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
        <div className="flex gap-4">
          <div className="space-y-1">
            <label htmlFor="price-chart-symbol" className="text-xs text-muted-foreground">Symbol</label>
            <Select value={symbol} onValueChange={setSymbol}>
              <SelectTrigger id="price-chart-symbol" className="w-28 h-8" aria-label="Chart symbol">
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
            <label htmlFor="price-chart-timeframe" className="text-xs text-muted-foreground">Timeframe</label>
            <Select value={timeframe} onValueChange={setTimeframe}>
              <SelectTrigger id="price-chart-timeframe" className="w-20 h-8" aria-label="Chart timeframe">
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
            <div className="mb-1 flex justify-end gap-2">
              <span className={`rounded-full border px-2 py-0.5 text-[10px] font-semibold uppercase tracking-wide ${marketDataTone}`}>
                {marketDataLabel}
              </span>
              <span className="rounded-full border border-border bg-muted/20 px-2 py-0.5 text-[10px] font-semibold uppercase tracking-wide text-muted-foreground">
                {paperTrading ? 'Paper' : 'Live Trading'}
              </span>
            </div>
            <p className="text-2xl font-mono font-bold">
              {currentPrice > 0 ? formatCurrency(currentPrice) : '--'}
            </p>
            <p
              className={`text-sm font-mono ${
                priceChange >= 0 ? 'text-success' : 'text-destructive'
              }`}
            >
              {isError && !fallbackQuote
                ? 'Unavailable'
                : `${priceChange >= 0 ? '+' : ''}${formatCurrency(priceChange)} (${priceChangePercent.toFixed(2)}%)`}
            </p>
          </div>
        </div>

        {isDegraded ? (
          <div className="rounded-md border border-warning/40 bg-warning/10 px-3 py-2 text-xs text-warning">
            {degradedMessage || `${requestedTimeframe} candles are unavailable. Showing ${chartTimeframe} instead.`}
          </div>
        ) : null}

        {isError ? (
          <div className="h-[300px] flex flex-col items-center justify-center gap-2 rounded-md border border-dashed border-border text-sm text-muted-foreground">
            {fallbackQuote ? (
              <>
                <div className="text-xs uppercase tracking-wide text-warning">Delayed Quote</div>
                <div className="font-mono text-lg text-foreground">{formatCurrency(fallbackQuote.price)}</div>
                <div>Live candles are currently unavailable.</div>
              </>
            ) : (
              <div>Live chart data is unavailable.</div>
            )}
          </div>
        ) : (
          <div ref={chartContainerRef} className="w-full" />
        )}
      </div>
    </CollapsiblePanel>
  );
}
