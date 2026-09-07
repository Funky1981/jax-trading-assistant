DROP TRIGGER IF EXISTS trg_workflow_breaker_events_append_only ON workflow_breaker_events;
DROP FUNCTION IF EXISTS reject_workflow_breaker_event_mutation();
DROP TABLE IF EXISTS workflow_breaker_events;
DROP TABLE IF EXISTS workflow_breakers;
