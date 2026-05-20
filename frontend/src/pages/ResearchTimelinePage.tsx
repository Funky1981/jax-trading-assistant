import { useBeginnerMode } from '@/context/BeginnerUXContext';
import { ResearchTimeline, generateExampleTimeline } from '@/components/trading/ResearchTimeline';

/**
 * Research Timeline page - shows narrative flow of trades
 * Part of Step 9: Beginner UX
 */
export function ResearchTimelinePage() {
  const { mode } = useBeginnerMode();

  // In production, this would fetch actual timeline events from the backend
  // For now, we show an example timeline
  const events = generateExampleTimeline();

  return (
    <div className="overflow-auto bg-background">
      <ResearchTimeline
        events={events}
        mode={mode}
        title="Trade Timeline Example"
        description={
          mode === 'simple'
            ? 'This shows how a trade flows from news detection through execution and reflection.'
            : 'Complete workflow from event detection through post-trade analysis'
        }
      />
    </div>
  );
}
