import { useBeginnerMode } from '@/context/BeginnerUXContextValue';
import { ModuleGuideLayout } from '@/components/trading/ModuleGuideLayout';

export function EquityAlphaGuidePage() {
  const { mode } = useBeginnerMode();
  const isBeginner = mode === 'simple' || mode === 'detailed';

  return (
    <ModuleGuideLayout
      title="Equity Alpha Beginner Guide"
      subtitle="This module is focused on non-ETF equity alpha workflows: opening-range, earnings drift, and event-driven entries."
      isBeginner={isBeginner}
      checklist={[
        { title: 'Pick setup and symbol', description: 'Choose a setup style and confirm the symbol is in the Equity Alpha workflow.' },
        { title: 'Validate context', description: 'Check opening range, catalyst strength, and relative volume before entry.' },
        { title: 'Define risk first', description: 'Set stop and take-profit before submitting; keep position size inside risk limits.' },
        { title: 'Manage execution', description: 'Use blotter to manage working orders and positions to update protection after fills.' },
      ]}
      glossary={[
        { title: 'Opening Range', description: 'The high/low boundary from the first market window, often used for breakout triggers.' },
        { title: 'Earnings Drift', description: 'Directional continuation after earnings when gap and volume support the move.' },
        { title: 'Event Gap', description: 'A catalyst-driven open away from prior close that may continue after confirmation.' },
        { title: 'Volume Multiple', description: 'Current bar volume divided by baseline volume; higher values indicate stronger participation.' },
      ]}
      actions={[
        { label: 'Open Equity Alpha Trading', to: '/equity-alpha/trading' },
        { label: 'Open Equity Alpha Strategies', to: '/equity-alpha/strategies', variant: 'outline' },
        { label: 'Open Equity Alpha Timeline', to: '/equity-alpha/timeline', variant: 'outline' },
      ]}
    />
  );
}
