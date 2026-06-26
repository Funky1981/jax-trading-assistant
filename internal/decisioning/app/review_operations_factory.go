package app

import (
	"fmt"

	"jax-trading-assistant/internal/decisioning/operations"
	"jax-trading-assistant/internal/decisioning/operator"
)

type ReviewOperationsService struct {
	config   ReviewOperationsConfig
	repo     operations.Repository
	operator operator.Service
}

func NewReviewOperationsService(config ReviewOperationsConfig, repo operations.Repository) (ReviewOperationsService, error) {
	if repo == nil {
		return ReviewOperationsService{}, fmt.Errorf("review operations repository is required")
	}
	normalized := config.normalize()
	return ReviewOperationsService{
		config:   normalized,
		repo:     repo,
		operator: operator.NewService(repo),
	}, nil
}

func NewInMemoryReviewOperationsService(config ReviewOperationsConfig) (ReviewOperationsService, error) {
	return NewReviewOperationsService(config, operations.NewMemoryRepository())
}

func (s ReviewOperationsService) SafetyDefaults() SafetyDefaults {
	return SafetyDefaults{
		RequiresHumanApproval: true,
		AutoApplyAllowed:      false,
		ReadOnly:              true,
		ForbiddenActions:      append([]string{}, s.config.ForbiddenActions...),
	}
}
