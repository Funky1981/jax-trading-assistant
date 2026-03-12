import { useEffect, useRef, useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Send, Plus, AlertCircle } from 'lucide-react';
import { chatService, type ChatMessage, type ChatSession } from '@/data/chat-service';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { ChatMessageBubble } from './ChatMessage';
import { ToolResultCard } from './ToolResultCard';

interface ChatPanelProps {
  sessionId?: string;
  onSessionCreated?: (id: string) => void;
}

export function ChatPanel({ sessionId: initialSessionId, onSessionCreated }: ChatPanelProps) {
  const qc = useQueryClient();
  const [sessionId, setSessionId] = useState<string | undefined>(initialSessionId);
  const [draft, setDraft] = useState('');
  // Holds the user's message while the server is processing — gives instant feedback.
  const [optimisticContent, setOptimisticContent] = useState<string | null>(null);
  const bottomRef = useRef<HTMLDivElement>(null);

  // Sync sessionId when the parent changes it (e.g. user picks a different session).
  useEffect(() => {
    setSessionId(initialSessionId);
    setOptimisticContent(null);
  }, [initialSessionId]);

  // Load history when session is selected.
  const { data: history = [], isLoading } = useQuery({
    queryKey: ['chat-history', sessionId],
    queryFn: () => chatService.getHistory(sessionId!),
    enabled: !!sessionId,
    refetchInterval: false,
  });

  // Scroll to bottom on new messages or while pending.
  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [history.length, optimisticContent]);

  const sendMutation = useMutation({
    mutationFn: (content: string) =>
      chatService.sendMessage({ sessionId, content }),
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
    sendMutation.mutate(content);
  };

  const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      handleSend();
    }
  };

  return (
    <div className="flex flex-col h-full">
      {/* Safety banner */}
      <div className="flex items-start gap-2 rounded-md border border-yellow-400/40 bg-yellow-50/10 px-3 py-2 text-xs text-yellow-600 dark:text-yellow-400 mb-2">
        <AlertCircle className="h-3 w-3 mt-0.5 shrink-0" />
        <span>Jax Assistant is <strong>advisory only</strong>. It cannot place orders or approve trades on your behalf.</span>
      </div>

      {/* Messages */}
      <div className="flex-1 overflow-y-auto pr-1">
        {!sessionId && !isLoading && (
          <p className="text-center text-sm text-muted-foreground py-8">
            Start a conversation to analyse candidate trades, signals, and strategy behaviour.
          </p>
        )}
        {isLoading && <p className="text-center text-sm text-muted-foreground py-4">Loading history…</p>}

        <div className="space-y-2 pb-2">
          {history.map((msg: ChatMessage) =>
            msg.role === 'tool' ? (
              <ToolResultCard key={msg.id} message={msg} />
            ) : (
              <ChatMessageBubble key={msg.id} message={msg} />
            )
          )}

          {/* Optimistic user bubble — shows immediately on send */}
          {optimisticContent && (
            <div className="flex justify-end">
              <div className="max-w-[80%] rounded-lg bg-primary px-3 py-2 text-sm text-primary-foreground">
                <p className="whitespace-pre-wrap break-words">{optimisticContent}</p>
              </div>
            </div>
          )}

          {/* Assistant "Thinking…" indicator while awaiting response */}
          {sendMutation.isPending && (
            <div className="flex justify-start">
              <div className="rounded-lg bg-muted px-3 py-2 text-sm text-muted-foreground flex gap-1 items-center">
                <span className="inline-block w-1.5 h-1.5 rounded-full bg-muted-foreground/60 animate-bounce [animation-delay:0ms]" />
                <span className="inline-block w-1.5 h-1.5 rounded-full bg-muted-foreground/60 animate-bounce [animation-delay:150ms]" />
                <span className="inline-block w-1.5 h-1.5 rounded-full bg-muted-foreground/60 animate-bounce [animation-delay:300ms]" />
              </div>
            </div>
          )}
        </div>
        <div ref={bottomRef} />
      </div>

      {/* Input */}
      <div className="flex gap-2 pt-2 border-t border-border">
        <Input
          value={draft}
          onChange={(e) => setDraft(e.target.value)}
          onKeyDown={handleKeyDown}
          placeholder="Ask about a trade, strategy, or scenario…"
          disabled={sendMutation.isPending}
          className="flex-1"
        />
        <Button size="icon" onClick={handleSend} disabled={!draft.trim() || sendMutation.isPending}>
          <Send className="h-4 w-4" />
        </Button>
      </div>

      {sendMutation.isError && (
        <p className="mt-1 text-xs text-destructive">
          Failed to send message. Please try again.
        </p>
      )}
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
        <Plus className="h-4 w-4 mr-1" /> New Chat
      </Button>
      {sessions.map((s) => (
        <button
          key={s.id}
          onClick={() => onSelect(s.id)}
          className={`text-left truncate rounded px-2 py-1.5 text-sm transition-colors ${
            activeId === s.id
              ? 'bg-accent text-accent-foreground'
              : 'text-muted-foreground hover:bg-muted hover:text-foreground'
          }`}
        >
          {s.title || 'Untitled chat'}
        </button>
      ))}
      {sessions.length === 0 && (
        <p className="text-xs text-muted-foreground px-2">No previous sessions.</p>
      )}
    </div>
  );
}
