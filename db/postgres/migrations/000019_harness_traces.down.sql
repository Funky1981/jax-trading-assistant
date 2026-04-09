DROP INDEX IF EXISTS idx_harness_traces_session_created;

DROP TABLE IF EXISTS harness_traces;

ALTER TABLE chat_messages
DROP COLUMN IF EXISTS trace_id;
