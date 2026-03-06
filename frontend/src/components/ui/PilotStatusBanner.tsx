import { AlertTriangle, ShieldAlert } from 'lucide-react';
import { cn } from '@/lib/utils';

interface PilotStatusBannerProps {
  title: string;
  readOnly?: boolean;
  reasons?: string[];
  checklist?: string[];
  className?: string;
  compact?: boolean;
}

export function PilotStatusBanner({
  title,
  readOnly = false,
  reasons = [],
  checklist = [],
  className,
  compact = false,
}: PilotStatusBannerProps) {
  const Icon = readOnly ? ShieldAlert : AlertTriangle;
  return (
    <div
      className={cn(
        'rounded-md border px-3 py-2 text-sm',
        readOnly
          ? 'border-destructive/40 bg-destructive/10 text-destructive'
          : 'border-warning/40 bg-warning/10 text-warning',
        className
      )}
    >
      <div className="flex items-start gap-2">
        <Icon className="mt-0.5 h-4 w-4 shrink-0" />
        <div className="space-y-2">
          <p className="font-semibold">{title}</p>
          {reasons.length > 0 ? (
            <div className="space-y-1">
              {reasons.map((reason) => (
                <p key={reason} className="text-xs leading-relaxed">
                  {reason}
                </p>
              ))}
            </div>
          ) : null}
          {!compact && checklist.length > 0 ? (
            <div className="space-y-1 border-t border-current/15 pt-2">
              {checklist.map((item) => (
                <p key={item} className="text-xs leading-relaxed text-foreground/90">
                  {item}
                </p>
              ))}
            </div>
          ) : null}
        </div>
      </div>
    </div>
  );
}
