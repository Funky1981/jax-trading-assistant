UPDATE exploratory_paper_reviews
SET status='EXIT_RECOMMENDED'
WHERE status IN ('EXIT_APPROVED','EXIT_REJECTED');

ALTER TABLE exploratory_paper_reviews
    DROP CONSTRAINT IF EXISTS exploratory_paper_reviews_status_check;

ALTER TABLE exploratory_paper_reviews
    ADD CONSTRAINT exploratory_paper_reviews_status_check
    CHECK (status IN ('PENDING', 'COMPLETED', 'MISSING_DATA', 'FAILED_CLOSED', 'EXIT_RECOMMENDED'));
