DROP TRIGGER IF EXISTS trg_paper_fills_append_only ON paper_fills;
DROP FUNCTION IF EXISTS reject_paper_fill_mutation();
DROP TABLE IF EXISTS paper_fills;
