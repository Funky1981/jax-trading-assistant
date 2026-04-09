package harness

import (
	"fmt"
	"time"
)

type Mode string

const (
	ModeResearch Mode = "research"
	ModePaper    Mode = "paper"
	ModeLive     Mode = "live"
)

type Policy struct {
	Mode              Mode
	AdvisoryOnly      bool
	AllowExternalData bool
	AllowPriceClaims  bool
	MaxToolCalls      int
	MaxSteps          int
	ToolTimeout       time.Duration
}

func DefaultPolicy(mode Mode) Policy {
	policy := Policy{
		Mode:              mode,
		AdvisoryOnly:      true,
		AllowExternalData: false,
		AllowPriceClaims:  false,
		MaxToolCalls:      3,
		MaxSteps:          5,
		ToolTimeout:       3 * time.Second,
	}
	switch mode {
	case ModePaper:
		policy.MaxToolCalls = 2
		policy.MaxSteps = 4
		policy.ToolTimeout = 2 * time.Second
	case ModeLive:
		policy.MaxToolCalls = 1
		policy.MaxSteps = 3
		policy.ToolTimeout = 1500 * time.Millisecond
	}
	return policy
}

func (p Policy) CheckToolAllowed(def ToolDefinition) error {
	if def.Name == "" {
		return fmt.Errorf("tool definition missing name")
	}
	if !def.ReadOnly {
		return fmt.Errorf("tool %s is not read-only", def.Name)
	}
	if p.Mode == ModePaper && def.EvidenceLevel == EvidenceWeakInference {
		return fmt.Errorf("tool %s is blocked in %s mode because it relies on weak inference", def.Name, p.Mode)
	}
	if p.Mode == ModeLive {
		if def.EvidenceLevel != EvidenceHardInternal {
			return fmt.Errorf("tool %s is blocked in %s mode because it is not hard internal evidence", def.Name, p.Mode)
		}
		switch def.FreshnessExpectation {
		case "historical_snapshot", "reference_data", "local_docs_snapshot":
			return fmt.Errorf("tool %s is blocked in %s mode because its freshness is %s", def.Name, p.Mode, def.FreshnessExpectation)
		}
	}
	return nil
}

func (p Policy) CheckAnswerAllowed(answer string) error {
	forbidden := []string{
		"I executed",
		"I approved",
		"I placed the trade",
	}
	for _, phrase := range forbidden {
		if containsInsensitive(answer, phrase) {
			return fmt.Errorf("forbidden action language detected: %s", phrase)
		}
	}
	return nil
}
