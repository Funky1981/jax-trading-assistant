import { useQuery } from '@tanstack/react-query';
import { chatService, type ChatMessage as ChatMessageType } from '@/data/chat-service';
import { Badge } from '@/components/ui/badge';

interface ToolResultCardProps {
  message: ChatMessageType;
}

export function ToolResultCard({ message }: ToolResultCardProps) {
  const result = message.toolResult as { ok?: boolean; data?: unknown; error?: string } | null;
  if (!result) return null;

  return (
    <div className="my-1 rounded border border-border bg-card p-3 text-xs">
      <div className="flex items-center gap-2 mb-2">
        <Badge variant={result.ok ? 'default' : 'destructive'}>
          {result.ok ? 'OK' : 'Error'}
        </Badge>
        <span className="font-mono text-muted-foreground">{message.toolName as string}</span>
      </div>
      {result.ok && !!result.data && (
        <pre className="overflow-x-auto whitespace-pre-wrap break-all text-foreground/80">
          {JSON.stringify(result.data, null, 2)}
        </pre>
      )}
      {!result.ok && result.error && (
        <p className="text-destructive">{result.error}</p>
      )}
    </div>
  );
}

// Preloads tool catalogue — exported for use in ChatPanel.
export function useToolCatalogue() {
  return useQuery({
    queryKey: ['chat-tools'],
    queryFn: () => chatService.getTools(),
    staleTime: 60_000,
  });
}
