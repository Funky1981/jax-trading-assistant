ALTER TABLE candidate_economic_inputs
    ADD COLUMN IF NOT EXISTS slippage_allowance NUMERIC CHECK (slippage_allowance >= 0);
