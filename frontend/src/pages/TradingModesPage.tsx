import { useEffect, useState } from 'react';
import { tradingModesService } from '../data/trading-modes-service';
import type { TradingMode } from '../data/types';

function BadgePill({ label, variant = 'default' }: { label: string; variant?: 'default' | 'green' | 'blue' | 'yellow' }) {
  const colours: Record<string, string> = {
    default: 'bg-zinc-700 text-zinc-200',
    green: 'bg-emerald-800 text-emerald-200',
    blue: 'bg-blue-800 text-blue-200',
    yellow: 'bg-yellow-800 text-yellow-200',
  };
  return (
    <span className={`inline-block rounded px-1.5 py-0.5 text-xs font-medium ${colours[variant]}`}>{label}</span>
  );
}

function ModeCard({ mode }: { mode: TradingMode }) {
  return (
    <div className="rounded-xl border border-zinc-700 bg-zinc-900 p-5 space-y-4">
      {/* Header */}
      <div className="flex items-start justify-between gap-2">
        <div>
          <h3 className="text-base font-semibold text-white">{mode.name}</h3>
          <p className="text-sm text-zinc-400 mt-0.5">{mode.displayCopy ?? mode.description}</p>
        </div>
        <div className="flex flex-col items-end gap-1 shrink-0">
          {mode.horizonLabel && <BadgePill label={mode.horizonLabel} variant="green" />}
          <BadgePill label={mode.runtimeMode} variant="blue" />
          <BadgePill label={mode.executionPolicy} variant="yellow" />
        </div>
      </div>

      {/* Universe */}
      {mode.universe && mode.universe.length > 0 && (
        <div>
          <p className="text-xs text-zinc-500 uppercase tracking-wide mb-1.5">Universe</p>
          <div className="flex flex-wrap gap-1.5">
            {mode.universe.map(sym => (
              <BadgePill key={sym} label={sym} variant="green" />
            ))}
          </div>
        </div>
      )}

      {/* Strategies */}
      {mode.strategies && mode.strategies.length > 0 && (
        <div>
          <p className="text-xs text-zinc-500 uppercase tracking-wide mb-1.5">Strategies</p>
          <ul className="space-y-1">
            {mode.strategies.map(s => (
              <li key={s.strategyTypeId} className="rounded bg-zinc-800 px-3 py-2">
                <p className="text-sm font-medium text-zinc-100">{s.name}</p>
                <p className="text-xs text-zinc-400 mt-0.5">{s.description}</p>
              </li>
            ))}
          </ul>
        </div>
      )}

      {/* Risk Defaults */}
      {mode.riskDefaults && (
        <div>
          <p className="text-xs text-zinc-500 uppercase tracking-wide mb-1.5">Risk Defaults</p>
          <div className="grid grid-cols-2 gap-x-4 gap-y-1 text-sm">
            <span className="text-zinc-400">Max trades/day</span>
            <span className="text-zinc-100">{mode.riskDefaults.maxTradesPerDay}</span>
            <span className="text-zinc-400">Max open positions</span>
            <span className="text-zinc-100">{mode.riskDefaults.maxOpenPositions}</span>
            <span className="text-zinc-400">Risk per trade</span>
            <span className="text-zinc-100">{(mode.riskDefaults.riskPerTradePct * 100).toFixed(1)}%</span>
            <span className="text-zinc-400">Min confidence</span>
            <span className="text-zinc-100">{(mode.riskDefaults.minConfidence * 100).toFixed(0)}%</span>
            <span className="text-zinc-400">Flatten by</span>
            <span className="text-zinc-100">{mode.riskDefaults.flattenBy}</span>
            <span className="text-zinc-400">Approval required</span>
            <span className="text-zinc-100">{mode.riskDefaults.approvalRequired ? 'Yes' : 'No'}</span>
          </div>
        </div>
      )}
    </div>
  );
}

export function TradingModesPage() {
  const [modes, setModes] = useState<TradingMode[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    tradingModesService.list()
      .then(data => { if (!cancelled) { setModes(data); setLoading(false); } })
      .catch(err => { if (!cancelled) { setError(String(err)); setLoading(false); } });
    return () => { cancelled = true; };
  }, []);

  if (loading) {
    return (
      <div className="flex items-center justify-center h-40 text-zinc-400">
        Loading trading modes…
      </div>
    );
  }

  if (error) {
    return (
      <div className="rounded-lg border border-red-700 bg-red-950 p-4 text-red-300 text-sm">
        Failed to load trading modes: {error}
      </div>
    );
  }

  return (
    <div className="max-w-3xl mx-auto space-y-6 py-6 px-4">
      <div>
        <h1 className="text-xl font-bold text-white">Trading Modes</h1>
        <p className="text-sm text-zinc-400 mt-1">
          Pre-configured trading modes defining strategy sets, risk defaults, horizons, and universes.
        </p>
      </div>

      {modes.length === 0 ? (
        <p className="text-zinc-400 text-sm">No trading modes available.</p>
      ) : (
        <div className="space-y-4">
          {modes.map(mode => (
            <ModeCard key={mode.id} mode={mode} />
          ))}
        </div>
      )}
    </div>
  );
}
