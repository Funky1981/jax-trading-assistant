DROP TRIGGER IF EXISTS trg_paper_ledger_events_append_only ON paper_ledger_events;
DROP FUNCTION IF EXISTS reject_paper_ledger_event_mutation();
DROP TABLE IF EXISTS paper_ledger_events;
DROP TABLE IF EXISTS paper_accounts;
