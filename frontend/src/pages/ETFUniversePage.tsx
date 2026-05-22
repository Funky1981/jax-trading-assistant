import { useBeginnerMode } from '@/context/BeginnerUXContextValue';
import { ETFUniverseScreen } from '@/components/trading/ETFUniverseScreen';

/**
 * ETF Universe page - shows all approved ETFs
 * Part of Step 9: Beginner UX
 */
export function ETFUniversePage() {
  const { mode } = useBeginnerMode();

  return (
    <div className="overflow-auto">
      <ETFUniverseScreen
        mode={mode}
        onSelectETF={(symbol) => {
          console.log('Selected ETF:', symbol);
          // Could navigate to a detail page or filter approvals by this ETF
        }}
      />
    </div>
  );
}
