export interface MarketSymbol {
  code: string;
  name: string;
}

export interface PricePoint {
  value: number;
  timestamp: number;
}

export function formatPrice(value: number) {
  return new Intl.NumberFormat('en-US', {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  }).format(value);
}
