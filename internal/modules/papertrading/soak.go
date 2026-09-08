package papertrading

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"
)

type SoakStatus string

const (
	SoakRunning   SoakStatus = "RUNNING"
	SoakCompleted SoakStatus = "COMPLETED"
	SoakFailed    SoakStatus = "FAILED"
)

type EvidenceClass string

const (
	SoakInfrastructureDemonstrated EvidenceClass = "SOAK_INFRASTRUCTURE_DEMONSTRATED"
	AcceleratedSyntheticSoak       EvidenceClass = "ACCELERATED_SYNTHETIC_SOAK"
	RealTimeForwardSoak            EvidenceClass = "REAL_TIME_FORWARD_SOAK"
)

type SoakProtocol struct {
	ProtocolID                string        `json:"protocol_id"`
	ContractVersion           string        `json:"contract_version"`
	RequiredDuration          time.Duration `json:"required_duration"`
	MinimumRecommendations    int           `json:"minimum_recommendations"`
	MaximumReconciliationFail int           `json:"maximum_reconciliation_failures"`
	MaximumDuplicateFills     int           `json:"maximum_duplicate_fills"`
	RequireNoLiveActivation   bool          `json:"require_no_live_activation"`
}

func DefaultSoakProtocol() SoakProtocol {
	return SoakProtocol{ProtocolID: "jax-six-month-paper-soak", ContractVersion: SoakProtocolVersion, RequiredDuration: 180 * 24 * time.Hour, MinimumRecommendations: 1, MaximumReconciliationFail: 0, MaximumDuplicateFills: 0, RequireNoLiveActivation: true}
}

func (protocol SoakProtocol) Validate() error {
	if protocol.ProtocolID == "" || protocol.ContractVersion != SoakProtocolVersion || protocol.RequiredDuration <= 0 || protocol.MinimumRecommendations < 0 || protocol.MaximumReconciliationFail < 0 || protocol.MaximumDuplicateFills < 0 || !protocol.RequireNoLiveActivation {
		return ErrInvalidContract
	}
	return nil
}

type SoakObservation struct {
	At                    time.Time `json:"at"`
	Recommendations       int       `json:"recommendations"`
	EligiblePaperIntents  int       `json:"eligible_paper_intents"`
	Orders                int       `json:"orders"`
	Fills                 int       `json:"fills"`
	PartialFills          int       `json:"partial_fills"`
	ReconciliationFailure int       `json:"reconciliation_failures"`
	BreakerEvents         int       `json:"breaker_events"`
	Crashes               int       `json:"crashes"`
	DuplicateFills        int       `json:"duplicate_fills"`
	NegativeBalance       bool      `json:"negative_balance"`
	AuditGap              bool      `json:"audit_gap"`
	LivePathActivated     bool      `json:"live_path_activated"`
}

type SoakRun struct {
	RunID                  string        `json:"run_id"`
	Protocol               SoakProtocol  `json:"protocol"`
	StartedAt              time.Time     `json:"started_at"`
	CurrentAt              time.Time     `json:"current_at"`
	Status                 SoakStatus    `json:"status"`
	EvidenceClass          EvidenceClass `json:"evidence_class"`
	ActualForwardDuration  time.Duration `json:"actual_forward_duration"`
	Recommendations        int           `json:"recommendations"`
	EligiblePaperIntents   int           `json:"eligible_paper_intents"`
	Orders                 int           `json:"orders"`
	Fills                  int           `json:"fills"`
	PartialFills           int           `json:"partial_fills"`
	ReconciliationFailures int           `json:"reconciliation_failures"`
	BreakerEvents          int           `json:"breaker_events"`
	Crashes                int           `json:"crashes"`
	DuplicateFills         int           `json:"duplicate_fills"`
	FailureReasons         []string      `json:"failure_reasons,omitempty"`
}

func NewSoakRun(protocol SoakProtocol, startedAt time.Time, accelerated bool) (SoakRun, error) {
	if err := protocol.Validate(); err != nil || startedAt.IsZero() || startedAt.Location() != time.UTC {
		return SoakRun{}, ErrInvalidArtifact
	}
	run := SoakRun{Protocol: protocol, StartedAt: startedAt, CurrentAt: startedAt, Status: SoakRunning, EvidenceClass: SoakInfrastructureDemonstrated}
	if accelerated {
		run.EvidenceClass = AcceleratedSyntheticSoak
	}
	run.RunID = soakIdentity(run)
	return run, nil
}

func (run *SoakRun) Record(observation SoakObservation) error {
	if run.Status != SoakRunning || observation.At.IsZero() || observation.At.Location() != time.UTC || observation.At.Before(run.CurrentAt) || observation.Recommendations < 0 || observation.EligiblePaperIntents < 0 || observation.Orders < 0 || observation.Fills < 0 || observation.PartialFills < 0 || observation.ReconciliationFailure < 0 || observation.DuplicateFills < 0 {
		return ErrInvalidArtifact
	}
	run.CurrentAt = observation.At
	run.Recommendations += observation.Recommendations
	run.EligiblePaperIntents += observation.EligiblePaperIntents
	run.Orders += observation.Orders
	run.Fills += observation.Fills
	run.PartialFills += observation.PartialFills
	run.ReconciliationFailures += observation.ReconciliationFailure
	run.BreakerEvents += observation.BreakerEvents
	run.Crashes += observation.Crashes
	run.DuplicateFills += observation.DuplicateFills
	if observation.NegativeBalance || observation.AuditGap || observation.LivePathActivated || run.ReconciliationFailures > run.Protocol.MaximumReconciliationFail || run.DuplicateFills > run.Protocol.MaximumDuplicateFills {
		run.Status = SoakFailed
		if observation.NegativeBalance {
			run.FailureReasons = append(run.FailureReasons, "NEGATIVE_BALANCE")
		}
		if observation.AuditGap {
			run.FailureReasons = append(run.FailureReasons, "AUDIT_GAP")
		}
		if observation.LivePathActivated {
			run.FailureReasons = append(run.FailureReasons, "LIVE_PATH_ACTIVATED")
		}
		if run.ReconciliationFailures > run.Protocol.MaximumReconciliationFail {
			run.FailureReasons = append(run.FailureReasons, "RECONCILIATION_FAILURE_THRESHOLD")
		}
		if run.DuplicateFills > run.Protocol.MaximumDuplicateFills {
			run.FailureReasons = append(run.FailureReasons, "DUPLICATE_FILL_THRESHOLD")
		}
	}
	return nil
}

func (run *SoakRun) Complete(at time.Time) error {
	if run.Status != SoakRunning || at.IsZero() || at.Location() != time.UTC || at.Before(run.CurrentAt) || at.Sub(run.StartedAt) < run.Protocol.RequiredDuration || run.Recommendations < run.Protocol.MinimumRecommendations {
		return fmt.Errorf("%w: soak completion criteria not met", ErrInvalidArtifact)
	}
	run.CurrentAt = at
	run.Status = SoakCompleted
	if run.EvidenceClass == RealTimeForwardSoak {
		run.ActualForwardDuration = at.Sub(run.StartedAt)
	}
	return nil
}

func soakIdentity(run SoakRun) string {
	copyRun := run
	copyRun.RunID = ""
	data, _ := json.Marshal(copyRun)
	digest := sha256.Sum256(data)
	return "psr_" + hex.EncodeToString(digest[:])
}
