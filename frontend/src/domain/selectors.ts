import type { DomainState } from './state';
import { calculateTotalExposure, calculateTotalUnrealizedPnl, isRiskBreached } from './calculations';

export function selectOrders(state: DomainState) {
  return Object.values(state.orders);
}

export function selectOpenOrders(state: DomainState) {
  return selectOrders(state).filter((order) => order.status === 'open');
}

export function selectPositions(state: DomainState) {
  return Object.values(state.positions);
}

export function selectTicks(state: DomainState) {
  return Object.values(state.ticks).sort((a, b) => a.symbol.localeCompare(b.symbol));
}

export function selectTickBySymbol(state: DomainState, symbol: string) {
  return state.ticks[symbol] ?? null;
}

export function selectTotalExposure(state: DomainState) {
  return calculateTotalExposure(selectPositions(state));
}

export function selectTotalUnrealizedPnl(state: DomainState) {
  return calculateTotalUnrealizedPnl(selectPositions(state));
}

export function selectRiskBreach(state: DomainState) {
  const exposure = selectTotalExposure(state);
  const pnl = selectTotalUnrealizedPnl(state);
  return isRiskBreached(state.riskLimits, exposure, pnl);
}
