import { OrderTicket } from '../components';
import { useDomain } from '../domain/store';
import { selectTickBySymbol } from '../domain/selectors';

export function OrderTicketPage() {
  const { state, actions } = useDomain();
  const primarySymbol = 'AAPL';
  const tick = selectTickBySymbol(state, primarySymbol);

  return (
    <div className="space-y-4 max-w-lg">
      <h1 className="text-3xl font-semibold">Order Ticket</h1>
      <p className="text-sm text-muted-foreground">
        Place orders quickly with pre-filled defaults.
      </p>
      <OrderTicket
        symbol={primarySymbol}
        defaultPrice={tick?.price}
        onSubmit={actions.placeOrder}
      />
    </div>
  );
}
