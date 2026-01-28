-- Rollback audit events table

DROP INDEX IF EXISTS idx_audit_category_timestamp;
DROP INDEX IF EXISTS idx_audit_timestamp;
DROP INDEX IF EXISTS idx_audit_outcome;
DROP INDEX IF EXISTS idx_audit_action;
DROP INDEX IF EXISTS idx_audit_category;
DROP INDEX IF EXISTS idx_audit_correlation_id;

DROP TABLE IF EXISTS audit_events;
