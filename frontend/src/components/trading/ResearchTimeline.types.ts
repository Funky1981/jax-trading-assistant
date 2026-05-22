export interface TimelineEvent {
  timestamp: string;
  type:
    | 'news'
    | 'analysis'
    | 'approval'
    | 'rejected'
    | 'execution'
    | 'exit'
    | 'reflection'
    | 'snooze';
  title: string;
  description: string;
  etf?: string;
  signal?: 'BUY' | 'SELL' | 'HOLD';
  details?: Record<string, string | number>;
  severity?: 'info' | 'success' | 'warning' | 'error';
}
