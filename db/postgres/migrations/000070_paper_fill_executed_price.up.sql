ALTER TABLE paper_fills
    ADD COLUMN IF NOT EXISTS executed_price NUMERIC;
