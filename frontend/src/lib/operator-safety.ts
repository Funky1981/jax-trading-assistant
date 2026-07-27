import type { OperatorEvidenceOverview } from '@/data/operator-evidence-service';

export function isPaperSafe(data: OperatorEvidenceOverview | undefined): boolean {
  return interpretOperatorSafety(data).state === 'safe';
}

export type SafetyState = 'safe' | 'unsafe' | 'unknown';

export interface SafetyCheck {
  key: 'mode' | 'liveTrading' | 'execution' | 'worker' | 'broker' | 'leverage';
  label: string;
  value: string;
  state: SafetyState;
  explanation: string;
}

export function interpretOperatorSafety(data: OperatorEvidenceOverview | undefined) {
  const checks: SafetyCheck[] = [
    check(
      'mode',
      'Mode',
      data?.runtimeMode,
      'paper',
      'Paper',
      'Live',
      'Jax evaluates evidence without placing live orders.',
    ),
    check(
      'liveTrading',
      'Live trading',
      data?.allowLiveTrading,
      false,
      'Off',
      'On',
      'Whether Jax may interact with a live trading environment.',
    ),
    check(
      'execution',
      'Execution',
      data?.executionEnabled,
      false,
      'Disabled',
      'Enabled',
      'Whether execution-side activity is permitted.',
    ),
    check(
      'worker',
      'Execution worker',
      data?.executionWorkerEnabled,
      false,
      'Stopped',
      'Running',
      'The background execution worker must remain stopped.',
    ),
    check(
      'broker',
      'Broker execution',
      data?.brokerExecutionAllowed,
      false,
      'Not allowed',
      'Allowed',
      'Whether activity may be sent to a broker.',
    ),
    leverageCheck(data?.maximumLeverage),
  ];
  const state: SafetyState = checks.some((item) => item.state === 'unsafe')
    ? 'unsafe'
    : checks.some((item) => item.state === 'unknown')
      ? 'unknown'
      : 'safe';
  return { state, checks };
}

function check(
  key: SafetyCheck['key'],
  label: string,
  value: string | boolean | null | undefined,
  safeValue: string | boolean,
  safeLabel: string,
  unsafeLabel: string,
  explanation: string,
): SafetyCheck {
  if (value === null || value === undefined || value === '')
    return { key, label, value: 'Unknown', state: 'unknown', explanation };
  const safe = value === safeValue;
  return {
    key,
    label,
    value: safe ? safeLabel : unsafeLabel,
    state: safe ? 'safe' : 'unsafe',
    explanation,
  };
}

function leverageCheck(value: number | null | undefined): SafetyCheck {
  const base = {
    key: 'leverage' as const,
    label: 'Maximum leverage',
    explanation: 'Maximum permitted exposure relative to assumed account equity.',
  };
  if (value === null || value === undefined || !Number.isFinite(value))
    return { ...base, value: 'Unknown', state: 'unknown' };
  return { ...base, value: `${value}x`, state: value <= 1 ? 'safe' : 'unsafe' };
}
