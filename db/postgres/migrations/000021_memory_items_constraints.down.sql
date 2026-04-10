DROP TRIGGER IF EXISTS trg_memory_items_updated_at ON memory_items;
DROP FUNCTION IF EXISTS set_memory_items_updated_at();

DROP INDEX IF EXISTS idx_memory_items_bank_type_ts;
DROP INDEX IF EXISTS uq_memory_items_bank_source_ref;

ALTER TABLE memory_items
    DROP CONSTRAINT IF EXISTS chk_memory_items_data_object;

ALTER TABLE memory_items
    DROP CONSTRAINT IF EXISTS chk_memory_items_tags_array;

ALTER TABLE memory_items
    DROP CONSTRAINT IF EXISTS chk_memory_items_summary_not_blank;

ALTER TABLE memory_items
    DROP CONSTRAINT IF EXISTS chk_memory_items_bank;
