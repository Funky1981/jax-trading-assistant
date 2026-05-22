import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import {
  Clock,
  TrendingUp,
  TrendingDown,
  CheckCircle,
  XCircle,
  Pause,
  BookMarked,
  AlertCircle,
} from 'lucide-react';
import { type UXMode } from '@/utils/beginner-helpers';
import type { TimelineEvent } from './ResearchTimeline.types';

interface TimelineEventCardProps {
  event: TimelineEvent;
  isLast: boolean;
}

function TimelineEventCard({ event, isLast }: TimelineEventCardProps) {
  const icons = {
    news: <AlertCircle className="w-5 h-5 text-blue-600" />,
    analysis: <BookMarked className="w-5 h-5 text-purple-600" />,
    approval: <CheckCircle className="w-5 h-5 text-green-600" />,
    rejected: <XCircle className="w-5 h-5 text-red-600" />,
    execution: <TrendingUp className="w-5 h-5 text-emerald-600" />,
    exit: <TrendingDown className="w-5 h-5 text-orange-600" />,
    reflection: <BookMarked className="w-5 h-5 text-slate-600" />,
    snooze: <Pause className="w-5 h-5 text-yellow-600" />,
  };

  const colors = {
    info: 'bg-blue-50 border-blue-200',
    success: 'bg-green-50 border-green-200',
    warning: 'bg-yellow-50 border-yellow-200',
    error: 'bg-red-50 border-red-200',
  };

  const textColors = {
    info: 'text-blue-900',
    success: 'text-green-900',
    warning: 'text-yellow-900',
    error: 'text-red-900',
  };

  const severity = event.severity || 'info';

  return (
    <div className="flex gap-4">
      {/* Timeline dot and line */}
      <div className="flex flex-col items-center">
        <div className="w-10 h-10 bg-white border-2 border-current rounded-full flex items-center justify-center">
          {icons[event.type]}
        </div>
        {!isLast && <div className="w-0.5 h-12 bg-gray-200 mt-2" />}
      </div>

      {/* Event card */}
      <div className="flex-1 pb-4">
        <Card className={colors[severity]}>
          <CardHeader className="pb-3">
            <div className="flex items-start justify-between gap-2">
              <div className="flex-1">
                <CardTitle className={`text-base ${textColors[severity]}`}>{event.title}</CardTitle>
                <p className="text-xs text-muted-foreground mt-1">{event.timestamp}</p>
              </div>
              {event.signal && (
                <Badge
                  variant={event.signal === 'BUY' ? 'default' : 'destructive'}
                  className="font-mono"
                >
                  {event.signal}
                </Badge>
              )}
            </div>
          </CardHeader>
          <CardContent className="space-y-2">
            <p className={`text-sm ${textColors[severity]}`}>{event.description}</p>

            {event.etf && (
              <p className="text-xs font-semibold text-muted-foreground">
                ETF: <span className="font-mono">{event.etf}</span>
              </p>
            )}

            {event.details && Object.keys(event.details).length > 0 && (
              <div className="grid grid-cols-2 gap-2 text-xs pt-2 border-t">
                {Object.entries(event.details).map(([key, value]) => (
                  <div key={key}>
                    <p className="font-semibold text-muted-foreground capitalize">{key}:</p>
                    <p className="font-medium">{value}</p>
                  </div>
                ))}
              </div>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  );
}

export interface ResearchTimelineProps {
  events: TimelineEvent[];
  mode: UXMode;
  title?: string;
  description?: string;
}

/**
 * Research Timeline Screen - Shows narrative flow of research, analysis, approval, execution, and reflection
 * Part of Step 9: Beginner UX
 *
 * Example timeline:
 * 1. News breaks (ETF impact detected)
 * 2. Jax analyzes (priced-in check, confounder search)
 * 3. Approval sent to mobile
 * 4. User approves (or rejects)
 * 5. Paper order submitted
 * 6. Order fills
 * 7. Position exits
 * 8. Post-trade reflection (what happened, what was learned)
 */
export function ResearchTimeline({
  events,
  mode,
  title = 'Trade Timeline',
  description,
}: ResearchTimelineProps) {
  return (
    <div className="p-6 max-w-3xl mx-auto">
      <div className="mb-8">
        <h1 className="text-3xl font-bold mb-2">{title}</h1>
        {description && <p className="text-muted-foreground">{description}</p>}
      </div>

      {mode === 'simple' && (
        <Card className="mb-6 bg-blue-50 border-blue-200">
          <CardHeader>
            <CardTitle className="text-blue-900">How to Read This Timeline</CardTitle>
          </CardHeader>
          <CardContent className="space-y-2 text-sm text-blue-800">
            <p>
              <strong>📰 News:</strong> A market event that affects the ETF.
            </p>
            <p>
              <strong>🔍 Analysis:</strong> Jax researches whether the market has already reacted.
            </p>
            <p>
              <strong>✓ Approval:</strong> You approved the trade.
            </p>
            <p>
              <strong>📈 Execution:</strong> Paper order submitted and filled.
            </p>
            <p>
              <strong>🚪 Exit:</strong> Position closed and profit/loss locked in.
            </p>
            <p>
              <strong>💭 Reflection:</strong> What was learned for next time.
            </p>
          </CardContent>
        </Card>
      )}

      <div className="space-y-2">
        {events.map((event, idx) => (
          <TimelineEventCard
            key={idx}
            event={event}
            isLast={idx === events.length - 1}
          />
        ))}
      </div>

      {events.length === 0 && (
        <Card className="bg-gray-50 border-gray-200">
          <CardContent className="py-12 text-center">
            <Clock className="w-12 h-12 text-gray-400 mx-auto mb-4" />
            <p className="text-gray-600">No timeline events yet.</p>
            <p className="text-sm text-gray-500 mt-1">
              {mode === 'simple'
                ? "Once you approve a trade, you'll see its full journey here."
                : 'Timeline events are populated as the trade progresses through approval and execution.'}
            </p>
          </CardContent>
        </Card>
      )}
    </div>
  );
}

/**
 * Generate example timeline events for demo/testing
 */
