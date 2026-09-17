CREATE TABLE IF NOT EXISTS exploratory_paper_entry_queue (
    request_id TEXT PRIMARY KEY,
    candidate_id TEXT NOT NULL UNIQUE,
    payload JSONB NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('PENDING', 'PROCESSED', 'FAILED_CLOSED')),
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_exploratory_paper_entry_queue_pending
    ON exploratory_paper_entry_queue(status, created_at);

COMMENT ON TABLE exploratory_paper_entry_queue IS
    'Idempotent PAPER-only handoff from an already human-approved canonical workflow to the exploratory runtime loop.';
