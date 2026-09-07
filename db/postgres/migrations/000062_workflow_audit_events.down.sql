DROP TRIGGER IF EXISTS trg_workflow_audit_append_only ON workflow_audit_events;
DROP FUNCTION IF EXISTS reject_workflow_audit_mutation();
DROP TABLE IF EXISTS workflow_audit_events;
