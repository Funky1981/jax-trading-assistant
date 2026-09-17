UPDATE exploratory_paper_reviews SET status='PENDING' WHERE status='EXIT_RECOMMENDED';

ALTER TABLE exploratory_paper_reviews
    DROP CONSTRAINT IF EXISTS exploratory_paper_reviews_status_check;

ALTER TABLE exploratory_paper_reviews
    ADD CONSTRAINT exploratory_paper_reviews_status_check
    CHECK (status IN ('PENDING', 'COMPLETED', 'MISSING_DATA', 'FAILED_CLOSED'));
