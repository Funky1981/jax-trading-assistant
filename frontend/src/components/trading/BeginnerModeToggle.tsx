import { Settings } from 'lucide-react';
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
    <Collapsible open={open} onOpenChange={setOpen}>
      <CollapsibleTrigger asChild>
        <Button
          variant="outline"
          size="sm"
          className="gap-2"
        >
          <Settings className="w-4 h-4" />
          <span className="hidden sm:inline text-xs font-semibold capitalize">{mode}</span>
        </Button>
      </CollapsibleTrigger>

      <CollapsibleContent className="absolute right-0 mt-2 w-64 bg-card border border-border rounded-lg shadow-lg z-50 p-4 space-y-3">
        <div className="space-y-1">
          <p className="font-semibold text-sm">Jax Display Mode</p>
          <p className="text-xs text-muted-foreground">Choose how much detail you want to see</p>
        </div>

        <div className="space-y-2">
          {modes.map((m) => (
            <Button
              key={m}
              variant={mode === m ? 'default' : 'outline'}
              size="sm"
              className="w-full justify-start"
              onClick={() => {
                setMode(m);
                setOpen(false);
              }}
            >
              <div className="text-left flex-1">
                <p className="font-medium capitalize">{m}</p>
                <p className="text-xs text-muted-foreground">{modeDescriptions[m]}</p>
              </div>
            </Button>
          ))}
        </div>

        <div className="text-xs text-muted-foreground space-y-1 pt-2 border-t">
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
  );
}
