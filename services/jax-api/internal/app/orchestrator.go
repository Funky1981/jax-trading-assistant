package app

import (
	"context"

	"jax-trading-assistant/services/jax-api/internal/domain"
)

type Storage interface {
	SaveTrade(ctx context.Context, setup domain.TradeSetup, risk domain.RiskResult, event *domain.Event) error
}

type Dexter interface {
	ResearchCompany(ctx context.Context, ticker string, questions []string) (*domain.ResearchBundle, error)
}

type Orchestrator struct {
	detector  *EventDetector
	generator *TradeGenerator
	risk      *RiskEngine
	storage   Storage
	dexter    Dexter
}

func NewOrchestrator(detector *EventDetector, generator *TradeGenerator, risk *RiskEngine, storage Storage, dexter Dexter) *Orchestrator {
	return &Orchestrator{
		detector:  detector,
		generator: generator,
		risk:      risk,
		storage:   storage,
		dexter:    dexter,
	}
}

type ProcessResult struct {
	Events []domain.Event `json:"events"`
	Trades []struct {
		Setup domain.TradeSetup `json:"setup"`
		Risk  domain.RiskResult `json:"risk"`
	} `json:"trades"`
}

func (o *Orchestrator) ProcessSymbol(ctx context.Context, symbol string, accountSize float64, riskPercent float64, gapThresholdPct float64) (ProcessResult, error) {
	events, err := o.detector.DetectGaps(ctx, symbol, gapThresholdPct)
	if err != nil {
		return ProcessResult{}, err
	}

	var result ProcessResult
	result.Events = events

	for _, e := range events {
		setups, err := o.generator.GenerateFromEvent(ctx, e)
		if err != nil {
			return ProcessResult{}, err
		}

		for _, s := range setups {
			var target *float64
			if len(s.Targets) > 0 {
				target = &s.Targets[0]
			}

			r, err := o.risk.Calculate(ctx, accountSize, riskPercent, s.Entry, s.Stop, target)
			if err != nil {
				return ProcessResult{}, err
			}

			if o.dexter != nil {
				bundle, err := o.dexter.ResearchCompany(ctx, s.Symbol, []string{
					"Summarise the last 4 quarters of earnings.",
					"Highlight key risks and catalysts for this trade idea.",
				})
				if err == nil && bundle != nil {
					s.Research = bundle
				}
			}

			if o.storage != nil {
				if err := o.storage.SaveTrade(ctx, s, r, &e); err != nil {
					return ProcessResult{}, err
				}
			}

			result.Trades = append(result.Trades, struct {
				Setup domain.TradeSetup `json:"setup"`
				Risk  domain.RiskResult `json:"risk"`
			}{Setup: s, Risk: r})
		}
	}

	return result, nil
}
