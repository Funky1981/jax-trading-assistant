-- Restore the historical registry state if this forward migration is rolled
-- back. The source row itself is never deleted because historical events may
-- still reference it.
UPDATE event_sources
SET enabled = TRUE,
    priority = 20,
    metadata = (COALESCE(metadata, '{}'::jsonb) - 'retired' - 'retired_reason') || jsonb_build_object('api', 'finnhub'),
    updated_at = NOW()
WHERE id = 'finnhub';
