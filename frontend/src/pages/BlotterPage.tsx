import { TradeBlotterPanel } from '@/components/dashboard';
import { HelpHint } from '@/components/ui/help-hint';

export function BlotterPage() {
  return (
    <div className="space-y-4">
      <h1 className="flex items-center gap-2 text-3xl font-semibold">
        Blotter
        <HelpHint text="Execution history and order activity log." />
      </h1>
      <p className="text-sm text-muted-foreground">
        Review recent orders and their status.
      </p>
      <TradeBlotterPanel isOpen onToggle={() => undefined} />
    </div>
  );
}
