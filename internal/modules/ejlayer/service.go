package ejlayer

import (
	"github.com/jackc/pgx/v5/pgxpool"
)

// Service is the top-level EJLayer module
type Service struct {
	Evaluator *Evaluator
	Recorder  *Recorder
}

// NewService creates a new EJLayer service
func NewService(pool *pgxpool.Pool) *Service {
	return &Service{
		Evaluator: NewEvaluator(pool),
		Recorder:  NewRecorder(pool),
	}
}
