import { useBeginnerMode } from '@/context/BeginnerUXContext';
import { StrategyCardsScreen } from '@/components/trading/StrategyCardsScreen';

/**
 * Strategy Cards page - shows beginner-friendly strategy explanations
 * Part of Step 9: Beginner UX
 */
export function StrategyCardsPage() {
  const { mode } = useBeginnerMode();

  return (
    <div className="overflow-auto">
      <StrategyCardsScreen
        mode={mode}
        onSelectStrategy={(strategyId) => {
          console.log('Selected strategy:', strategyId);
          // Could enable/disable strategy or navigate to strategy settings
        }}
        showIntroduction={true}
      />
    </div>
  );
}
