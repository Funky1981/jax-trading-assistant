package exploratorypaper

import (
	"fmt"
	"time"
)

type Checkpoint struct {
	Sessions  int     `json:"sessions"`
	Price     float64 `json:"price"`
	NetReturn float64 `json:"netReturn"`
}

type Outcome struct {
	OutcomeID          string              `json:"outcomeId"`
	Mode               string              `json:"mode"`
	TraderModelVersion string              `json:"traderModelVersion"`
	ThesisID           string              `json:"thesisId"`
	EventID            string              `json:"eventId"`
	IssuerID           string              `json:"issuerId"`
	InstrumentID       string              `json:"instrumentId"`
	EntryAt            time.Time           `json:"entryAt"`
	ExitAt             time.Time           `json:"exitAt"`
	EntryPrice         float64             `json:"entryPrice"`
	ExitPrice          float64             `json:"exitPrice"`
	Costs              float64             `json:"costs"`
	NetReturn          float64             `json:"netReturn"`
	SessionsHeld       int                 `json:"sessionsHeld"`
	ExitReason         ExitReason          `json:"exitReason"`
	MFE                float64             `json:"mfe"`
	MAE                float64             `json:"mae"`
	ThesisTransitions  []Transition        `json:"thesisTransitions"`
	NewEvidence        []EvidenceReference `json:"newEvidence"`
	Checkpoints        []Checkpoint        `json:"checkpoints"`
	PolicyVersions     PolicyVersions      `json:"policyVersions"`
}

func (o Outcome) Validate() error {
	if o.Mode != ExploratoryPaperMode {
		return fmt.Errorf("outcome mode must be %s", ExploratoryPaperMode)
	}
	if o.TraderModelVersion == "" || o.OutcomeID == "" || o.ThesisID == "" || o.EventID == "" || o.IssuerID == "" || o.InstrumentID == "" {
		return fmt.Errorf("outcome identity and trader model version are required")
	}
	if o.EntryAt.IsZero() || o.ExitAt.IsZero() || o.ExitAt.Before(o.EntryAt) || o.EntryPrice <= 0 || o.ExitPrice <= 0 {
		return fmt.Errorf("outcome entry/exit prices and timestamps are invalid")
	}
	if o.SessionsHeld < 1 || o.SessionsHeld > 5 {
		return fmt.Errorf("outcome horizon must be 1-5 trading sessions")
	}
	switch o.ExitReason {
	case ExitStop, ExitTarget, ExitThesisInvalidated, ExitRiskKill, ExitTimeLimit, ExitManualOperator:
	default:
		return fmt.Errorf("invalid exit reason %q", o.ExitReason)
	}
	if o.MFE < 0 || o.MAE > 0 {
		return fmt.Errorf("MFE/MAE must be nonnegative/nonpositive")
	}
	if err := o.PolicyVersions.Validate(); err != nil {
		return err
	}
	seen := map[int]bool{}
	for _, c := range o.Checkpoints {
		if c.Sessions != 1 && c.Sessions != 2 && c.Sessions != 3 && c.Sessions != 5 {
			return fmt.Errorf("checkpoint must be at session 1, 2, 3, or 5")
		}
		if seen[c.Sessions] {
			return fmt.Errorf("duplicate checkpoint at session %d", c.Sessions)
		}
		seen[c.Sessions] = true
	}
	return nil
}

type FutureOnlyIdentity struct {
	RunID              string
	StartedAt          time.Time
	FirstObservationAt time.Time
	Retrospective      bool
}

func ValidateEvidenceAdmission(mode string, formalRequested bool, identity FutureOnlyIdentity) error {
	if mode == ExploratoryPaperMode && formalRequested {
		return fmt.Errorf("exploratory paper evidence cannot populate a formal run")
	}
	if mode == FormalForwardMode {
		if identity.RunID == "" || identity.StartedAt.IsZero() || identity.FirstObservationAt.IsZero() || identity.Retrospective {
			return fmt.Errorf("formal paper evidence requires future-only run identity")
		}
	}
	return nil
}

func CanPopulateFormal(mode string) bool { return mode == FormalForwardMode }
func RelabelExploratoryAsFormal() error {
	return fmt.Errorf("exploratory paper records are immutable and cannot be relabelled as formal")
}
