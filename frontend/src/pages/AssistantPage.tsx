import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { chatService } from '@/data/chat-service';
import { ChatPanel, SessionList } from '@/components/assistant/ChatPanel';

export function AssistantPage() {
  const [activeSessionId, setActiveSessionId] = useState<string | undefined>();

  const { data: sessions = [], refetch: refetchSessions } = useQuery({
    queryKey: ['chat-sessions'],
    queryFn: () => chatService.listSessions(),
    staleTime: 10_000,
  });

  const handleNewSession = () => {
    setActiveSessionId(undefined);
  };

  const handleSessionCreated = (id: string) => {
    setActiveSessionId(id);
    refetchSessions();
  };

  return (
    <div className="flex h-[calc(100vh-4rem)] gap-0">
      {/* Sidebar: session list */}
      <aside className="w-56 shrink-0 border-r border-border p-3 overflow-y-auto">
        <p className="mb-3 text-xs font-semibold uppercase tracking-wider text-muted-foreground">
          Conversations
        </p>
        <SessionList
          sessions={sessions}
          activeId={activeSessionId}
          onSelect={setActiveSessionId}
          onNew={handleNewSession}
        />
      </aside>

      {/* Main: chat panel */}
      <main className="flex-1 flex flex-col p-4 overflow-hidden">
        <div className="mb-3">
          <h1 className="text-xl font-bold">Jax Assistant</h1>
          <p className="text-xs text-muted-foreground">
            Answers questions about candidates, signals, strategies, and research runs.
            Advisory only — cannot execute orders or approve trades.
          </p>
        </div>
        <div className="flex-1 overflow-hidden">
          <ChatPanel
            sessionId={activeSessionId}
            onSessionCreated={handleSessionCreated}
          />
        </div>
      </main>
    </div>
  );
}
