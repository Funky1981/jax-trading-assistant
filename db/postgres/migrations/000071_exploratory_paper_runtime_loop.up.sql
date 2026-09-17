ALTER TABLE exploratory_paper_reviews
    DROP CONSTRAINT IF EXISTS exploratory_paper_reviews_status_check;

ALTER TABLE exploratory_paper_reviews
    ADD CONSTRAINT exploratory_paper_reviews_status_check
    CHECK (status IN ('PENDING', 'COMPLETED', 'MISSING_DATA', 'FAILED_CLOSED', 'EXIT_RECOMMENDED'));

COMMENT ON COLUMN exploratory_paper_reviews.status IS
    'Runtime review state. EXIT_RECOMMENDED is durable pending human exit approval; it is not an execution action.';
