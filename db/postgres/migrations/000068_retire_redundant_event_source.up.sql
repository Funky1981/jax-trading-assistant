-- CR-02D: retain the historical source row for replay/audit references, but
-- prevent the retired Finnhub fallback from remaining active in the registry.
UPDATE event_sources
SET enabled = FALSE,
    priority = 9999,
    metadata = COALESCE(metadata, '{}'::jsonb) || jsonb_build_object(
        'retired', TRUE,
        'retired_reason', 'CR-02D redundant provider consolidation'
    ),
    updated_at = NOW()
WHERE id = 'finnhub';
