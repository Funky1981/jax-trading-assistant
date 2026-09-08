DROP TRIGGER IF EXISTS trg_paper_soak_events_append_only ON paper_soak_events;
DROP FUNCTION IF EXISTS reject_paper_soak_event_mutation();
DROP TABLE IF EXISTS paper_soak_events;
DROP TABLE IF EXISTS paper_soak_runs;
