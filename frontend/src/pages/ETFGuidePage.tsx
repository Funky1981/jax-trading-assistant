import { useBeginnerMode } from '@/context/BeginnerUXContextValue';
import { ModuleGuideLayout } from '@/components/trading/ModuleGuideLayout';

export function ETFGuidePage() {
  const { mode } = useBeginnerMode();
  const isBeginner = mode === 'simple' || mode === 'detailed';

  return (
    <ModuleGuideLayout
      title="ETF Beginner Guide"
      subtitle="This module is focused on policy-gated ETF workflows with approvals, evidence review, and controlled entry paths."
      isBeginner={isBeginner}
      checklist={[
        { title: 'Confirm eligibility', description: 'Use ETF Universe to verify approved symbols and policy status before any action.' },
        { title: 'Review strategy context', description: 'Check strategies, timeline, and trading modes to align entry logic to current market regime.' },
        { title: 'Use approvals workflow', description: 'ETF entries should flow through candidate evidence and approvals where policy requires it.' },
        { title: 'Execute and monitor', description: 'Submit through trading tools, then manage exposure with blotter and position protection.' },
      ]}
      glossary={[
        { title: 'ETF Eligibility', description: 'Policy decision indicating whether an ETF symbol is currently approved for workflow use.' },
        { title: 'Approval Gate', description: 'A risk/control checkpoint that can require candidate evidence and explicit approval before execution.' },
        { title: 'Evidence Pack', description: 'Structured rationale, market context, and audit trace supporting a candidate trade decision.' },
        { title: 'Trading Mode', description: 'Preset operating profile that guides strategy aggressiveness and control thresholds.' },
      ]}
      actions={[
        { label: 'Open ETF Trading', to: '/etf/trading' },
        { label: 'Open ETF Universe', to: '/etf/universe', variant: 'outline' },
        { label: 'Open ETF Approvals', to: '/etf/approvals', variant: 'outline' },
      ]}
    />
  );
}
