CREATE TABLE IF NOT EXISTS ai_shadow_benchmark_runs (
    id UUID PRIMARY KEY,
    manifest_version TEXT NOT NULL,
    manifest_fingerprint TEXT NOT NULL,
    provider TEXT NOT NULL,
    model TEXT NOT NULL,
    prompt_version TEXT NOT NULL,
    schema_version TEXT NOT NULL,
    seed BIGINT NOT NULL,
    temperature DOUBLE PRECISION NOT NULL,
    event_limit INTEGER NOT NULL CHECK (event_limit BETWEEN 1 AND 60),
    started_at TIMESTAMPTZ NOT NULL,
    completed_at TIMESTAMPTZ,
    status TEXT NOT NULL CHECK (status IN ('running', 'completed', 'invalid')),
    failure_reason TEXT,
    safety_before JSONB NOT NULL,
    safety_after JSONB,
    report_paths JSONB
);

CREATE TABLE IF NOT EXISTS ai_shadow_benchmark_attempts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    run_id UUID NOT NULL REFERENCES ai_shadow_benchmark_runs(id) ON DELETE RESTRICT,
    event_id UUID NOT NULL REFERENCES world_monitor_research_inbox(id) ON DELETE RESTRICT,
    attempt_number INTEGER NOT NULL CHECK (attempt_number IN (1, 2)),
    input_fingerprint TEXT NOT NULL,
    provider TEXT NOT NULL,
    model TEXT NOT NULL,
    model_reported_identifier TEXT,
    prompt_version TEXT NOT NULL,
    schema_version TEXT NOT NULL,
    seed BIGINT NOT NULL,
    temperature DOUBLE PRECISION NOT NULL,
    request_timestamp TIMESTAMPTZ NOT NULL,
    response_timestamp TIMESTAMPTZ NOT NULL,
    duration_ms BIGINT NOT NULL CHECK (duration_ms >= 0),
    raw_response_hash TEXT NOT NULL,
    validation_status TEXT NOT NULL CHECK (validation_status IN ('accepted', 'rejected')),
    validation_errors TEXT[] NOT NULL DEFAULT '{}',
    failure_reason TEXT,
    UNIQUE (run_id, event_id, attempt_number)
);

CREATE TABLE IF NOT EXISTS ai_shadow_benchmark_results (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    run_id UUID NOT NULL REFERENCES ai_shadow_benchmark_runs(id) ON DELETE RESTRICT,
    manifest_version TEXT NOT NULL,
    event_id UUID NOT NULL REFERENCES world_monitor_research_inbox(id) ON DELETE RESTRICT,
    input_fingerprint TEXT NOT NULL,
    provider TEXT NOT NULL,
    model TEXT NOT NULL,
    model_reported_identifier TEXT,
    prompt_version TEXT NOT NULL,
    schema_version TEXT NOT NULL,
    seed BIGINT NOT NULL,
    temperature DOUBLE PRECISION NOT NULL,
    request_timestamp TIMESTAMPTZ NOT NULL,
    response_timestamp TIMESTAMPTZ NOT NULL,
    duration_ms BIGINT NOT NULL CHECK (duration_ms >= 0),
    retry_count INTEGER NOT NULL CHECK (retry_count BETWEEN 0 AND 1),
    raw_response_hash TEXT NOT NULL,
    parsed_result JSONB,
    validation_status TEXT NOT NULL CHECK (validation_status IN ('accepted', 'rejected')),
    validation_errors TEXT[] NOT NULL DEFAULT '{}',
    failure_reason TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (run_id, event_id)
);

CREATE INDEX IF NOT EXISTS idx_ai_shadow_results_event
    ON ai_shadow_benchmark_results(event_id, created_at);

CREATE OR REPLACE FUNCTION protect_ai_shadow_append_only()
RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'AI shadow attempt and result provenance is append-only';
END;
$$;

DROP TRIGGER IF EXISTS trg_protect_ai_shadow_attempts ON ai_shadow_benchmark_attempts;
CREATE TRIGGER trg_protect_ai_shadow_attempts
    BEFORE UPDATE OR DELETE ON ai_shadow_benchmark_attempts
    FOR EACH ROW EXECUTE FUNCTION protect_ai_shadow_append_only();

DROP TRIGGER IF EXISTS trg_protect_ai_shadow_results ON ai_shadow_benchmark_results;
CREATE TRIGGER trg_protect_ai_shadow_results
    BEFORE UPDATE OR DELETE ON ai_shadow_benchmark_results
    FOR EACH ROW EXECUTE FUNCTION protect_ai_shadow_append_only();
