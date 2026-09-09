package advancedquant

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

const OOSAdmissionContractV1 = "jax.phase12.oos_admission/v1"

type OOSAdmissionState string

const (
	OOSSealed          OOSAdmissionState = "SEALED"
	OOSAdmissible      OOSAdmissionState = "ADMISSIBLE_FOR_OOS"
	OOSOpenedFormal    OOSAdmissionState = "OPENED_FORMAL_OOS"
	OOSContaminated    OOSAdmissionState = "CONTAMINATED"
	OOSExploratoryOnly OOSAdmissionState = "EXPLORATORY_ONLY"
)

const ProgressionProceedToOOS = "PROCEED_TO_FORMAL_OOS"

// OOSAdmission is an immutable admission decision. State is a one-way
// lifecycle marker; the identity intentionally covers only the prerequisite
// bindings, so opening or contaminating the same admission cannot create a
// replacement pristine identity.
type OOSAdmission struct {
	ContractVersion        string            `json:"contract_version"`
	ID                     string            `json:"id"`
	HypothesisID           string            `json:"hypothesis_id"`
	DatasetID              string            `json:"dataset_id"`
	ClassifierID           string            `json:"classifier_id"`
	ProtocolID             string            `json:"protocol_id"`
	DevelopmentResultID    string            `json:"development_result_id"`
	ValidationResultID     string            `json:"validation_result_id"`
	FalsificationSuiteID   string            `json:"falsification_suite_id"`
	CandidateID            string            `json:"candidate_id"`
	ProgressionRuleID      string            `json:"progression_rule_id"`
	ProgressionDecisionID  string            `json:"progression_decision_id"`
	ProgressionDecision    string            `json:"progression_decision"`
	CandidateFreezeID      string            `json:"candidate_freeze_id"`
	OOSPartitionID         string            `json:"oos_partition_id"`
	DevelopmentCompleted   time.Time         `json:"development_completed_at"`
	ValidationCompleted    time.Time         `json:"validation_completed_at"`
	FalsificationCompleted time.Time         `json:"falsification_completed_at"`
	DecisionAt             time.Time         `json:"decision_at"`
	CandidateFrozenAt      time.Time         `json:"candidate_frozen_at"`
	State                  OOSAdmissionState `json:"state"`
	OpenedAt               *time.Time        `json:"opened_at,omitempty"`
	ContaminationReason    string            `json:"contamination_reason,omitempty"`
}

func NewOOSAdmission(admission OOSAdmission) (OOSAdmission, error) {
	admission.ContractVersion = OOSAdmissionContractV1
	admission.ID = oosAdmissionID(admission)
	if admission.State == "" {
		admission.State = OOSAdmissible
	}
	if err := admission.Validate(); err != nil {
		return OOSAdmission{}, err
	}
	return admission, nil
}

func (a OOSAdmission) Validate() error {
	if a.ContractVersion != OOSAdmissionContractV1 || !validHashID(a.ID, "oosadm_") ||
		strings.TrimSpace(a.HypothesisID) == "" || strings.TrimSpace(a.DatasetID) == "" ||
		strings.TrimSpace(a.ClassifierID) == "" || strings.TrimSpace(a.ProtocolID) == "" ||
		strings.TrimSpace(a.DevelopmentResultID) == "" || strings.TrimSpace(a.ValidationResultID) == "" ||
		strings.TrimSpace(a.FalsificationSuiteID) == "" || strings.TrimSpace(a.CandidateID) == "" ||
		strings.TrimSpace(a.ProgressionRuleID) == "" || strings.TrimSpace(a.ProgressionDecisionID) == "" ||
		a.ProgressionDecision != ProgressionProceedToOOS || strings.TrimSpace(a.CandidateFreezeID) == "" ||
		strings.TrimSpace(a.OOSPartitionID) == "" {
		return fmt.Errorf("OOS admission is missing immutable prerequisite bindings")
	}
	for name, value := range map[string]time.Time{
		"development": a.DevelopmentCompleted, "validation": a.ValidationCompleted,
		"falsification": a.FalsificationCompleted, "decision": a.DecisionAt, "candidate freeze": a.CandidateFrozenAt,
	} {
		if value.IsZero() || value.Location() != time.UTC {
			return fmt.Errorf("%s completion timestamp must be UTC", name)
		}
	}
	if !a.ValidationCompleted.After(a.DevelopmentCompleted) || !a.FalsificationCompleted.After(a.ValidationCompleted) ||
		!a.DecisionAt.After(a.FalsificationCompleted) || !a.CandidateFrozenAt.After(a.DecisionAt) {
		return fmt.Errorf("OOS prerequisites are not completed in order")
	}
	switch a.State {
	case OOSSealed:
		return fmt.Errorf("sealed OOS admission is not a complete admission record")
	case OOSAdmissible:
		if a.OpenedAt != nil || a.ContaminationReason != "" {
			return fmt.Errorf("admissible OOS admission has lifecycle fields")
		}
	case OOSOpenedFormal:
		if !validUTCOptional(a.OpenedAt) || a.ContaminationReason != "" {
			return fmt.Errorf("opened OOS admission has invalid lifecycle fields")
		}
	case OOSContaminated, OOSExploratoryOnly:
		if !validUTCOptional(a.OpenedAt) || strings.TrimSpace(a.ContaminationReason) == "" {
			return fmt.Errorf("contaminated OOS admission requires an opening time and reason")
		}
	default:
		return fmt.Errorf("unsupported OOS admission state %q", a.State)
	}
	if a.ID != oosAdmissionID(a) {
		return fmt.Errorf("OOS admission identity does not match immutable bindings")
	}
	return nil
}

func (a *OOSAdmission) Open(now time.Time) error {
	if err := a.Validate(); err != nil {
		return err
	}
	if a.State != OOSAdmissible {
		return fmt.Errorf("OOS admission cannot open from state %s", a.State)
	}
	if now.IsZero() || now.Location() != time.UTC || now.Before(a.CandidateFrozenAt) {
		return fmt.Errorf("OOS opening timestamp is invalid or precedes candidate freeze")
	}
	a.State = OOSOpenedFormal
	a.OpenedAt = &now
	return nil
}

func (a *OOSAdmission) MarkContaminated(reason string) error {
	if a.State != OOSOpenedFormal {
		return fmt.Errorf("only opened formal OOS can become contaminated")
	}
	if strings.TrimSpace(reason) == "" {
		return fmt.Errorf("contaminated OOS requires a reason")
	}
	a.State = OOSContaminated
	a.ContaminationReason = reason
	return nil
}

func (a *OOSAdmission) MarkExploratoryOnly(reason string) error {
	if a.State != OOSContaminated {
		return fmt.Errorf("only contaminated OOS can become exploratory-only")
	}
	if strings.TrimSpace(reason) == "" {
		return fmt.Errorf("exploratory-only state requires a reason")
	}
	a.State = OOSExploratoryOnly
	a.ContaminationReason = reason
	return nil
}

func (a OOSAdmission) IsPristineFormal() bool { return a.State == OOSOpenedFormal }

func validUTCOptional(value *time.Time) bool {
	return value != nil && !value.IsZero() && value.Location() == time.UTC
}

func oosAdmissionID(a OOSAdmission) string {
	a.ID = ""
	a.State = ""
	a.OpenedAt = nil
	a.ContaminationReason = ""
	b, _ := json.Marshal(a)
	d := sha256.Sum256(b)
	return "oosadm_" + hex.EncodeToString(d[:])
}
