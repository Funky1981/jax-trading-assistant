import { OrderTicket } from '../components';
import { useDomain } from '../domain/store';
import { selectTickBySymbol } from '../domain/selectors';
import { HelpHint } from '@/components/ui/help-hint';

export function OrderTicketPage() {
  const { state, actions } = useDomain();
  const primarySymbol = 'AAPL';
  const tick = selectTickBySymbol(state, primarySymbol);

  return (
    <div className="space-y-4 w-full max-w-lg">
      <h1 className="flex items-center gap-2 text-3xl font-semibold">
        Order Ticket
        <HelpHint text="Place an order with the current market price prefilled." />
      </h1>
      <p className="text-sm text-muted-foreground">
        Place a paper order using the latest price.
      </p>
      <OrderTicket
        symbol={primarySymbol}
        defaultPrice={tick?.price}
        onSubmit={actions.placeOrder}
      />
    </div>
  );
}
