import { useEffect, useMemo, useRef, useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { AlertCircle, Plus, Send, Wrench, X } from 'lucide-react';
import { chatService, type AssistantTool, type ChatMessage, type ChatSession } from '@/data/chat-service';
import { candidatesService } from '@/data/approvals-service';
import { instancesService } from '@/data/instances-service';
import { signalsService } from '@/data/signals-service';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { ChatMessageBubble } from './ChatMessage';
import { ToolResultCard } from './ToolResultCard';

const SUGGESTED_PROMPTS = [
  'What candidate trades are currently waiting for my approval?',
  'Which candidates were recently blocked, and why?',
  'What strategies are currently active?',
  'Summarise the most recent research run',
  'What do the knowledge docs say about paper readiness?',
  'Are there any high-confidence signals I should be aware of?',
];

const TOOL_HELP_TEXT: Record<string, string> = {
  list_pending_approvals: 'Leave the value blank for the default queue size or pick a quick limit below.',
  list_recent_blocked_candidates: 'Use this to scan the latest blocked setups and reason codes.',
  search_candidates: 'Search by symbol or status such as blocked, approved, submitted, or filled.',
  search_research_runs: 'Filter recent orchestration and research runs by symbol.',
  query_knowledge: 'Search local markdown docs, runbooks, and finish-plan notes.',
  get_candidate_trade: 'Select a candidate to inspect provenance, blockers, and execution linkage.',
  explain_trade_blockers: 'Pick a candidate to explain why it was blocked or rejected.',
};

const TOOL_QUICK_VALUES: Record<string, Array<{ label: string; value: string }>> = {
  list_pending_approvals: [
    { label: 'Default', value: '' },
    { label: '5', value: '5' },
    { label: '10', value: '10' },
    { label: '20', value: '20' },
  ],
  list_recent_blocked_candidates: [
    { label: 'Default', value: '' },
    { label: '5', value: '5' },
    { label: '10', value: '10' },
    { label: '20', value: '20' },
  ],
  search_candidates: [
    { label: 'blocked', value: 'blocked' },
    { label: 'approved', value: 'approved' },
    { label: 'filled', value: 'filled' },
    { label: 'AAPL', value: 'AAPL' },
  ],
  search_research_runs: [
    { label: 'AAPL', value: 'AAPL' },
    { label: 'MSFT', value: 'MSFT' },
    { label: 'NVDA', value: 'NVDA' },
  ],
  query_knowledge: [
    { label: 'paper readiness', value: 'paper readiness' },
    { label: 'approval flow', value: 'approval flow' },
    { label: 'flatten proof', value: 'flatten proof' },
    { label: 'shadow parity', value: 'shadow parity' },
  ],
};

interface ChatPanelProps {
  sessionId?: string;
  onSessionCreated?: (id: string) => void;
}

export function ChatPanel({ sessionId: initialSessionId, onSessionCreated }: ChatPanelProps) {
  const qc = useQueryClient();
  const [sessionId, setSessionId] = useState<string | undefined>(initialSessionId);
  const [draft, setDraft] = useState('');
  const [optimisticContent, setOptimisticContent] = useState<string | null>(null);
  const [showTools, setShowTools] = useState(false);
  const [selectedTool, setSelectedTool] = useState<AssistantTool | null>(null);
  const [toolArgValue, setToolArgValue] = useState('');
  const bottomRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    setSessionId(initialSessionId);
    setOptimisticContent(null);
  }, [initialSessionId]);

  const { data: history = [], isLoading } = useQuery({
    queryKey: ['chat-history', sessionId],
    queryFn: () => chatService.getHistory(sessionId!),
    enabled: !!sessionId,
    refetchInterval: false,
  });

  const { data: toolsData } = useQuery({
    queryKey: ['chat-tools'],
    queryFn: () => chatService.getTools(),
    staleTime: Infinity,
  });
  const tools = toolsData?.tools ?? [];
  const runtimeMode = toolsData?.mode ?? 'research';
  const harnessEnabled = toolsData?.harnessEnabled ?? false;
  const shadowMode = toolsData?.shadowMode ?? false;
  const sessionRateLimit = toolsData?.sessionRateLimitPerMinute;

  const isCandidateTool = !!selectedTool && ['get_candidate_trade', 'explain_trade_blockers'].includes(selectedTool.name);
  const isSignalTool = !!selectedTool && selectedTool.name === 'get_signal';
  const isInstanceTool = !!selectedTool && selectedTool.name === 'get_strategy_instance';

  const { data: candidateOptions = [] } = useQuery({
    queryKey: ['assistant-entity-candidates'],
    queryFn: async () => {
      const items = await candidatesService.list({ limit: 30 });
      return items.map((c) => ({ id: c.id, label: `${c.symbol} ${c.signalType} - ${c.status}` }));
    },
    staleTime: 30_000,
    enabled: isCandidateTool,
  });

  const { data: signalOptions = [] } = useQuery({
    queryKey: ['assistant-entity-signals'],
    queryFn: async () => {
      const resp = await signalsService.list({ limit: 30 });
      return resp.signals.map((s) => ({ id: s.id, label: `${s.symbol} ${s.signal_type} - ${s.status}` }));
    },
    staleTime: 30_000,
    enabled: isSignalTool,
  });

  const { data: instanceOptions = [] } = useQuery({
    queryKey: ['assistant-entity-instances'],
    queryFn: async () => {
      const items = await instancesService.list();
      return items.map((i) => ({ id: i.id, label: i.name }));
    },
    staleTime: 60_000,
    enabled: isInstanceTool,
  });

  const entityOptions = isCandidateTool ? candidateOptions : isSignalTool ? signalOptions : isInstanceTool ? instanceOptions : [];
  const quickValues = selectedTool ? TOOL_QUICK_VALUES[selectedTool.name] ?? [] : [];
  const helperText = selectedTool ? TOOL_HELP_TEXT[selectedTool.name] ?? selectedTool.description : '';

  const selectedToolPlaceholder = useMemo(() => {
    if (!selectedTool) return 'Ask about a trade, strategy, or scenario...';
    return `Ask a question and attach ${selectedTool.description.toLowerCase()}...`;
  }, [selectedTool]);

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [history.length, optimisticContent]);

  const sendMutation = useMutation({
    mutationFn: ({ content, toolCall }: { content: string; toolCall?: { name: string; args: Record<string, string> } }) =>
      chatService.sendMessage({ sessionId, content, toolCall }),
    onSuccess: (resp) => {
      setOptimisticContent(null);
      if (!sessionId) {
        setSessionId(resp.sessionId);
        onSessionCreated?.(resp.sessionId);
      }
      qc.invalidateQueries({ queryKey: ['chat-history', resp.sessionId] });
      qc.invalidateQueries({ queryKey: ['chat-sessions'] });
    },
    onError: () => {
      setOptimisticContent(null);
    },
  });

  const handleSend = () => {
    const content = draft.trim();
    if (!content || sendMutation.isPending) return;
    setDraft('');
    setOptimisticContent(content);

    let toolCall: { name: string; args: Record<string, string> } | undefined;
    if (selectedTool) {
      toolCall = { name: selectedTool.name, args: { [selectedTool.argKey]: toolArgValue.trim() } };
      setSelectedTool(null);
      setToolArgValue('');
      setShowTools(false);
    }

    sendMutation.mutate({ content, toolCall });
  };

  const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      handleSend();
    }
  };

  return (
    <div className="flex h-full flex-col">
      <div className="mb-2 flex items-start gap-2 rounded-md border border-yellow-400/40 bg-yellow-50/10 px-3 py-2 text-xs text-yellow-600 dark:text-yellow-400">
        <AlertCircle className="mt-0.5 h-3 w-3 shrink-0" />
        <span>
          Jax Assistant is <strong>advisory only</strong>. It cannot place orders or approve trades on your behalf.
        </span>
      </div>
      <div className="mb-3 flex flex-wrap items-center gap-2 text-xs">
        <Badge variant={runtimeMode === 'live' ? 'destructive' : runtimeMode === 'paper' ? 'warning' : 'secondary'}>
          {runtimeMode} mode
        </Badge>
        <Badge variant={harnessEnabled ? 'success' : 'outline'}>{harnessEnabled ? 'Harness on' : 'Harness off'}</Badge>
        {shadowMode && <Badge variant="outline">Shadow validation</Badge>}
        {typeof sessionRateLimit === 'number' && <span className="text-muted-foreground">Session limit: {sessionRateLimit}/min</span>}
      </div>

      <div className="flex-1 overflow-y-auto pr-1">
        {!sessionId && !isLoading && (
          <div className="flex flex-col items-center gap-4 py-6">
            <p className="text-center text-sm text-muted-foreground">Start a conversation, or pick a suggestion:</p>
            <div className="flex w-full max-w-md flex-col gap-2">
              {SUGGESTED_PROMPTS.map((prompt) => (
                <button
                  key={prompt}
                  onClick={() => {
                    setDraft(prompt);
                  }}
                  className="rounded-lg border border-border bg-muted/40 px-3 py-2 text-left text-xs text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
                >
                  {prompt}
                </button>
              ))}
            </div>
          </div>
        )}
        {isLoading && <p className="py-4 text-center text-sm text-muted-foreground">Loading history...</p>}

        <div className="space-y-2 pb-2">
          {history.map((msg: ChatMessage) =>
            msg.role === 'tool' ? <ToolResultCard key={msg.id} message={msg} /> : <ChatMessageBubble key={msg.id} message={msg} />
          )}

          {optimisticContent && (
            <div className="flex justify-end">
              <div className="max-w-[80%] rounded-lg bg-primary px-3 py-2 text-sm text-primary-foreground">
                <p className="whitespace-pre-wrap break-words">{optimisticContent}</p>
              </div>
            </div>
          )}

          {sendMutation.isPending && (
            <div className="flex justify-start">
              <div className="flex items-center gap-1 rounded-lg bg-muted px-3 py-2 text-sm text-muted-foreground">
                <span className="inline-block h-1.5 w-1.5 animate-bounce rounded-full bg-muted-foreground/60 [animation-delay:0ms]" />
                <span className="inline-block h-1.5 w-1.5 animate-bounce rounded-full bg-muted-foreground/60 [animation-delay:150ms]" />
                <span className="inline-block h-1.5 w-1.5 animate-bounce rounded-full bg-muted-foreground/60 [animation-delay:300ms]" />
              </div>
            </div>
          )}
        </div>
        <div ref={bottomRef} />
      </div>

      {tools.length > 0 && (
        <div className="border-t border-border pt-2">
          {!showTools ? (
            <Button
              variant="ghost"
              size="sm"
              className="h-7 px-2 text-xs text-muted-foreground hover:text-foreground"
              onClick={() => setShowTools(true)}
              disabled={sendMutation.isPending}
            >
              <Wrench className="mr-1 h-3 w-3" /> Use a tool
            </Button>
          ) : (
            <div className="flex flex-col gap-2">
              <div className="flex items-center gap-2">
                <select
                  className="h-8 flex-1 rounded-md border border-input bg-background px-2 text-xs text-foreground focus:outline-none focus:ring-1 focus:ring-ring"
                  title="Select tool"
                  value={selectedTool?.name ?? ''}
                  onChange={(e) => {
                    const tool = tools.find((x) => x.name === e.target.value) ?? null;
                    setSelectedTool(tool);
                    setToolArgValue('');
                  }}
                >
                  <option value="">Select tool...</option>
                  {tools.map((tool) => (
                    <option key={tool.name} value={tool.name} disabled={tool.allowed === false}>
                      {tool.description}
                    </option>
                  ))}
                </select>
                <Button
                  variant="ghost"
                  size="icon"
                  className="h-7 w-7 shrink-0 text-muted-foreground"
                  onClick={() => {
                    setShowTools(false);
                    setSelectedTool(null);
                    setToolArgValue('');
                  }}
                >
                  <X className="h-3 w-3" />
                </Button>
              </div>

              {selectedTool && (
                <>
                  <p className="text-xs text-muted-foreground">{helperText}</p>
                  {selectedTool.allowed === false && (
                    <p className="text-xs text-destructive">{selectedTool.policyReason ?? 'This tool is blocked in the current mode.'}</p>
                  )}
                  {entityOptions.length > 0 ? (
                    <select
                      className="h-8 w-full rounded-md border border-input bg-background px-2 text-xs text-foreground focus:outline-none focus:ring-1 focus:ring-ring"
                      title={`Select ${selectedTool.argLabel}`}
                      value={toolArgValue}
                      onChange={(e) => setToolArgValue(e.target.value)}
                    >
                      <option value="">Select {selectedTool.argLabel}...</option>
                      {entityOptions.map((opt) => (
                        <option key={opt.id} value={opt.id}>
                          {opt.label}
                        </option>
                      ))}
                    </select>
                  ) : (
                    <Input
                      className="h-8 text-xs"
                      placeholder={selectedTool.argLabel}
                      value={toolArgValue}
                      onChange={(e) => setToolArgValue(e.target.value)}
                      onKeyDown={(e) => {
                        if (e.key === 'Enter' && !e.shiftKey) {
                          e.preventDefault();
                          handleSend();
                        }
                      }}
                    />
                  )}
                  {quickValues.length > 0 && (
                    <div className="flex flex-wrap gap-2">
                      {quickValues.map((item) => (
                        <button
                          key={`${selectedTool.name}-${item.label}`}
                          type="button"
                          onClick={() => setToolArgValue(item.value)}
                          className={`rounded-full border px-2 py-1 text-[11px] transition-colors ${
                            toolArgValue === item.value
                              ? 'border-primary bg-primary/10 text-primary'
                              : 'border-border bg-muted/40 text-muted-foreground hover:bg-muted hover:text-foreground'
                          }`}
                        >
                          {item.label}
                        </button>
                      ))}
                    </div>
                  )}
                </>
              )}
            </div>
          )}
        </div>
      )}

      <div className="flex gap-2 border-t border-border pt-2">
        <Input
          value={draft}
          onChange={(e) => setDraft(e.target.value)}
          onKeyDown={handleKeyDown}
          placeholder={selectedToolPlaceholder}
          disabled={sendMutation.isPending || selectedTool?.allowed === false}
          className="flex-1"
        />
        <Button size="icon" onClick={handleSend} disabled={!draft.trim() || sendMutation.isPending || selectedTool?.allowed === false}>
          <Send className="h-4 w-4" />
        </Button>
      </div>

      {sendMutation.isError && <p className="mt-1 text-xs text-destructive">Failed to send message. Please try again.</p>}
    </div>
  );
}

interface SessionListProps {
  sessions: ChatSession[];
  activeId?: string;
  onSelect: (id: string) => void;
  onNew: () => void;
}

export function SessionList({ sessions, activeId, onSelect, onNew }: SessionListProps) {
  return (
    <div className="flex flex-col gap-1">
      <Button variant="outline" size="sm" className="mb-2 w-full" onClick={onNew}>
        <Plus className="mr-1 h-4 w-4" /> New Chat
      </Button>
      {sessions.map((session) => (
        <button
          key={session.id}
          onClick={() => onSelect(session.id)}
          className={`truncate rounded px-2 py-1.5 text-left text-sm transition-colors ${
            activeId === session.id
              ? 'bg-accent text-accent-foreground'
              : 'text-muted-foreground hover:bg-muted hover:text-foreground'
          }`}
        >
          {session.title || 'Untitled chat'}
        </button>
      ))}
      {sessions.length === 0 && <p className="px-2 text-xs text-muted-foreground">No previous sessions.</p>}
    </div>
  );
}
