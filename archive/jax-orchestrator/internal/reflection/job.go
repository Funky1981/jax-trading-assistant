package reflection

import (
	"context"
	"fmt"
	"time"

	"jax-trading-assistant/libs/contracts"
	"jax-trading-assistant/libs/observability"
)

type MemoryClient interface {
	Recall(ctx context.Context, bank string, query contracts.MemoryQuery) ([]contracts.MemoryItem, error)
	Retain(ctx context.Context, bank string, item contracts.MemoryItem) (contracts.MemoryID, error)
}

type Job struct {
	memory   MemoryClient
	now      func() time.Time
	maxItems int
}

type JobOption func(*Job)

func WithNow(now func() time.Time) JobOption {
	return func(j *Job) {
		j.now = now
	}
}

func WithMaxItems(maxItems int) JobOption {
	return func(j *Job) {
		j.maxItems = maxItems
	}
}

func NewJob(memory MemoryClient, opts ...JobOption) *Job {
	job := &Job{
		memory:   memory,
		now:      time.Now,
		maxItems: 200,
	}
	for _, opt := range opts {
		opt(job)
	}
	return job
}

func (j *Job) Run(ctx context.Context, cfg RunConfig) (RunResult, error) {
	start := time.Now()
	beliefCount := 0
	defer func() {
		observability.RecordReflectionDuration(ctx, time.Since(start), beliefCount)
	}()
	if j.memory == nil {
		return RunResult{}, fmt.Errorf("reflection job: memory client required")
	}

	windowDays := cfg.WindowDays
	if windowDays <= 0 {
		windowDays = 7
	}

	maxItems := cfg.MaxItems
	if maxItems <= 0 {
		maxItems = j.maxItems
	}

	to := cfg.To
	if to.IsZero() {
		to = j.now()
	}
	to = to.UTC()
	from := to.AddDate(0, 0, -windowDays)

	decisions, err := j.memory.Recall(ctx, DecisionsBank, contracts.MemoryQuery{
		Types: []string{"decision"},
		From:  &from,
		To:    &to,
		Limit: maxItems,
	})
	if err != nil {
		return RunResult{}, err
	}

	outcomes, err := j.memory.Recall(ctx, OutcomesBank, contracts.MemoryQuery{
		Types: []string{"outcome"},
		From:  &from,
		To:    &to,
		Limit: maxItems,
	})
	if err != nil {
		return RunResult{}, err
	}

	window := Window{From: from, To: to}
	beliefs := GenerateBeliefs(to, window, decisions, outcomes)
	beliefCount = len(beliefs)

	retained := 0
	if !cfg.DryRun {
		for _, belief := range beliefs {
			if err := contracts.ValidateMemoryItem(belief); err != nil {
				return RunResult{}, err
			}
			if _, err := j.memory.Retain(ctx, BeliefsBank, belief); err != nil {
				return RunResult{}, err
			}
			retained++
		}
	}

	result := RunResult{
		Beliefs:     len(beliefs),
		Retained:    retained,
		Window:      window,
		BeliefItems: beliefs,
	}
	return result, nil
}
