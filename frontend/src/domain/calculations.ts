import type { Position, RiskLimits } from './models';

export function calculatePositionValue(position: Position) {
  return position.marketPrice * position.quantity;
}

export function calculateUnrealizedPnl(position: Position) {
  return (position.marketPrice - position.avgPrice) * position.quantity;
}

export function calculateTotalExposure(positions: Position[]) {
  return positions.reduce((total, position) => total + Math.abs(calculatePositionValue(position)), 0);
}

export function calculateTotalUnrealizedPnl(positions: Position[]) {
  return positions.reduce((total, position) => total + calculateUnrealizedPnl(position), 0);
}

export function isRiskBreached(limits: RiskLimits, exposure: number, pnl: number) {
  if (exposure > limits.maxPositionValue) return true;
  if (pnl < -Math.abs(limits.maxDailyLoss)) return true;
  return false;
}
