import { Check, Settings } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { useBeginnerMode } from '@/context/BeginnerUXContext';
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from '@/components/ui/collapsible';
import { useState } from 'react';

/**
 * Beginner Mode Toggle - Place in header or settings
 * Allows users to switch between Simple/Detailed/Technical modes
 * Part of Step 9: Beginner UX
 */
export function BeginnerModeToggle() {
  const { mode, setMode } = useBeginnerMode();
  const [open, setOpen] = useState(false);

  const modeDescriptions = {
    simple: 'Plain English, no jargon',
    detailed: 'More detail, some technical terms',
    technical: 'Full technical analysis',
  };

  const modes: Array<'simple' | 'detailed' | 'technical'> = ['simple', 'detailed', 'technical'];

  return (
    <div className="relative">
      <Collapsible open={open} onOpenChange={setOpen}>
        <CollapsibleTrigger asChild>
          <Button
            variant="outline"
            size="sm"
            className="gap-2"
          >
            <Settings className="w-4 h-4" />
            <span className="hidden sm:inline text-xs font-semibold">Display: <span className="capitalize">{mode}</span></span>
          </Button>
        </CollapsibleTrigger>

        <CollapsibleContent className="absolute right-0 mt-2 w-80 bg-card border border-border rounded-lg shadow-lg z-50 p-4 space-y-4">
          <div className="space-y-1">
            <p className="font-semibold text-sm">Jax Display Mode</p>
            <p className="text-xs text-muted-foreground">Choose how much detail to show across beginner-focused pages.</p>
          </div>

          <div className="space-y-2.5">
            {modes.map((m) => (
              <button
                key={m}
                type="button"
                className={
                  mode === m
                    ? 'w-full rounded-md border bg-accent px-3 py-2.5 text-left text-accent-foreground'
                    : 'w-full rounded-md border border-border bg-background px-3 py-2.5 text-left hover:bg-muted'
                }
                onClick={() => {
                  setMode(m);
                  setOpen(false);
                }}
                aria-pressed={mode === m}
              >
                <div className="flex items-start gap-2">
                  <div className="flex-1">
                    <p className="font-medium capitalize">{m}</p>
                    <p className="text-xs text-muted-foreground mt-1">{modeDescriptions[m]}</p>
                  </div>
                  {mode === m && <Check className="h-4 w-4 mt-0.5 shrink-0" />}
                </div>
              </button>
            ))}
          </div>

          <div className="text-xs text-muted-foreground space-y-1.5 pt-3 border-t">
            <p>
              <strong>Simple:</strong> Perfect for traders new to ETF trading.
            </p>
            <p>
              <strong>Detailed:</strong> For traders with some experience.
            </p>
            <p>
              <strong>Technical:</strong> For advanced traders.
            </p>
          </div>
        </CollapsibleContent>
      </Collapsible>
    </div>
  );
}
