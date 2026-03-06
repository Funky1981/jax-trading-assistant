import { OrderTicketPanel } from '@/components/dashboard/OrderTicketPanel';
import { HelpHint } from '@/components/ui/help-hint';

export function OrderTicketPage() {
  return (
    <div className="space-y-4 w-full max-w-2xl">
      <h1 className="flex items-center gap-2 text-3xl font-semibold">
        Order Ticket
        <HelpHint text="Place a broker-backed entry order with optional stop-loss and take-profit protection." />
      </h1>
      <p className="text-sm text-muted-foreground">
        Submit broker orders through the IB bridge, then manage pending orders in the blotter and open exposure in the positions panel.
      </p>
      <OrderTicketPanel isOpen onToggle={() => undefined} />
    </div>
  );
}
