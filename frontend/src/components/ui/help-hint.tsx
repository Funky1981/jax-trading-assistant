import { Info } from 'lucide-react';
import { cn } from '@/lib/utils';

type HelpHintProps = {
  text: string;
  className?: string;
};

export function HelpHint({ text, className }: HelpHintProps) {
  return (
    <span
      className={cn('inline-flex items-center text-muted-foreground', className)}
      title={text}
      aria-label={text}
      role="img"
    >
      <Info className="h-4 w-4" />
    </span>
  );
}
