import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useReducer,
  useRef,
  type Dispatch,
  type PropsWithChildren,
} from 'react';
import { createStreamBuffer } from '../data/stream-buffer';
import type { MarketTick } from '../data/types';
import type { DomainEvent } from './events';
import type { DomainState } from './state';
import { defaultState, reduceDomainState } from './state';
import type { Order, OrderDraft, Position } from './models';

const watchlistSeed = [
  { symbol: 'AAPL', price: 249.42 },
  { symbol: 'MSFT', price: 413.1 },
  { symbol: 'SPY', price: 541.22 },
  { symbol: 'TSLA', price: 248.5 },
];

const seedPositions: Position[] = [
  { symbol: 'AAPL', quantity: 250, avgPrice: 231.12, marketPrice: 249.42 },
  { symbol: 'MSFT', quantity: 120, avgPrice: 402.55, marketPrice: 413.1 },
];

const seedOrders: Order[] = [
  {
    id: 'ord-1',
    symbol: 'AAPL',
    side: 'buy',
    quantity: 150,
    price: 249.42,
    status: 'filled',
    createdAt: Date.now() - 1000 * 60 * 15,
  },
  {
    id: 'ord-2',
    symbol: 'MSFT',
    side: 'sell',
    quantity: 50,
    price: 413.1,
    status: 'open',
    createdAt: Date.now() - 1000 * 60 * 4,
  },
  {
    id: 'ord-3',
    symbol: 'SPY',
    side: 'buy',
    quantity: 200,
    price: 541.22,
    status: 'cancelled',
    createdAt: Date.now() - 1000 * 60 * 30,
  },
];

const DomainContext = createContext<DomainStore | null>(null);

interface DomainStore {
  state: DomainState;
  dispatch: Dispatch<DomainEvent>;
  actions: {
    placeOrder: (draft: OrderDraft) => void;
  };
}

function mapBySymbol<T extends { symbol: string }>(items: T[]) {
  return items.reduce<Record<string, T>>((acc, item) => {
    acc[item.symbol] = item;
    return acc;
  }, {});
}

function mapById<T extends { id: string }>(items: T[]) {
  return items.reduce<Record<string, T>>((acc, item) => {
    acc[item.id] = item;
    return acc;
  }, {});
}

function buildSeedTicks() {
  const timestamp = Date.now();
  const ticks: MarketTick[] = watchlistSeed.map((seed) => ({
    symbol: seed.symbol,
    price: seed.price,
    changePct: 0,
    timestamp,
  }));

  for (const position of seedPositions) {
    const existing = ticks.find((tick) => tick.symbol === position.symbol);
    if (existing) {
      existing.price = position.marketPrice;
    } else {
      ticks.push({
        symbol: position.symbol,
        price: position.marketPrice,
        changePct: 0,
        timestamp,
      });
    }
  }

  return ticks;
}

function createSeedState(): DomainState {
  return {
    ...defaultState,
    positions: mapBySymbol(seedPositions),
    orders: mapById(seedOrders),
    ticks: mapBySymbol(buildSeedTicks()),
  };
}

function createId(prefix: string) {
  if (typeof crypto !== 'undefined' && 'randomUUID' in crypto) {
    return `${prefix}-${crypto.randomUUID()}`;
  }
  return `${prefix}-${Math.random().toString(36).slice(2, 10)}`;
}

function applyFill(existing: Position | undefined, order: Order): Position {
  const signedQty = order.side === 'buy' ? order.quantity : -order.quantity;
  const currentQty = existing?.quantity ?? 0;
  const nextQty = currentQty + signedQty;
  const marketPrice = existing?.marketPrice ?? order.price;

  if (!existing || currentQty === 0 || Math.sign(currentQty) !== Math.sign(nextQty)) {
    return {
      symbol: order.symbol,
      quantity: nextQty,
      avgPrice: order.price,
      marketPrice,
    };
  }

  if (nextQty === 0) {
    return {
      symbol: order.symbol,
      quantity: 0,
      avgPrice: order.price,
      marketPrice,
    };
  }

  const weighted =
    (currentQty * existing.avgPrice + signedQty * order.price) / nextQty;

  return {
    symbol: order.symbol,
    quantity: nextQty,
    avgPrice: weighted,
    marketPrice,
  };
}

export function DomainProvider({ children }: PropsWithChildren) {
  const initialState = useMemo(() => createSeedState(), []);
  const [state, dispatch] = useReducer(reduceDomainState, initialState);
  const stateRef = useRef(state);
  const timeoutsRef = useRef<ReturnType<typeof setTimeout>[]>([]);
  const symbolsRef = useRef(Object.keys(initialState.ticks));
  const priceRef = useRef<Record<string, number>>(
    Object.fromEntries(
      Object.values(initialState.ticks).map((tick) => [tick.symbol, tick.price])
    )
  );

  useEffect(() => {
    stateRef.current = state;
  }, [state]);

  useEffect(() => {
    return () => {
      timeoutsRef.current.forEach((timeout) => clearTimeout(timeout));
    };
  }, []);

  useEffect(() => {
    const buffer = createStreamBuffer<MarketTick>({
      flushIntervalMs: 500,
      getKey: (tick) => tick.symbol,
      onFlush: (ticks) => {
        ticks.forEach((tick) => {
          dispatch({
            type: 'PriceUpdated',
            symbol: tick.symbol,
            price: tick.price,
            changePct: tick.changePct,
            timestamp: tick.timestamp,
          });
        });
      },
    });

    const interval = setInterval(() => {
      const now = Date.now();
      symbolsRef.current.forEach((symbol) => {
        const current = priceRef.current[symbol] ?? 100;
        const drift = (Math.random() - 0.5) * 0.8;
        const next = Math.max(0.01, current + drift);
        const changePct = ((next - current) / current) * 100;
        priceRef.current[symbol] = next;
        buffer.push({ symbol, price: next, changePct, timestamp: now });
      });
    }, 250);

    return () => {
      clearInterval(interval);
      buffer.stop();
    };
  }, []);

  const placeOrder = useCallback((draft: OrderDraft) => {
    if (!draft.symbol || draft.quantity <= 0) return;
    const fallbackPrice = stateRef.current.ticks[draft.symbol]?.price ?? 0;
    const resolvedPrice = draft.price || fallbackPrice;
    if (!resolvedPrice) return;

    const order: Order = {
      id: createId('ord'),
      symbol: draft.symbol,
      side: draft.side,
      quantity: draft.quantity,
      price: resolvedPrice,
      status: 'open',
      createdAt: Date.now(),
    };

    dispatch({ type: 'OrderPlaced', order });

    const fillTimeout = setTimeout(() => {
      dispatch({ type: 'OrderUpdated', orderId: order.id, status: 'filled' });
      const nextPosition = applyFill(
        stateRef.current.positions[order.symbol],
        order
      );
      dispatch({ type: 'PositionUpdated', position: nextPosition });
    }, 800);

    timeoutsRef.current.push(fillTimeout);

    if (!(draft.symbol in priceRef.current)) {
      symbolsRef.current = [...symbolsRef.current, draft.symbol];
      priceRef.current[draft.symbol] = resolvedPrice;
    }
  }, []);

  const store = useMemo<DomainStore>(
    () => ({
      state,
      dispatch,
      actions: {
        placeOrder,
      },
    }),
    [state, placeOrder]
  );

  return (
    <DomainContext.Provider value={store}>{children}</DomainContext.Provider>
  );
}

export function useDomain() {
  const store = useContext(DomainContext);
  if (!store) {
    throw new Error('useDomain must be used within DomainProvider');
  }
  return store;
}
