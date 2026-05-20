DROP INDEX IF EXISTS idx_event_priced_in_scores_hard_reject;

ALTER TABLE event_priced_in_scores
    DROP COLUMN IF EXISTS hard_reject_reasons,
    DROP COLUMN IF EXISTS hard_reject,
    DROP COLUMN IF EXISTS post_event_5m_return;
