-- Mobile approval flow: notification queue and one-time mobile decision tokens.
-- The mobile channel only references existing candidate_trades rows; it never
-- accepts symbol, order shape, or live execution instructions from a callback.

CREATE TABLE IF NOT EXISTS notification_outbox (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    channel         TEXT NOT NULL CHECK (channel IN ('telegram')),
    recipient       TEXT,
    candidate_id    UUID NOT NULL REFERENCES candidate_trades(id) ON DELETE CASCADE,
    message         TEXT NOT NULL,
    payload         JSONB NOT NULL DEFAULT '{}'::jsonb,
    status          TEXT NOT NULL DEFAULT 'pending'
                        CHECK (status IN ('pending', 'sent', 'failed', 'cancelled')),
    send_after      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    sent_at         TIMESTAMPTZ,
    error           TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_notification_outbox_pending
    ON notification_outbox (send_after, created_at)
    WHERE status = 'pending';
CREATE INDEX IF NOT EXISTS idx_notification_outbox_candidate
    ON notification_outbox (candidate_id, created_at DESC);

CREATE TABLE IF NOT EXISTS mobile_approval_tokens (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    notification_id UUID REFERENCES notification_outbox(id) ON DELETE SET NULL,
    candidate_id    UUID NOT NULL REFERENCES candidate_trades(id) ON DELETE CASCADE,
    channel         TEXT NOT NULL CHECK (channel IN ('telegram')),
    token_hash      TEXT NOT NULL UNIQUE,
    guardrail_hash  TEXT NOT NULL,
    expires_at      TIMESTAMPTZ NOT NULL,
    used_at         TIMESTAMPTZ,
    decision        TEXT CHECK (decision IN ('approved', 'rejected', 'snoozed', 'reanalysis_requested')),
    used_by         TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_mobile_approval_tokens_expiry CHECK (expires_at > created_at)
);

CREATE INDEX IF NOT EXISTS idx_mobile_approval_tokens_token_hash
    ON mobile_approval_tokens (token_hash);
CREATE INDEX IF NOT EXISTS idx_mobile_approval_tokens_candidate
    ON mobile_approval_tokens (candidate_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_mobile_approval_tokens_expires
    ON mobile_approval_tokens (expires_at)
    WHERE used_at IS NULL;
