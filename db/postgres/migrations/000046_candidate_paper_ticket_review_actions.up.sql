ALTER TABLE candidate_paper_tickets
	ADD COLUMN IF NOT EXISTS review_notes TEXT;

CREATE OR REPLACE FUNCTION append_review_note(existing TEXT, new_note TEXT)
RETURNS TEXT
LANGUAGE SQL
IMMUTABLE
AS $$
	SELECT CASE
		WHEN NULLIF(BTRIM(COALESCE(new_note, '')), '') IS NULL THEN existing
		WHEN NULLIF(BTRIM(COALESCE(existing, '')), '') IS NULL THEN BTRIM(new_note)
		ELSE existing || E'\n' || BTRIM(new_note)
	END
$$;
