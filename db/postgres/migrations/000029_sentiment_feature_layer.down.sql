DROP INDEX IF EXISTS idx_sentiment_paper_live_handoffs_backtest;
DROP TABLE IF EXISTS sentiment_paper_live_handoffs;

DROP INDEX IF EXISTS idx_approval_override_reasons_code;
DROP INDEX IF EXISTS idx_approval_override_reasons_candidate;
DROP TABLE IF EXISTS approval_override_reasons;

DROP TABLE IF EXISTS notification_preferences;

DROP INDEX IF EXISTS idx_notification_events_kind;
DROP INDEX IF EXISTS idx_notification_events_created;
DROP TABLE IF EXISTS notification_events;

DROP TABLE IF EXISTS sentiment_alert_rules;

DROP INDEX IF EXISTS idx_opportunity_sentiment_snapshots_opportunity;
DROP TABLE IF EXISTS opportunity_sentiment_snapshots;

DROP INDEX IF EXISTS idx_sentiment_aggregates_state;
DROP INDEX IF EXISTS idx_sentiment_aggregates_symbol_window;
DROP TABLE IF EXISTS sentiment_aggregates;

DROP INDEX IF EXISTS idx_sentiment_events_news_item;
DROP INDEX IF EXISTS idx_sentiment_events_symbol_scored;
DROP TABLE IF EXISTS sentiment_events;

DROP INDEX IF EXISTS idx_news_items_source_family;
DROP INDEX IF EXISTS idx_news_items_symbol_published;
DROP TABLE IF EXISTS news_items;
