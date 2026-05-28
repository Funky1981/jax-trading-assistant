export type AnalyticsEventName =
  | 'page_viewed'
  | 'ai_scanner_enabled'
  | 'sentiment_settings_opened'
  | 'opportunity_sentiment_viewed'
  | 'sentiment_alert_opened'
  | 'approval_sentiment_evidence_viewed'
  | 'backtest_sentiment_enabled'
  | 'teach_me_sentiment_opened';

export interface AnalyticsEventPayload {
  source_surface?: 'home' | 'ai_trading' | 'notifications' | 'approvals' | 'research' | string;
  opportunity_id?: string;
  candidate_id?: string;
  route_type?: string;
  sentiment_mode?: string;
  enabled?: boolean;
  destination_path?: string;
  [key: string]: unknown;
}

interface AnalyticsTransport {
  track: (eventName: AnalyticsEventName, payload: AnalyticsEventPayload) => void;
}

declare global {
  interface Window {
    __JAX_ANALYTICS__?: AnalyticsTransport;
    __JAX_ANALYTICS_DEBUG__?: boolean;
  }
}

export function emitAnalyticsEvent(eventName: AnalyticsEventName, payload: AnalyticsEventPayload = {}): void {
  try {
    const transport = window.__JAX_ANALYTICS__;
    if (transport && typeof transport.track === 'function') {
      transport.track(eventName, payload);
      return;
    }

    if (window.__JAX_ANALYTICS_DEBUG__) {
      // Debug fallback keeps workflows safe when no transport is configured.
      console.debug('[analytics]', eventName, payload);
    }
  } catch {
    // Never let analytics failures break user workflows.
  }
}
