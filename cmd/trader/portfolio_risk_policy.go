package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"jax-trading-assistant/internal/modules/portfoliorisk"
)

const portfolioRiskPolicyFileEnv = "JAX_PORTFOLIO_RISK_POLICY_FILE"

func loadCanonicalPortfolioRiskPolicy() (portfoliorisk.RiskPolicy, error) {
	path := strings.TrimSpace(os.Getenv(portfolioRiskPolicyFileEnv))
	if path == "" {
		return portfoliorisk.RiskPolicy{}, errors.New("canonical portfolio risk policy is not configured")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return portfoliorisk.RiskPolicy{}, fmt.Errorf("read canonical portfolio risk policy: %w", err)
	}
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	var configured portfoliorisk.RiskPolicy
	if err := decoder.Decode(&configured); err != nil {
		return portfoliorisk.RiskPolicy{}, fmt.Errorf("decode canonical portfolio risk policy: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return portfoliorisk.RiskPolicy{}, errors.New("canonical portfolio risk policy must contain exactly one JSON value")
	}
	policy, err := portfoliorisk.BuildRiskPolicy(configured)
	if err != nil {
		return portfoliorisk.RiskPolicy{}, fmt.Errorf("validate canonical portfolio risk policy: %w", err)
	}
	return policy, nil
}
