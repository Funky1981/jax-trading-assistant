$ErrorActionPreference = "Stop"

docker compose exec -T postgres psql -U jax -d jax -v ON_ERROR_STOP=1 -c "
DELETE FROM macro_candidate_trades
WHERE macro_event_id IN (
    SELECT id FROM macro_events
    WHERE source IN ('test','fixture') OR raw_payload->>'fixture' = 'true'
);
DELETE FROM macro_evidence_bundles
WHERE macro_event_id IN (
    SELECT id FROM macro_events
    WHERE source IN ('test','fixture') OR raw_payload->>'fixture' = 'true'
);
DELETE FROM macro_event_etf_map
WHERE macro_event_id IN (
    SELECT id FROM macro_events
    WHERE source IN ('test','fixture') OR raw_payload->>'fixture' = 'true'
);
DELETE FROM macro_events
WHERE source IN ('test','fixture') OR raw_payload->>'fixture' = 'true';
"
