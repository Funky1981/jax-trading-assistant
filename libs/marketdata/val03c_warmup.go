package marketdata

import (
	"errors"
	"fmt"
)

// VAL03CInitializationSessions is the longest frozen operational indicator
// lookback. This helper is intentionally limited to session validity and does
// not calculate indicators or strategy decisions.
const VAL03CInitializationSessions = 200

type VAL03CInitializationState string

const (
	VAL03CWarmupOnly             VAL03CInitializationState = "WARMUP_ONLY"
	VAL03CInitializationComplete VAL03CInitializationState = "INITIALIZATION_COMPLETE"
)

// VAL03CSession is the structural input to the initialization contract.
// Valid means the synchronized bar-family session passed data-quality checks.
// StructuralBoundary resets the post-action valid-session count before the
// current session is counted.
type VAL03CSession struct {
	ProviderDate       string
	Valid              bool
	StructuralBoundary bool
}

type VAL03CInitializationResult struct {
	FirstValidSession      string                    `json:"first_valid_session"`
	InitializationComplete string                    `json:"initialization_complete_session"`
	WarmupOnlyCount        int                       `json:"warmup_only_count"`
	ValidSessionCount      int                       `json:"valid_session_count"`
	State                  VAL03CInitializationState `json:"state"`
}

// DeriveVAL03CInitialization finds the first session with 200 valid sessions
// in the current structural regime. It rejects unordered/duplicate input and
// any sealed-holdout date before it can affect readiness.
func DeriveVAL03CInitialization(sessions []VAL03CSession) (VAL03CInitializationResult, error) {
	if len(sessions) == 0 {
		return VAL03CInitializationResult{}, errors.New("VAL-03C initialization requires sessions")
	}
	ordered := append([]VAL03CSession(nil), sessions...)
	for i := range ordered {
		if err := ValidateVAL03BProviderDate(ordered[i].ProviderDate); err != nil {
			return VAL03CInitializationResult{}, err
		}
		if i > 0 && ordered[i].ProviderDate == ordered[i-1].ProviderDate {
			return VAL03CInitializationResult{}, fmt.Errorf("VAL-03C duplicate session: %s", ordered[i].ProviderDate)
		}
		if i > 0 && ordered[i].ProviderDate < ordered[i-1].ProviderDate {
			return VAL03CInitializationResult{}, errors.New("VAL-03C sessions must be chronological")
		}
	}

	result := VAL03CInitializationResult{State: VAL03CWarmupOnly}
	postBoundaryValid := 0
	initialized := false
	for _, session := range ordered {
		if session.StructuralBoundary {
			postBoundaryValid = 0
			initialized = false
			result.State = VAL03CWarmupOnly
		}
		if !session.Valid {
			continue
		}
		if result.FirstValidSession == "" {
			result.FirstValidSession = session.ProviderDate
		}
		result.ValidSessionCount++
		postBoundaryValid++
		if !initialized {
			if postBoundaryValid >= VAL03CInitializationSessions {
				initialized = true
				result.InitializationComplete = session.ProviderDate
				result.State = VAL03CInitializationComplete
			} else {
				result.WarmupOnlyCount++
			}
		}
	}
	if result.FirstValidSession == "" {
		return VAL03CInitializationResult{}, errors.New("VAL-03C initialization has no valid synchronized session")
	}
	return result, nil
}
