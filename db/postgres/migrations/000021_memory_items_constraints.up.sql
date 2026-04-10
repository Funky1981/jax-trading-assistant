ALTER TABLE memory_items
    ADD CONSTRAINT chk_memory_items_bank
        CHECK (bank IN ('research', 'trades', 'signals', 'reflections'));

ALTER TABLE memory_items
    ADD CONSTRAINT chk_memory_items_summary_not_blank
        CHECK (length(btrim(summary)) > 0);

ALTER TABLE memory_items
    ADD CONSTRAINT chk_memory_items_tags_array
        CHECK (jsonb_typeof(tags) = 'array');

ALTER TABLE memory_items
    ADD CONSTRAINT chk_memory_items_data_object
        CHECK (data IS NULL OR jsonb_typeof(data) = 'object');

CREATE UNIQUE INDEX IF NOT EXISTS uq_memory_items_bank_source_ref
    ON memory_items (bank, source_system, source_ref)
    WHERE source_system IS NOT NULL AND source_ref IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_memory_items_bank_type_ts
    ON memory_items (bank, type, ts DESC);

CREATE OR REPLACE FUNCTION set_memory_items_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_memory_items_updated_at ON memory_items;

CREATE TRIGGER trg_memory_items_updated_at
    BEFORE UPDATE ON memory_items
    FOR EACH ROW
    EXECUTE FUNCTION set_memory_items_updated_at();
