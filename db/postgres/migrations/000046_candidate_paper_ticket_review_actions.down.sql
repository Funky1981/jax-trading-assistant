DROP FUNCTION IF EXISTS append_review_note(TEXT, TEXT);

ALTER TABLE candidate_paper_tickets
	DROP COLUMN IF EXISTS review_notes;
