import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import { type ChatMessage } from '@/data/chat-service';
import { cn } from '@/lib/utils';

interface ChatMessageProps {
  message: ChatMessage;
}

/** Renders an assistant reply with GitHub-Flavored Markdown support. */
function AssistantContent({ content }: { content: string }) {
  return (
    <ReactMarkdown
      remarkPlugins={[remarkGfm]}
      components={{
        p: ({ children }) => <p className="mb-1 last:mb-0 whitespace-pre-wrap break-words">{children}</p>,
        h1: ({ children }) => <h1 className="text-base font-bold mt-2 mb-1">{children}</h1>,
        h2: ({ children }) => <h2 className="text-sm font-bold mt-2 mb-1">{children}</h2>,
        h3: ({ children }) => <h3 className="text-sm font-semibold mt-1.5 mb-0.5">{children}</h3>,
        ul: ({ children }) => <ul className="list-disc list-inside mb-1 space-y-0.5">{children}</ul>,
        ol: ({ children }) => <ol className="list-decimal list-inside mb-1 space-y-0.5">{children}</ol>,
        li: ({ children }) => <li className="text-sm">{children}</li>,
        code: ({ className, children, ...props }) => {
          const isBlock = className?.startsWith('language-');
          return isBlock ? (
            <code className="block bg-background/60 rounded px-2 py-1 text-xs font-mono overflow-x-auto my-1 whitespace-pre" {...props}>
              {children}
            </code>
          ) : (
            <code className="bg-background/60 rounded px-1 py-0.5 text-xs font-mono" {...props}>
              {children}
            </code>
          );
        },
        pre: ({ children }) => <pre className="my-1 overflow-x-auto">{children}</pre>,
        strong: ({ children }) => <strong className="font-semibold">{children}</strong>,
        em: ({ children }) => <em className="italic">{children}</em>,
        blockquote: ({ children }) => (
          <blockquote className="border-l-2 border-muted-foreground/40 pl-2 text-muted-foreground italic my-1">{children}</blockquote>
        ),
        a: ({ href, children }) => (
          <a href={href} className="underline text-primary hover:opacity-80" target="_blank" rel="noopener noreferrer">{children}</a>
        ),
        table: ({ children }) => <table className="text-xs border-collapse my-1 w-full">{children}</table>,
        th: ({ children }) => <th className="border border-border px-2 py-0.5 font-semibold bg-muted/50 text-left">{children}</th>,
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
        {isUser ? (
          <p className="whitespace-pre-wrap break-words">{message.content}</p>
        ) : (
          <AssistantContent content={message.content} />
        )}
        <p className={cn('mt-1 text-xs', isUser ? 'text-primary-foreground/60' : 'text-muted-foreground')}>
          {new Date(message.createdAt).toLocaleTimeString()}
        </p>
      </div>
    </div>
  );
}
