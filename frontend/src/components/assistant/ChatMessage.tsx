import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import { ExternalLink } from 'lucide-react';
import { chatService, type ChatMessage, type ChatTrace } from '@/data/chat-service';
import { Badge } from '@/components/ui/badge';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { cn } from '@/lib/utils';

interface ChatMessageProps {
  message: ChatMessage;
}

interface EvidenceItemShape {
  evidenceLevel?: string;
}

interface EvidenceBundleShape {
  items?: EvidenceItemShape[];
}

function shortId(value: string) {
  return value.length > 12 ? `${value.slice(0, 8)}...` : value;
}

function evidenceBadge(message: ChatMessage): { label: string; variant: 'success' | 'warning' | 'destructive' } | null {
  const bundle = message.evidenceBundle as EvidenceBundleShape | undefined;
  const levels = bundle?.items?.map((item) => item.evidenceLevel).filter(Boolean) ?? [];
  if (levels.length === 0) {
    return null;
  }
  if (levels.includes('hard_internal_data')) {
    return { label: 'High evidence', variant: 'success' };
  }
  if (levels.includes('derived_internal_data')) {
    return { label: 'Mixed evidence', variant: 'warning' };
  }
  return { label: 'Weak evidence', variant: 'destructive' };
}

function TraceDetailDialog({ traceId, open, onOpenChange }: { traceId: string; open: boolean; onOpenChange: (open: boolean) => void }) {
  const { data, isLoading, isError } = useQuery({
    queryKey: ['chat-trace', traceId],
    queryFn: () => chatService.getTrace(traceId),
    enabled: open,
    staleTime: 60_000,
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-3xl">
        <DialogHeader>
          <DialogTitle>Trace {shortId(traceId)}</DialogTitle>
          <DialogDescription>Audit trail for this advisory answer.</DialogDescription>
        </DialogHeader>
        {isLoading && <p className="text-sm text-muted-foreground">Loading trace…</p>}
        {isError && <p className="text-sm text-destructive">Failed to load trace.</p>}
        {data && <TraceDetail trace={data} />}
      </DialogContent>
    </Dialog>
  );
}

function TraceDetail({ trace }: { trace: ChatTrace }) {
  return (
    <div className="max-h-[70vh] space-y-4 overflow-y-auto pr-1 text-sm">
      <section className="rounded border border-border bg-muted/20 p-3">
        <div className="grid gap-2 md:grid-cols-2">
          <div>
            <p className="text-xs uppercase tracking-wide text-muted-foreground">Trace ID</p>
            <code className="text-xs">{trace.traceId}</code>
          </div>
          <div>
            <p className="text-xs uppercase tracking-wide text-muted-foreground">Created</p>
            <p>{new Date(trace.createdAt).toLocaleString()}</p>
          </div>
          <div className="md:col-span-2">
            <p className="text-xs uppercase tracking-wide text-muted-foreground">Question</p>
            <p className="whitespace-pre-wrap">{trace.question}</p>
          </div>
          {!!trace.finalAnswer && (
            <div className="md:col-span-2">
              <p className="text-xs uppercase tracking-wide text-muted-foreground">Final Answer</p>
              <p className="whitespace-pre-wrap">{trace.finalAnswer}</p>
            </div>
          )}
        </div>
      </section>

      {trace.toolRuns && trace.toolRuns.length > 0 && (
        <section>
          <h3 className="mb-2 text-sm font-semibold">Tool Runs</h3>
          <div className="space-y-2">
            {trace.toolRuns.map((run, index) => (
              <div key={`${run.call.name}-${index}`} className="rounded border border-border bg-background/70 p-3">
                <div className="flex items-center gap-2">
                  <span className="font-medium">{run.call.name}</span>
                  {run.error ? (
                    <span className="text-xs text-destructive">Error</span>
                  ) : (
                    <span className="text-xs text-emerald-600">OK</span>
                  )}
                </div>
                <div className="mt-2 grid gap-2 md:grid-cols-2">
                  <div>
                    <p className="text-xs uppercase tracking-wide text-muted-foreground">Args</p>
                    <pre className="overflow-x-auto whitespace-pre-wrap break-all rounded bg-muted/40 p-2 text-xs">
                      {JSON.stringify(run.call.args, null, 2)}
                    </pre>
                  </div>
                  <div>
                    <p className="text-xs uppercase tracking-wide text-muted-foreground">Result</p>
                    <pre className="overflow-x-auto whitespace-pre-wrap break-all rounded bg-muted/40 p-2 text-xs">
                      {run.error ? run.error : JSON.stringify(run.result, null, 2)}
                    </pre>
                  </div>
                </div>
              </div>
            ))}
          </div>
        </section>
      )}

      {trace.validationAttempts && trace.validationAttempts.length > 0 && (
        <section>
          <h3 className="mb-2 text-sm font-semibold">Validation</h3>
          <div className="space-y-2">
            {trace.validationAttempts.map((attempt) => (
              <div key={attempt.attempt} className="rounded border border-border bg-background/70 p-3">
                <div className="flex items-center gap-2">
                  <span className="font-medium">Attempt {attempt.attempt}</span>
                  <span className={cn('text-xs', attempt.accepted ? 'text-emerald-600' : 'text-destructive')}>
                    {attempt.accepted ? 'Accepted' : 'Rejected'}
                  </span>
                </div>
                <p className="mt-2 whitespace-pre-wrap">{attempt.answer}</p>
                {attempt.reasons && attempt.reasons.length > 0 && (
                  <ul className="mt-2 list-disc pl-4 text-xs text-muted-foreground">
                    {attempt.reasons.map((reason, index) => (
                      <li key={`${attempt.attempt}-${index}`}>{reason}</li>
                    ))}
                  </ul>
                )}
              </div>
            ))}
          </div>
        </section>
      )}
    </div>
  );
}

/** Renders an assistant reply with GitHub-Flavored Markdown support. */
function AssistantContent({ content }: { content: string }) {
  return (
    <ReactMarkdown
      remarkPlugins={[remarkGfm]}
      components={{
        p: ({ children }) => <p className="mb-1 last:mb-0 whitespace-pre-wrap break-words">{children}</p>,
        h1: ({ children }) => <h1 className="mb-1 mt-2 text-base font-bold">{children}</h1>,
        h2: ({ children }) => <h2 className="mb-1 mt-2 text-sm font-bold">{children}</h2>,
        h3: ({ children }) => <h3 className="mb-0.5 mt-1.5 text-sm font-semibold">{children}</h3>,
        ul: ({ children }) => <ul className="mb-1 list-inside list-disc space-y-0.5">{children}</ul>,
        ol: ({ children }) => <ol className="mb-1 list-inside list-decimal space-y-0.5">{children}</ol>,
        li: ({ children }) => <li className="text-sm">{children}</li>,
        code: ({ className, children, ...props }) => {
          const isBlock = className?.startsWith('language-');
          return isBlock ? (
            <code className="my-1 block overflow-x-auto whitespace-pre rounded bg-background/60 px-2 py-1 font-mono text-xs" {...props}>
              {children}
            </code>
          ) : (
            <code className="rounded bg-background/60 px-1 py-0.5 font-mono text-xs" {...props}>
              {children}
            </code>
          );
        },
        pre: ({ children }) => <pre className="my-1 overflow-x-auto">{children}</pre>,
        strong: ({ children }) => <strong className="font-semibold">{children}</strong>,
        em: ({ children }) => <em className="italic">{children}</em>,
        blockquote: ({ children }) => (
          <blockquote className="my-1 border-l-2 border-muted-foreground/40 pl-2 italic text-muted-foreground">{children}</blockquote>
        ),
        a: ({ href, children }) => (
          <a href={href} className="text-primary underline hover:opacity-80" target="_blank" rel="noopener noreferrer">
            {children}
          </a>
        ),
        table: ({ children }) => <table className="my-1 w-full border-collapse text-xs">{children}</table>,
        th: ({ children }) => <th className="border border-border bg-muted/50 px-2 py-0.5 text-left font-semibold">{children}</th>,
        td: ({ children }) => <td className="border border-border px-2 py-0.5">{children}</td>,
      }}
    >
      {content}
    </ReactMarkdown>
  );
}

export function ChatMessageBubble({ message }: ChatMessageProps) {
  const isUser = message.role === 'user';
  const isTool = message.role === 'tool';
  const [traceOpen, setTraceOpen] = useState(false);
  const evidence = evidenceBadge(message);

  if (isTool) {
    return (
      <div className="my-1 rounded bg-muted px-3 py-2 text-xs font-mono text-muted-foreground">
        <span className="font-semibold text-foreground">Tool: {message.toolName as string}</span>
        {!!message.toolResult && (
          <pre className="mt-1 overflow-x-auto whitespace-pre-wrap break-all">
            {JSON.stringify(message.toolResult, null, 2)}
          </pre>
        )}
      </div>
    );
  }

  return (
    <>
      <div className={cn('flex', isUser ? 'justify-end' : 'justify-start')}>
        <div
          className={cn(
            'max-w-[80%] rounded-lg px-3 py-2 text-sm',
            isUser
              ? 'bg-primary text-primary-foreground'
              : 'bg-muted text-foreground'
          )}
        >
          {isUser ? (
            <p className="whitespace-pre-wrap break-words">{message.content}</p>
          ) : (
            <AssistantContent content={message.content} />
          )}
          <div className={cn('mt-1 flex items-center gap-3 text-xs', isUser ? 'text-primary-foreground/60' : 'text-muted-foreground')}>
            <span>{new Date(message.createdAt).toLocaleTimeString()}</span>
            {!isUser && evidence && <Badge variant={evidence.variant}>{evidence.label}</Badge>}
            {!isUser && message.traceId && (
              <button
                type="button"
                className="inline-flex items-center gap-1 underline-offset-2 hover:text-foreground hover:underline"
                onClick={() => setTraceOpen(true)}
              >
                <ExternalLink className="h-3 w-3" />
                Trace {shortId(message.traceId)}
              </button>
            )}
          </div>
        </div>
      </div>
      {message.traceId && <TraceDetailDialog traceId={message.traceId} open={traceOpen} onOpenChange={setTraceOpen} />}
    </>
  );
}
