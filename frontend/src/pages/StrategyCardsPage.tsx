import { useBeginnerMode } from '@/context/BeginnerUXContextValue';
import { StrategyCardsScreen } from '@/components/trading/StrategyCardsScreen';
import { useNavigate } from 'react-router-dom';

/**
 * Strategy Cards page - shows beginner-friendly strategy explanations
 * Part of Step 9: Beginner UX
 */
export function StrategyCardsPage() {
  const { mode } = useBeginnerMode();
  const navigate = useNavigate();

  return (
    <div className="overflow-auto">
      <StrategyCardsScreen
        mode={mode}
        onSelectStrategy={(strategy) => {
          const params = new URLSearchParams({
            guidedStrategy: strategy.id,
            strategyName: strategy.name,
            symbols: strategy.relatedETFs.slice(0, 3).join(','),
          });
          navigate(`/research?${params.toString()}`);
        }}
        showIntroduction={true}
      />
    </div>
  );
}
