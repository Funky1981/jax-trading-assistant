package exploratorypaper

import (
	"fmt"
	"math"
	"time"

	"jax-trading-assistant/internal/modules/papertrading"
)

// PriceObservation is the identified, source-backed path used for excursion
// accounting. MFE/MAE are gross currency P&L, before costs.
type PriceObservation struct {
	ObservationID string    `json:"observationId"`
	At            time.Time `json:"at"`
	Price         float64   `json:"price"`
	Source        string    `json:"source"`
}

type Checkpoint struct {
	PositionID       string    `json:"positionId"`
	ThesisID         string    `json:"thesisId"`
	Sessions         int       `json:"sessions"`
	At               time.Time `json:"at"`
	Price            float64   `json:"price"`
	PriceSource      string    `json:"priceSource"`
	EvidenceReviewID string    `json:"evidenceReviewId"`
	GrossPnL         float64   `json:"grossPnl"`
	NetPnL           float64   `json:"netPnl"`
	NetReturn        float64   `json:"netReturn"`
	DataQuality      string    `json:"dataQuality"`
}

type Outcome struct {
	OutcomeID          string              `json:"outcomeId"`
	Mode               string              `json:"mode"`
	TraderModelVersion string              `json:"traderModelVersion"`
	ThesisID           string              `json:"thesisId"`
	ThesisHash         string              `json:"thesisHash"`
	PositionID         string              `json:"positionId"`
	EventID            string              `json:"eventId"`
	IssuerID           string              `json:"issuerId"`
	InstrumentID       string              `json:"instrumentId"`
	Direction          Direction           `json:"direction"`
	Quantity           float64             `json:"quantity"`
	EntryOrderID       string              `json:"entryOrderId"`
	EntryFillID        string              `json:"entryFillId"`
	ExitOrderID        string              `json:"exitOrderId"`
	ExitFillID         string              `json:"exitFillId"`
	EntryAt            time.Time           `json:"entryAt"`
	ExitAt             time.Time           `json:"exitAt"`
	EntryPrice         float64             `json:"entryPrice"`
	ExitPrice          float64             `json:"exitPrice"`
	EntryCosts         float64             `json:"entryCosts"`
	ExitCosts          float64             `json:"exitCosts"`
	TotalCosts         float64             `json:"totalCosts"`
	Costs              float64             `json:"costs"`
	GrossPnL           float64             `json:"grossPnl"`
	NetPnL             float64             `json:"netPnl"`
	ReturnDenominator  float64             `json:"returnDenominator"`
	NetReturn          float64             `json:"netReturn"`
	SessionsHeld       int                 `json:"sessionsHeld"`
	ExitReason         ExitReason          `json:"exitReason"`
	MFE                float64             `json:"mfe"`
	MAE                float64             `json:"mae"`
	ExcursionStatus    string              `json:"excursionStatus"`
	ExcursionKnown     bool                `json:"excursionKnown"`
	PricePath          []PriceObservation  `json:"pricePath"`
	ThesisTransitions  []Transition        `json:"thesisTransitions"`
	NewEvidence        []EvidenceReference `json:"newEvidence"`
	Checkpoints        []Checkpoint        `json:"checkpoints"`
	PolicyVersions     PolicyVersions      `json:"policyVersions"`
	CostModelVersion   string              `json:"costModelVersion"`
}

func (o Outcome) Validate() error {
	if o.Mode != ExploratoryPaperMode {
		return fmt.Errorf("outcome mode must be %s", ExploratoryPaperMode)
	}
	if o.TraderModelVersion != TraderModelVersion || o.OutcomeID == "" || o.ThesisID == "" || o.ThesisHash == "" || o.PositionID == "" || o.EventID == "" || o.IssuerID == "" || o.InstrumentID == "" || o.EntryOrderID == "" || o.EntryFillID == "" || o.ExitOrderID == "" || o.ExitFillID == "" || o.CostModelVersion == "" {
		return fmt.Errorf("outcome identity and trader model version are required")
	}
	if o.Direction != DirectionLong && o.Direction != DirectionShort || !finitePositive(o.Quantity) || o.EntryAt.IsZero() || o.ExitAt.IsZero() || o.EntryAt.Location() != time.UTC || o.ExitAt.Location() != time.UTC || o.ExitAt.Before(o.EntryAt) || !finitePositive(o.EntryPrice) || !finitePositive(o.ExitPrice) {
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
	if !finiteNonNegative(o.EntryCosts) || !finiteNonNegative(o.ExitCosts) || !finiteNonNegative(o.TotalCosts) || math.Abs(o.TotalCosts-(o.EntryCosts+o.ExitCosts)) > 1e-9 || math.Abs(o.Costs-o.TotalCosts) > 1e-9 {
		return fmt.Errorf("outcome costs are inconsistent")
	}
	direction := 1.0
	if o.Direction == DirectionShort {
		direction = -1
	}
	expectedGross := (o.ExitPrice - o.EntryPrice) * o.Quantity * direction
	if math.Abs(o.GrossPnL-expectedGross) > 1e-9 || math.Abs(o.NetPnL-(o.GrossPnL-o.TotalCosts)) > 1e-9 || !finitePositive(o.ReturnDenominator) || math.Abs(o.ReturnDenominator-o.EntryPrice*o.Quantity) > 1e-9 || math.Abs(o.NetReturn-(o.NetPnL/o.ReturnDenominator)) > 1e-9 {
		return fmt.Errorf("outcome arithmetic is inconsistent")
	}
	if o.ExcursionStatus != "COMPLETE" && o.ExcursionStatus != "INCOMPLETE" && o.ExcursionStatus != "UNKNOWN" {
		return fmt.Errorf("excursion status is invalid")
	}
	if o.ExcursionKnown {
		if o.ExcursionStatus != "COMPLETE" || o.MFE < 0 || o.MAE > 0 || len(o.PricePath) == 0 {
			return fmt.Errorf("known MFE/MAE requires a complete price path")
		}
		mfe, mae, err := calculateExcursions(o.Direction, o.Quantity, o.EntryPrice, o.PricePath)
		if err != nil || math.Abs(o.MFE-mfe) > 1e-9 || math.Abs(o.MAE-mae) > 1e-9 {
			return fmt.Errorf("MFE/MAE does not match the identified price path")
		}
	} else if o.ExcursionStatus == "COMPLETE" {
		return fmt.Errorf("complete excursion status requires known MFE/MAE")
	}
	if err := o.PolicyVersions.Validate(); err != nil {
		return err
	}
	seen := map[int]bool{}
	for _, c := range o.Checkpoints {
		if c.PositionID != o.PositionID || c.ThesisID != o.ThesisID || c.Sessions < 1 || c.Sessions > o.SessionsHeld || c.At.IsZero() || c.At.Location() != time.UTC || c.At.Before(o.EntryAt) || c.At.After(o.ExitAt) || !finitePositive(c.Price) || c.PriceSource == "" || c.DataQuality == "" {
			return fmt.Errorf("checkpoint identity, session, timestamp, or price provenance is invalid")
		}
		if seen[c.Sessions] {
			return fmt.Errorf("duplicate checkpoint at session %d", c.Sessions)
		}
		expectedGross = (c.Price - o.EntryPrice) * o.Quantity * direction
		expectedNet := expectedGross - o.TotalCosts
		if math.Abs(c.GrossPnL-expectedGross) > 1e-9 || math.Abs(c.NetPnL-expectedNet) > 1e-9 || math.Abs(c.NetReturn-(expectedNet/o.ReturnDenominator)) > 1e-9 {
			return fmt.Errorf("checkpoint arithmetic is inconsistent")
		}
		seen[c.Sessions] = true
	}
	return nil
}

func BuildOutcomeFromFills(position Position, binding EntryBinding, entry, exit papertrading.PaperFill, exitReason ExitReason, sessions int, path []PriceObservation, checkpoints []Checkpoint) (Outcome, error) {
	if err := position.VerifyFrozenIdentity(); err != nil {
		return Outcome{}, err
	}
	if err := position.VerifyApprovalBinding(binding); err != nil || binding.ThesisID != position.Thesis.ThesisID {
		return Outcome{}, fmt.Errorf("approval binding is invalid for outcome")
	}
	if err := entry.Validate(); err != nil {
		return Outcome{}, fmt.Errorf("entry fill: %w", err)
	}
	if err := exit.Validate(); err != nil {
		return Outcome{}, fmt.Errorf("exit fill: %w", err)
	}
	if entry.InstrumentID != position.Thesis.InstrumentID || exit.InstrumentID != entry.InstrumentID || entry.Direction != string(position.Thesis.Direction) || exit.Direction == entry.Direction || entry.Quantity != exit.Quantity {
		return Outcome{}, fmt.Errorf("fills do not match the exploratory position")
	}
	entryCosts := entry.Costs.SpreadCost + entry.Costs.SlippageCost + entry.Costs.Commission
	exitCosts := exit.Costs.SpreadCost + exit.Costs.SlippageCost + exit.Costs.Commission
	direction := 1.0
	if position.Thesis.Direction == DirectionShort {
		direction = -1
	}
	gross := (exit.Price - entry.Price) * entry.Quantity * direction
	total := entryCosts + exitCosts
	net := gross - total
	denominator := entry.Price * entry.Quantity
	outcome := Outcome{OutcomeID: "outcome_" + ThesisContentHash(position.Thesis)[7:23], Mode: ExploratoryPaperMode, TraderModelVersion: TraderModelVersion, ThesisID: position.Thesis.ThesisID, ThesisHash: position.FrozenThesisHash, PositionID: position.PositionID, EventID: position.Thesis.EventID, IssuerID: position.Thesis.IssuerID, InstrumentID: position.Thesis.InstrumentID, Direction: position.Thesis.Direction, Quantity: entry.Quantity, EntryOrderID: entry.OrderID, EntryFillID: entry.FillID, ExitOrderID: exit.OrderID, ExitFillID: exit.FillID, EntryAt: entry.FilledAt, ExitAt: exit.FilledAt, EntryPrice: entry.Price, ExitPrice: exit.Price, EntryCosts: entryCosts, ExitCosts: exitCosts, TotalCosts: total, Costs: total, GrossPnL: gross, NetPnL: net, ReturnDenominator: denominator, NetReturn: net / denominator, SessionsHeld: sessions, ExitReason: exitReason, ExcursionStatus: "UNKNOWN", PolicyVersions: binding.PolicyVersions, CostModelVersion: entry.Costs.ModelID, PricePath: append([]PriceObservation(nil), path...), ThesisTransitions: append([]Transition(nil), position.Transitions...), Checkpoints: checkpoints}
	if len(path) > 0 {
		mfe, mae, err := calculateExcursions(outcome.Direction, outcome.Quantity, outcome.EntryPrice, path)
		if err != nil {
			return Outcome{}, err
		}
		outcome.MFE, outcome.MAE, outcome.ExcursionKnown, outcome.ExcursionStatus = mfe, mae, true, "COMPLETE"
	} else {
		outcome.ExcursionStatus = "INCOMPLETE"
	}
	if err := outcome.Validate(); err != nil {
		return Outcome{}, err
	}
	return outcome, nil
}

func calculateExcursions(direction Direction, quantity, entryPrice float64, path []PriceObservation) (float64, float64, error) {
	if !finitePositive(quantity) || !finitePositive(entryPrice) || len(path) == 0 {
		return 0, 0, fmt.Errorf("identified price path is required")
	}
	if direction != DirectionLong && direction != DirectionShort {
		return 0, 0, fmt.Errorf("direction is required for excursion calculation")
	}
	multiplier := 1.0
	if direction == DirectionShort {
		multiplier = -1
	}
	mfe, mae := 0.0, 0.0
	for _, observation := range path {
		if observation.ObservationID == "" || observation.Source == "" || observation.At.IsZero() || observation.At.Location() != time.UTC || !finitePositive(observation.Price) {
			return 0, 0, fmt.Errorf("price path observation is invalid")
		}
		value := (observation.Price - entryPrice) * quantity * multiplier
		if value > mfe {
			mfe = value
		}
		if value < mae {
			mae = value
		}
	}
	return mfe, mae, nil
}

func finitePositive(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0) && value > 0
}
func finiteNonNegative(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0) && value >= 0
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
