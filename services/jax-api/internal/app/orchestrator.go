package app

import (
	"context"

	"jax-trading-assistant/services/jax-api/internal/domain"
)

type Storage interface {
	SaveTrade(ctx context.Context, setup domain.TradeSetup, risk domain.RiskResult, event *domain.Event) error
	SaveAuditEvent(ctx context.Context, event domain.AuditEvent) error
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
	audit     *AuditLogger
}

func NewOrchestrator(detector *EventDetector, generator *TradeGenerator, risk *RiskEngine, storage Storage, dexter Dexter, audit *AuditLogger) *Orchestrator {
	return &Orchestrator{
		detector:  detector,
		generator: generator,
		risk:      risk,
		storage:   storage,
		dexter:    dexter,
		audit:     audit,
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
	ctx = EnsureCorrelationID(ctx)
	if o.audit != nil {
		_ = o.audit.LogDecision(ctx, "process_symbol_start", domain.AuditOutcomeStarted, map[string]any{
			"symbol":       symbol,
			"accountSet":   accountSize != 0,
			"riskSet":      riskPercent != 0,
			"thresholdPct": gapThresholdPct,
		}, nil)
	}
	events, err := o.detector.DetectGaps(ctx, symbol, gapThresholdPct)
	if err != nil {
		if o.audit != nil {
			_ = o.audit.LogDecision(ctx, "process_symbol_error", domain.AuditOutcomeError, map[string]any{
				"symbol": symbol,
			}, err)
		}
		return ProcessResult{}, err
	}

	var result ProcessResult
	result.Events = events
	if o.audit != nil && len(events) == 0 {
		_ = o.audit.LogDecision(ctx, "process_symbol_no_events", domain.AuditOutcomeSkipped, map[string]any{
			"symbol": symbol,
		}, nil)
	}

	for _, e := range events {
		setups, err := o.generator.GenerateFromEvent(ctx, e)
		if err != nil {
			if o.audit != nil {
				_ = o.audit.LogDecision(ctx, "process_symbol_generate_error", domain.AuditOutcomeError, redactEventPayload(e), err)
			}
			return ProcessResult{}, err
		}
		if o.audit != nil && len(setups) == 0 {
			_ = o.audit.LogDecision(ctx, "process_symbol_no_setups", domain.AuditOutcomeSkipped, redactEventPayload(e), nil)
		}

		for _, s := range setups {
			var target *float64
			if len(s.Targets) > 0 {
				target = &s.Targets[0]
			}

			r, err := o.risk.Calculate(ctx, accountSize, riskPercent, s.Entry, s.Stop, target)
			if err != nil {
				if o.audit != nil {
					payload := redactTradeSetupPayload(s)
					payload["symbol"] = s.Symbol
					_ = o.audit.LogDecision(ctx, "process_symbol_risk_error", domain.AuditOutcomeError, payload, err)
				}
				return ProcessResult{}, err
			}

			if o.dexter != nil {
				bundle, err := o.dexter.ResearchCompany(ctx, s.Symbol, []string{
					"Summarise the last 4 quarters of earnings.",
					"Highlight key risks and catalysts for this trade idea.",
				})
				if err != nil {
					if o.audit != nil {
						_ = o.audit.LogDecision(ctx, "process_symbol_research_error", domain.AuditOutcomeError, map[string]any{
							"symbol":        s.Symbol,
							"questionCount": 2,
						}, err)
					}
				} else if bundle != nil {
					s.Research = bundle
					if o.audit != nil {
						_ = o.audit.LogDecision(ctx, "process_symbol_research_success", domain.AuditOutcomeSuccess, map[string]any{
							"symbol":        s.Symbol,
							"questionCount": 2,
						}, nil)
					}
				}
			}

			if o.storage != nil {
				if err := o.storage.SaveTrade(ctx, s, r, &e); err != nil {
					if o.audit != nil {
						payload := redactTradeSetupPayload(s)
						payload["eventId"] = e.ID
						_ = o.audit.LogDecision(ctx, "process_symbol_save_trade_error", domain.AuditOutcomeError, payload, err)
					}
					return ProcessResult{}, err
				}
				if o.audit != nil {
					payload := redactTradeSetupPayload(s)
					payload["eventId"] = e.ID
					_ = o.audit.LogDecision(ctx, "process_symbol_save_trade_success", domain.AuditOutcomeSuccess, payload, nil)
				}
			}

			result.Trades = append(result.Trades, struct {
				Setup domain.TradeSetup `json:"setup"`
				Risk  domain.RiskResult `json:"risk"`
			}{Setup: s, Risk: r})
		}
	}

	if o.audit != nil {
		_ = o.audit.LogDecision(ctx, "process_symbol_success", domain.AuditOutcomeSuccess, map[string]any{
			"symbol": symbol,
			"events": len(result.Events),
			"trades": len(result.Trades),
		}, nil)
	}

	return result, nil
}
