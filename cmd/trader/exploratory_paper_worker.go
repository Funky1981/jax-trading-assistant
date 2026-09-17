package main

import (
	"context"
	"log"
	"time"

	"jax-trading-assistant/internal/modules/exploratorypaper"

	"github.com/jackc/pgx/v5/pgxpool"
)

func startExploratoryPaperReviewWorker(ctx context.Context, pool *pgxpool.Pool) {
	store := exploratorypaper.NewPostgresStore(pool)
	run := func() {
		if err := processDueExploratoryReviews(ctx, pool, store); err != nil {
			log.Printf("exploratory paper review worker: %v", err)
		}
	}
	run()
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			run()
		}
	}
}

func processDueExploratoryReviews(ctx context.Context, pool *pgxpool.Pool, store *exploratorypaper.PostgresStore) error {
	rows, err := pool.Query(ctx, `SELECT position_id,session_number FROM exploratory_paper_reviews WHERE status='PENDING' AND scheduled_at <= NOW() ORDER BY scheduled_at FOR UPDATE SKIP LOCKED`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var positionID string
		var session int
		if err := rows.Scan(&positionID, &session); err != nil {
			return err
		}
		if err := store.RecordReviewUnavailable(ctx, positionID, session, time.Now().UTC(), "required evidence or market observation was unavailable; no review result fabricated"); err != nil {
			return err
		}
	}
	return rows.Err()
}
