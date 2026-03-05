import { OrderTicketPanel } from '@/components/dashboard/OrderTicketPanel';
import { HelpHint } from '@/components/ui/help-hint';

export function OrderTicketPage() {
  return (
    <div className="space-y-4 w-full max-w-2xl">
      <h1 className="flex items-center gap-2 text-3xl font-semibold">
        Order Ticket
        <HelpHint text="Place a broker-backed paper order from the live order form." />
      </h1>
      <p className="text-sm text-muted-foreground">
        Submit orders through the IB bridge integration.
      </p>
      <OrderTicketPanel isOpen onToggle={() => undefined} />
    </div>
  );
}
