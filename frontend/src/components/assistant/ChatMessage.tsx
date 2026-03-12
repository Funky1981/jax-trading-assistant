import { type ChatMessage } from '@/data/chat-service';
import { cn } from '@/lib/utils';

interface ChatMessageProps {
  message: ChatMessage;
}

export function ChatMessageBubble({ message }: ChatMessageProps) {
  const isUser = message.role === 'user';
  const isTool = message.role === 'tool';

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
    <div className={cn('flex', isUser ? 'justify-end' : 'justify-start')}>
      <div
        className={cn(
          'max-w-[80%] rounded-lg px-3 py-2 text-sm',
          isUser
            ? 'bg-primary text-primary-foreground'
            : 'bg-muted text-foreground'
        )}
      >
        <p className="whitespace-pre-wrap break-words">{message.content}</p>
        <p className={cn('mt-1 text-xs', isUser ? 'text-primary-foreground/60' : 'text-muted-foreground')}>
          {new Date(message.createdAt).toLocaleTimeString()}
        </p>
      </div>
    </div>
  );
}
