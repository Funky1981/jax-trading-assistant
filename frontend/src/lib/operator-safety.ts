import type { OperatorEvidenceOverview } from '@/data/operator-evidence-service';

export function isPaperSafe(data: OperatorEvidenceOverview | undefined): boolean {
  return Boolean(
    data &&
    data.runtimeMode === 'paper' &&
    data.allowLiveTrading === false &&
    data.executionEnabled === false &&
    data.executionWorkerEnabled === false &&
    data.brokerExecutionAllowed === false &&
    Number.isFinite(data.maximumLeverage) &&
    data.maximumLeverage <= 1,
  );
}
