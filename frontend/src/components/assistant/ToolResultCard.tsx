import { type ChatMessage as ChatMessageType } from '@/data/chat-service';
import { Badge } from '@/components/ui/badge';

interface ToolResultCardProps {
  message: ChatMessageType;
}

interface ToolResultShape {
  ok?: boolean;
  data?: unknown;
  error?: string;
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return Boolean(value) && typeof value === 'object' && !Array.isArray(value);
}

function asString(value: unknown): string {
  return typeof value === 'string' ? value : '';
}

function asNumber(value: unknown): number | undefined {
  return typeof value === 'number' ? value : undefined;
}

function formatMaybeDate(value: unknown): string {
  const raw = asString(value);
  if (!raw) return '-';
  const date = new Date(raw);
  return Number.isNaN(date.getTime()) ? raw : date.toLocaleString();
}

function formatConfidence(value: unknown): string {
  const num = asNumber(value);
  return typeof num === 'number' ? `${Math.round(num * 100)}%` : '-';
}

function compactId(value: unknown): string {
  const raw = asString(value);
  if (!raw) return '-';
  return raw.length > 12 ? `${raw.slice(0, 8)}...` : raw;
}

function basename(value: string): string {
  const normalized = value.replace(/\\/g, '/');
  const parts = normalized.split('/');
  return parts[parts.length - 1] || value;
}

function Field({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex items-center gap-2">
      <span className="text-[11px] font-medium uppercase tracking-wide text-muted-foreground">{label}</span>
      <span className="text-sm text-foreground/85">{value}</span>
    </div>
  );
}

function IdField({ label, value }: { label: string; value: unknown }) {
  return (
    <div className="flex items-center gap-2">
      <span className="text-[11px] font-medium uppercase tracking-wide text-muted-foreground">{label}</span>
      <code className="rounded bg-muted px-2 py-0.5 text-xs">{compactId(value)}</code>
    </div>
  );
}

function CandidateList({ items }: { items: Record<string, unknown>[] }) {
  return (
    <div className="space-y-2">
      {items.map((item, index) => (
        <div key={`${asString(item.id)}-${index}`} className="rounded border border-border bg-background/70 p-3">
          <div className="flex flex-wrap items-center gap-2">
            <span className="font-semibold">{asString(item.symbol) || 'Unknown symbol'}</span>
            {asString(item.signalType) && (
              <Badge variant={asString(item.signalType) === 'BUY' ? 'default' : 'destructive'} className="font-mono">
                {asString(item.signalType)}
              </Badge>
            )}
            {asString(item.status) && <Badge variant="outline">{asString(item.status)}</Badge>}
            {asString(item.blockedReasonCode) && <Badge variant="warning">{asString(item.blockedReasonCode)}</Badge>}
          </div>
          <div className="mt-2 grid gap-2 md:grid-cols-2">
            <IdField label="Candidate" value={item.id} />
            <IdField label="Signal" value={item.signalId} />
            <IdField label="Artifact" value={item.artifactId} />
            <Field label="Confidence" value={formatConfidence(item.confidence)} />
            <Field label="Detected" value={formatMaybeDate(item.detectedAt)} />
            <Field label="Blocked" value={formatMaybeDate(item.blockedAt)} />
          </div>
          {asString(item.blockReason) && <p className="mt-2 text-sm text-foreground/80">{asString(item.blockReason)}</p>}
        </div>
      ))}
    </div>
  );
}

function KnowledgeMatches({ items }: { items: Record<string, unknown>[] }) {
  return (
    <div className="space-y-2">
      {items.map((item, index) => {
        const rawPath = asString(item.path);
        const label = rawPath ? basename(rawPath) : `Match ${index + 1}`;
        return (
          <div key={`${rawPath}-${index}`} className="rounded border border-border bg-background/70 p-3">
            <div className="flex flex-wrap items-center gap-2">
              <Badge variant="secondary">Knowledge</Badge>
              <span className="font-semibold">{label}</span>
            </div>
            {rawPath && <p className="mt-1 break-all font-mono text-[11px] text-muted-foreground">{rawPath}</p>}
            {asString(item.excerpt) && <p className="mt-2 whitespace-pre-wrap text-sm text-foreground/80">{asString(item.excerpt)}</p>}
          </div>
        );
      })}
    </div>
  );
}

function DetailCard({ item }: { item: Record<string, unknown> }) {
  return (
    <div className="rounded border border-border bg-background/70 p-3">
      <div className="flex flex-wrap items-center gap-2">
        {asString(item.symbol) && <span className="font-semibold">{asString(item.symbol)}</span>}
        {asString(item.signalType) && (
          <Badge variant={asString(item.signalType) === 'BUY' ? 'default' : 'destructive'} className="font-mono">
            {asString(item.signalType)}
          </Badge>
        )}
        {asString(item.status) && <Badge variant="outline">{asString(item.status)}</Badge>}
        {asString(item.blockedReasonCode) && <Badge variant="warning">{asString(item.blockedReasonCode)}</Badge>}
      </div>
      <div className="mt-2 grid gap-2 md:grid-cols-2">
        <IdField label="Candidate" value={item.id} />
        <IdField label="Signal" value={item.signalId} />
        <IdField label="Strategy" value={item.strategyId} />
        <IdField label="Artifact" value={item.artifactId} />
        <IdField label="Instruction" value={item.execution_instruction_id ?? item.executionInstructionId} />
        <IdField label="Trade" value={item.trade_id ?? item.tradeId} />
        <Field label="Confidence" value={formatConfidence(item.confidence)} />
        <Field label="Detected" value={formatMaybeDate(item.detectedAt)} />
      </div>
      {asString(item.block_reason ?? item.blockReason) && (
        <p className="mt-2 text-sm text-foreground/80">{asString(item.block_reason ?? item.blockReason)}</p>
      )}
      {asString(item.reasoning) && <p className="mt-2 whitespace-pre-wrap text-sm text-foreground/80">{asString(item.reasoning)}</p>}
    </div>
  );
}

function renderStructuredData(toolName: string, data: unknown) {
  if (Array.isArray(data) && data.every(isRecord)) {
    if (['list_pending_approvals', 'list_recent_blocked_candidates', 'search_candidates'].includes(toolName)) {
      return <CandidateList items={data} />;
    }
    if (toolName === 'query_knowledge') {
      return <KnowledgeMatches items={data} />;
    }
  }

  if (isRecord(data) && ['get_candidate_trade', 'explain_trade_blockers', 'get_trade', 'get_signal'].includes(toolName)) {
    return <DetailCard item={data} />;
  }

  return (
    <pre className="overflow-x-auto whitespace-pre-wrap break-all text-foreground/80">
      {JSON.stringify(data, null, 2)}
    </pre>
  );
}

export function ToolResultCard({ message }: ToolResultCardProps) {
  const result = message.toolResult as ToolResultShape | null;
  if (!result) return null;

  return (
    <div className="my-1 rounded border border-border bg-card p-3 text-xs">
      <div className="mb-2 flex items-center gap-2">
        <Badge variant={result.ok ? 'default' : 'destructive'}>{result.ok ? 'OK' : 'Error'}</Badge>
        <span className="font-mono text-muted-foreground">{message.toolName as string}</span>
      </div>
      {result.ok && result.data !== undefined && renderStructuredData(asString(message.toolName), result.data)}
      {!result.ok && result.error && <p className="text-destructive">{result.error}</p>}
      <details className="mt-3 rounded border border-border/70 bg-background/50 p-2">
        <summary className="cursor-pointer text-[11px] font-medium uppercase tracking-wide text-muted-foreground">
          Tool payload
        </summary>
        <div className="mt-2 grid gap-2 md:grid-cols-2">
          <div>
            <p className="mb-1 text-[11px] uppercase tracking-wide text-muted-foreground">Args</p>
            <pre className="overflow-x-auto whitespace-pre-wrap break-all text-foreground/80">
              {JSON.stringify(message.toolArgs ?? {}, null, 2)}
            </pre>
          </div>
          <div>
            <p className="mb-1 text-[11px] uppercase tracking-wide text-muted-foreground">Raw result</p>
            <pre className="overflow-x-auto whitespace-pre-wrap break-all text-foreground/80">
              {JSON.stringify(result, null, 2)}
            </pre>
          </div>
        </div>
      </details>
    </div>
  );
}
