-- Step 5: persist priced-in engine decision details.

ALTER TABLE event_priced_in_scores
    ADD COLUMN IF NOT EXISTS post_event_5m_return NUMERIC,
    ADD COLUMN IF NOT EXISTS hard_reject BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS hard_reject_reasons JSONB NOT NULL DEFAULT '[]'::jsonb;

CREATE INDEX IF NOT EXISTS idx_event_priced_in_scores_hard_reject
    ON event_priced_in_scores(hard_reject, verdict);
