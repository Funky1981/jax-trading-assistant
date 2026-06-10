CREATE TABLE IF NOT EXISTS macro_scenario_results (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    macro_event_id UUID NOT NULL REFERENCES macro_events(id) ON DELETE CASCADE,
    scenario_key TEXT NOT NULL,
    candidate_bias TEXT NOT NULL,
    primary_symbols TEXT[] NOT NULL,
    secondary_symbols TEXT[] NOT NULL,
    required_confirmations TEXT[] NOT NULL,
    expected_reactions JSONB NOT NULL DEFAULT '{}'::jsonb,
    result TEXT NOT NULL,
    reason TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(macro_event_id, scenario_key),
    CONSTRAINT chk_macro_scenario_result CHECK (
        result IN (
            'eligible_for_reaction_check',
            'blocked_unknown_event',
            'blocked_no_etf_mapping',
            'blocked_conflicting_scenario',
            'blocked_disallowed_instrument'
        )
    )
);

CREATE INDEX IF NOT EXISTS idx_macro_scenario_results_event
    ON macro_scenario_results(macro_event_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_macro_scenario_results_result
    ON macro_scenario_results(result, created_at DESC);
