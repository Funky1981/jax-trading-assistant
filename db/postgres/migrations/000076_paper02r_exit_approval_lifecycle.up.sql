ALTER TABLE exploratory_paper_reviews
    DROP CONSTRAINT IF EXISTS exploratory_paper_reviews_status_check;

ALTER TABLE exploratory_paper_reviews
    ADD CONSTRAINT exploratory_paper_reviews_status_check
    CHECK (status IN ('PENDING', 'COMPLETED', 'MISSING_DATA', 'FAILED_CLOSED', 'EXIT_RECOMMENDED', 'EXIT_APPROVED', 'EXIT_REJECTED'));

COMMENT ON COLUMN exploratory_paper_reviews.status IS
    'PAPER-02R review state. Exit recommendation is durable before approval; approval or rejection is a separate workflow-bound state.';
