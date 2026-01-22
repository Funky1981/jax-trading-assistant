export interface MarketTick {
  symbol: string;
  price: number;
  changePct: number;
  timestamp: number;
}

export interface QuoteSnapshot {
  symbol: string;
  bid: number;
  ask: number;
  last: number;
  timestamp: number;
}
