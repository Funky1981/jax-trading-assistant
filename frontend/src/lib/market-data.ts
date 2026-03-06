export function normalizeMarketDataMode(mode?: string | null): string {
  const value = (mode ?? '').trim().toLowerCase();
  if (value === 'live' || value === 'delayed' || value === 'frozen' || value === 'delayed-frozen') {
    return value;
  }
  return 'unknown';
}

export function getMarketDataLabel(mode?: string | null): string {
  switch (normalizeMarketDataMode(mode)) {
    case 'live':
      return 'Live';
    case 'delayed':
      return 'Delayed';
    case 'frozen':
      return 'Frozen';
    case 'delayed-frozen':
      return 'Delayed Frozen';
    default:
      return 'Unknown';
  }
}

export function getMarketDataTone(mode?: string | null): string {
  switch (normalizeMarketDataMode(mode)) {
    case 'live':
      return 'bg-emerald-500/10 text-emerald-500 border border-emerald-500/20';
    case 'delayed':
    case 'delayed-frozen':
      return 'bg-yellow-500/10 text-yellow-500 border border-yellow-500/20';
    case 'frozen':
      return 'bg-orange-500/10 text-orange-500 border border-orange-500/20';
    default:
      return 'bg-muted/40 text-muted-foreground border border-border';
  }
}

export function getMarketDataBadgeText(mode?: string | null, paperTrading?: boolean): string {
  const label = getMarketDataLabel(mode).toUpperCase();
  return paperTrading ? `${label} PAPER` : label;
}
