DROP INDEX IF EXISTS idx_mobile_approval_tokens_expires;
DROP INDEX IF EXISTS idx_mobile_approval_tokens_candidate;
DROP INDEX IF EXISTS idx_mobile_approval_tokens_token_hash;
DROP TABLE IF EXISTS mobile_approval_tokens;

DROP INDEX IF EXISTS idx_notification_outbox_candidate;
DROP INDEX IF EXISTS idx_notification_outbox_pending;
DROP TABLE IF EXISTS notification_outbox;
