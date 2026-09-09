// Package planner owns Jax's bounded, advisory planning contract.
package planner

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"

	"jax-trading-assistant/libs/contracts"
	"jax-trading-assistant/libs/strategies"
)

const (
	ContractVersion = "jax-planner/v1"
	maxSteps        = 32
	maxStepLength   = 512
	maxTextLength   = 4096
)

var (
	ErrProviderRequired = errors.New("planner: provider required")
	ErrInvalidRequest   = errors.New("planner: invalid request")
	ErrInvalidResult    = errors.New("planner: invalid result")
)

// Request is the bounded planning input. It contains context and evidence
// only; it carries no execution authority.
type Request struct {
	Symbol      string
	Context     string
	Constraints map[string]any
	Memories    []contracts.MemoryItem
	Signals     []strategies.Signal
}

// Result is an advisory plan. Action is descriptive and never grants
// approval, order, broker, or execution authority.
type Result struct {
	Summary        string
	Steps          []string
	Action         string
	Confidence     float64
	ReasoningNotes string
}

// Planner is the Jax-owned planning boundary used by orchestration.
type Planner interface {
	Plan(ctx context.Context, req Request) (Result, error)
}

// Provider is a replaceable implementation boundary. Providers return
// structured advisory data and receive no tool or execution APIs.
type Provider interface {
	Generate(ctx context.Context, req Request) (Result, error)
}

// Service validates requests and provider output at the Jax boundary.
type Service struct{ provider Provider }

func NewService(provider Provider) (*Service, error) {
	if provider == nil {
		return nil, ErrProviderRequired
	}
	return &Service{provider: provider}, nil
}

func (s *Service) Plan(ctx context.Context, req Request) (Result, error) {
	if s == nil || s.provider == nil {
		return Result{}, ErrProviderRequired
	}
	if err := ValidateRequest(req); err != nil {
		return Result{}, err
	}
	select {
	case <-ctx.Done():
		return Result{}, ctx.Err()
	default:
	}
	result, err := s.provider.Generate(ctx, req)
	if err != nil {
		return Result{}, err
	}
	if err := ValidateResult(result); err != nil {
		return Result{}, err
	}
	return result, nil
}

func ValidateRequest(req Request) error {
	if strings.TrimSpace(req.Symbol) == "" {
		return fmt.Errorf("%w: symbol is required", ErrInvalidRequest)
	}
	if len(req.Context) > maxTextLength {
		return fmt.Errorf("%w: context exceeds %d characters", ErrInvalidRequest, maxTextLength)
	}
	return nil
}

func ValidateResult(result Result) error {
	if strings.TrimSpace(result.Summary) == "" {
		return fmt.Errorf("%w: summary is required", ErrInvalidResult)
	}
	if len(result.Summary) > maxTextLength {
		return fmt.Errorf("%w: summary exceeds %d characters", ErrInvalidResult, maxTextLength)
	}
	if len(result.Steps) > maxSteps {
		return fmt.Errorf("%w: steps exceed %d", ErrInvalidResult, maxSteps)
	}
	for i, step := range result.Steps {
		if strings.TrimSpace(step) == "" || len(step) > maxStepLength {
			return fmt.Errorf("%w: step %d is empty or too long", ErrInvalidResult, i)
		}
	}
	if strings.TrimSpace(result.Action) == "" {
		return fmt.Errorf("%w: action is required", ErrInvalidResult)
	}
	if isForbiddenAction(result.Action) {
		return fmt.Errorf("%w: action cannot grant execution authority", ErrInvalidResult)
	}
	if math.IsNaN(result.Confidence) || math.IsInf(result.Confidence, 0) || result.Confidence < 0 || result.Confidence > 1 {
		return fmt.Errorf("%w: confidence must be finite and between 0 and 1", ErrInvalidResult)
	}
	if len(result.ReasoningNotes) > maxTextLength {
		return fmt.Errorf("%w: reasoning notes exceed %d characters", ErrInvalidResult, maxTextLength)
	}
	return nil
}

func isForbiddenAction(action string) bool {
	switch strings.ToUpper(strings.TrimSpace(action)) {
	case "APPROVE", "EXECUTE", "ORDER", "BROKER", "SUBMIT", "TRADE":
		return true
	default:
		return false
	}
}

// DeterministicProvider is the zero-cost default for the local research
// runtime. It cannot invoke tools or execution.
type DeterministicProvider struct{}

func (DeterministicProvider) Generate(ctx context.Context, req Request) (Result, error) {
	select {
	case <-ctx.Done():
		return Result{}, ctx.Err()
	default:
	}
	return Result{
		Summary:        fmt.Sprintf("Jax native planner produced a bounded advisory HOLD for %s.", strings.ToUpper(strings.TrimSpace(req.Symbol))),
		Steps:          []string{"collect_context", "evaluate_constraints", "defer_execution"},
		Action:         "HOLD",
		Confidence:     0.5,
		ReasoningNotes: "Deterministic local advisory planner; execution authority is none.",
	}, nil
}
