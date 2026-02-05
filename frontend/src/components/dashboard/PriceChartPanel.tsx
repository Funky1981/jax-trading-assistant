import { useEffect, useRef, useState } from 'react';
import { LineChart } from 'lucide-react';
import { createChart, IChartApi, ISeriesApi, CandlestickData, Time } from 'lightweight-charts';
import { CollapsiblePanel } from './CollapsiblePanel';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { formatCurrency } from '@/lib/utils';

interface PriceChartPanelProps {
  isOpen: boolean;
  onToggle: () => void;
}

const symbols = ['AAPL', 'MSFT', 'GOOGL', 'NVDA', 'TSLA', 'SPY', 'QQQ'];
const timeframes = ['1m', '5m', '15m', '1h', '1d'];

// Generate mock candlestick data
function generateMockData(symbol: string, timeframe: string): CandlestickData[] {
  const data: CandlestickData[] = [];
  const basePrice = {
    AAPL: 185,
    MSFT: 412,
    GOOGL: 138,
    NVDA: 721,
    TSLA: 231,
    SPY: 512,
    QQQ: 438,
  }[symbol] || 100;

  const now = Date.now();
  const intervalMs = {
    '1m': 60000,
    '5m': 300000,
    '15m': 900000,
    '1h': 3600000,
    '1d': 86400000,
  }[timeframe] || 60000;

  let price = basePrice;
  
  for (let i = 100; i >= 0; i--) {
    const time = Math.floor((now - i * intervalMs) / 1000) as Time;
    const open = price;
    const volatility = basePrice * 0.02;
    const change = (Math.random() - 0.5) * volatility;
    const high = open + Math.random() * volatility * 0.5;
    const low = open - Math.random() * volatility * 0.5;
    const close = open + change;
    
    data.push({
      time,
      open: Number(open.toFixed(2)),
      high: Number(Math.max(open, close, high).toFixed(2)),
      low: Number(Math.min(open, close, low).toFixed(2)),
      close: Number(close.toFixed(2)),
    });
    
    price = close;
  }
  
  return data;
}

export function PriceChartPanel({ isOpen, onToggle }: PriceChartPanelProps) {
  const [symbol, setSymbol] = useState('AAPL');
  const [timeframe, setTimeframe] = useState('15m');
  const chartContainerRef = useRef<HTMLDivElement>(null);
  const chartRef = useRef<IChartApi | null>(null);
  const seriesRef = useRef<ISeriesApi<'Candlestick'> | null>(null);

  const data = generateMockData(symbol, timeframe);
  const currentPrice = data[data.length - 1]?.close || 0;
  const prevClose = data[data.length - 2]?.close || currentPrice;
  const priceChange = currentPrice - prevClose;
  const priceChangePercent = (priceChange / prevClose) * 100;

  useEffect(() => {
    if (!chartContainerRef.current || !isOpen) return;

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
  }, [data, isOpen, symbol, timeframe]);

  // Update data when symbol/timeframe changes
  useEffect(() => {
    if (seriesRef.current && isOpen) {
      seriesRef.current.setData(data);
      chartRef.current?.timeScale().fitContent();
    }
  }, [data, symbol, timeframe, isOpen]);

  const summary = (
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

  return (
    <CollapsiblePanel
      title="Price Chart"
      icon={<LineChart className="h-4 w-4" />}
      summary={summary}
      isOpen={isOpen}
      onToggle={onToggle}
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
                  <SelectItem key={tf} value={tf}>
                    {tf}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <div className="ml-auto text-right">
            <p className="text-2xl font-mono font-bold">{formatCurrency(currentPrice)}</p>
            <p
              className={`text-sm font-mono ${
                priceChange >= 0 ? 'text-success' : 'text-destructive'
              }`}
            >
              {priceChange >= 0 ? '+' : ''}{formatCurrency(priceChange)} ({priceChangePercent.toFixed(2)}%)
            </p>
          </div>
        </div>

        {/* Chart */}
        <div ref={chartContainerRef} className="w-full" />
      </div>
    </CollapsiblePanel>
  );
}
