package exploratorypaper

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PilotStore is intentionally separate from the paper ledger. It owns the
// prospective sample boundary and append-only evidence/opportunity audit trail;
// PAPER-01 lifecycle and papertrading remain the execution truth.
type PilotStore interface {
	CreateDraft(ctx context.Context, record PilotRecord) error
	GetPilot(ctx context.Context, pilotID string) (PilotRecord, error)
	MarkReadyForExternalReview(ctx context.Context, pilotID string) error
	ActivateWithExternalAuthorization(ctx context.Context, pilotID string, auth ExternalActivationAuthorization) error
	RecordOpportunity(ctx context.Context, pilotID string, intake OpportunityIntake) (PilotOpportunity, error)
	RecordDecision(ctx context.Context, pilotID, opportunityID string, decision OpportunityDecision) (PilotOpportunity, error)
	LinkLifecycle(ctx context.Context, pilotID, opportunityID string, lifecycleID, candidateID, thesisID string, entryAt time.Time, entryEvidenceIDs []string) error
	RecordEvidence(ctx context.Context, pilotID, opportunityID string, input EvidenceInput) (PilotEvidenceRecord, error)
	RecordMarketObservation(ctx context.Context, pilotID, opportunityID string, quality MarketObservationQuality) error
	RecordOutcome(ctx context.Context, pilotID, opportunityID string, outcome PilotOutcomeSummary) error
	RecordIncident(ctx context.Context, incident PilotIncident) error
	ListOpportunities(ctx context.Context, pilotID string) ([]PilotOpportunity, error)
	ReadModel(ctx context.Context, pilotID string) (PilotReadModel, error)
}

type PilotPostgresStore struct {
	pool *pgxpool.Pool
	now  func() time.Time
}

func NewPilotPostgresStore(pool *pgxpool.Pool) *PilotPostgresStore {
	return &PilotPostgresStore{pool: pool, now: func() time.Time { return time.Now().UTC() }}
}

func (s *PilotPostgresStore) CreateDraft(ctx context.Context, record PilotRecord) error {
	if s == nil || s.pool == nil {
		return ErrFailedClosed
	}
	if record.Status != PilotStatusDraft {
		return fmt.Errorf("new pilot identity must start in DRAFT")
	}
	if err := record.Identity.Validate(record.Protocol); err != nil {
		return err
	}
	payload, err := json.Marshal(record)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `INSERT INTO exploratory_paper_pilots(pilot_id,mode,status,protocol_version,protocol_hash,created_at,payload,formal_evidence_eligible) VALUES($1,$2,$3,$4,$5,$6,$7,FALSE) ON CONFLICT(pilot_id) DO NOTHING`, record.Identity.PilotID, PilotMode, record.Status, record.Identity.ProtocolVersion, record.Identity.ProtocolContentHash, record.Identity.CreatedAt, payload)
	if err != nil {
		return failClosed("create PAPER-02 draft", err)
	}
	var existing []byte
	if err := s.pool.QueryRow(ctx, `SELECT payload FROM exploratory_paper_pilots WHERE pilot_id=$1`, record.Identity.PilotID).Scan(&existing); err != nil {
		return failClosed("verify PAPER-02 identity", err)
	}
	if !sameJSON(existing, payload) {
		return ErrPilotIdentityConflict
	}
	return nil
}

func (s *PilotPostgresStore) GetPilot(ctx context.Context, pilotID string) (PilotRecord, error) {
	var payload []byte
	if err := s.pool.QueryRow(ctx, `SELECT payload FROM exploratory_paper_pilots WHERE pilot_id=$1`, pilotID).Scan(&payload); err != nil {
		return PilotRecord{}, err
	}
	var record PilotRecord
	if err := json.Unmarshal(payload, &record); err != nil {
		return PilotRecord{}, failClosed("decode PAPER-02 identity", err)
	}
	return record, nil
}

func (s *PilotPostgresStore) transition(ctx context.Context, pilotID, from, to string, record PilotRecord) error {
	payload, err := json.Marshal(record)
	if err != nil {
		return err
	}
	result, err := s.pool.Exec(ctx, `UPDATE exploratory_paper_pilots SET status=$2,payload=$3 WHERE pilot_id=$1 AND status=$4 AND formal_evidence_eligible=FALSE`, pilotID, to, payload, from)
	if err != nil {
		return failClosed("transition PAPER-02 status", err)
	}
	if result.RowsAffected() != 1 {
		return ErrPilotIdentityConflict
	}
	return nil
}

func (s *PilotPostgresStore) MarkReadyForExternalReview(ctx context.Context, pilotID string) error {
	record, err := s.GetPilot(ctx, pilotID)
	if err != nil {
		return err
	}
	if record.Status != PilotStatusDraft {
		return fmt.Errorf("pilot can become READY only from DRAFT")
	}
	if err := record.Identity.Validate(record.Protocol); err != nil {
		return err
	}
	record.Status = PilotStatusReadyForExternalReview
	return s.transition(ctx, pilotID, PilotStatusDraft, PilotStatusReadyForExternalReview, record)
}

// ActivateWithExternalAuthorization is a future external-review handoff. It
// is deliberately never called by this package's runtime or tests in the
// readiness commit; Codex does not activate PAPER-02.
func (s *PilotPostgresStore) ActivateWithExternalAuthorization(ctx context.Context, pilotID string, auth ExternalActivationAuthorization) error {
	if err := auth.Validate(); err != nil {
		return err
	}
	record, err := s.GetPilot(ctx, pilotID)
	if err != nil {
		return err
	}
	if record.Status != PilotStatusReadyForExternalReview {
		return fmt.Errorf("pilot activation requires READY_FOR_EXTERNAL_REVIEW")
	}
	now := auth.AuthorizedAt.UTC()
	record.Status = PilotStatusActive
	record.Identity.StartTimestamp = &now
	if err := record.Identity.Validate(record.Protocol); err != nil {
		return err
	}
	record.Identity.StartTimestamp = &now
	return s.transition(ctx, pilotID, PilotStatusReadyForExternalReview, PilotStatusActive, record)
}

func (s *PilotPostgresStore) admission(ctx context.Context, pilotID string) (PilotRecord, error) {
	record, err := s.GetPilot(ctx, pilotID)
	if err != nil {
		return PilotRecord{}, err
	}
	if record.Status != PilotStatusActive {
		return PilotRecord{}, ErrPilotNotAdmitting
	}
	return record, nil
}

func stableOpportunityID(pilotID, sourceEventIdentity string) string {
	return "ppo_" + contentHash(pilotID + "|" + sourceEventIdentity)[7:]
}

func contentHash(value string) string {
	data := []byte(value)
	digest := sha256Bytes(data)
	return "sha256:" + hexBytes(digest)
}

// Small local wrappers keep the pilot file independent of the thesis hash
// representation and make identity derivation obvious in audit logs.
func sha256Bytes(data []byte) [32]byte { return sha256.Sum256(data) }
func hexBytes(data [32]byte) string    { return hex.EncodeToString(data[:]) }

func (s *PilotPostgresStore) RecordOpportunity(ctx context.Context, pilotID string, intake OpportunityIntake) (PilotOpportunity, error) {
	pilot, err := s.admission(ctx, pilotID)
	if err != nil {
		return PilotOpportunity{}, err
	}
	if intake.OpportunityID == "" {
		intake.OpportunityID = stableOpportunityID(pilotID, intake.SourceEventIdentity)
	}
	if intake.FirstSeenAt.IsZero() || intake.IngestionAt.IsZero() {
		return PilotOpportunity{}, fmt.Errorf("prospective opportunity timestamps are required")
	}
	classification := OpportunityClassification(PilotClassificationUnresolved)
	now := s.now().UTC()
	versions := PolicyVersions{TraderModel: pilot.Identity.TraderModelVersion, CandidatePolicy: pilot.Identity.EvidencePolicyVersion, RiskPolicy: pilot.Identity.RiskPolicyVersion, EntryPolicy: pilot.Identity.EntryPolicyVersion, ExitPolicy: pilot.Identity.ExitPolicyVersion, CostModel: pilot.Identity.CostModelVersion, ThesisContract: ContractVersion}
	opportunity := PilotOpportunity{PilotID: pilotID, OpportunityID: intake.OpportunityID, EventID: intake.EventID, SourceEventIdentity: intake.SourceEventIdentity, EventCategory: intake.EventCategory, FirstSeenAt: intake.FirstSeenAt.UTC(), PublicationAt: utcPointer(intake.PublicationAt), IngestionAt: intake.IngestionAt.UTC(), SourceProvenance: intake.SourceProvenance, IssuerID: intake.IssuerID, InstrumentID: intake.InstrumentID, ResolutionState: intake.ResolutionState, EvidenceState: intake.EvidenceState, FinalClassification: classification, MissingDataFlags: intake.MissingDataFlags, PolicyModelVersions: versions, CreatedAt: now, UpdatedAt: now}
	if err := opportunity.ValidateForPilot(pilot); err != nil {
		return PilotOpportunity{}, err
	}
	payload, err := json.Marshal(opportunity)
	if err != nil {
		return PilotOpportunity{}, err
	}
	_, err = s.pool.Exec(ctx, `INSERT INTO exploratory_paper_opportunities(opportunity_id,pilot_id,event_identity,first_seen_at,payload,final_classification,formal_evidence_eligible,created_at,updated_at) VALUES($1,$2,$3,$4,$5,$6,FALSE,$7,$7) ON CONFLICT(pilot_id,event_identity) DO NOTHING`, opportunity.OpportunityID, pilotID, opportunity.SourceEventIdentity, opportunity.FirstSeenAt, payload, classification, now)
	if err != nil {
		return PilotOpportunity{}, failClosed("record prospective opportunity", err)
	}
	var existing []byte
	if err := s.pool.QueryRow(ctx, `SELECT payload FROM exploratory_paper_opportunities WHERE pilot_id=$1 AND event_identity=$2`, pilotID, opportunity.SourceEventIdentity).Scan(&existing); err != nil {
		return PilotOpportunity{}, failClosed("verify opportunity identity", err)
	}
	if !sameJSON(existing, payload) {
		return PilotOpportunity{}, ErrPilotIdentityConflict
	}
	if err := s.appendOpportunityEvent(ctx, opportunity, "INTAKE", "intake:"+opportunity.OpportunityID, opportunity); err != nil {
		return PilotOpportunity{}, err
	}
	return opportunity, nil
}

func (s *PilotPostgresStore) loadOpportunity(ctx context.Context, pilotID, opportunityID string) (PilotOpportunity, error) {
	var payload []byte
	if err := s.pool.QueryRow(ctx, `SELECT payload FROM exploratory_paper_opportunities WHERE pilot_id=$1 AND opportunity_id=$2`, pilotID, opportunityID).Scan(&payload); err != nil {
		return PilotOpportunity{}, err
	}
	var opportunity PilotOpportunity
	if err := json.Unmarshal(payload, &opportunity); err != nil {
		return PilotOpportunity{}, failClosed("decode opportunity projection", err)
	}
	return opportunity, nil
}

func (s *PilotPostgresStore) appendOpportunityEvent(ctx context.Context, opportunity PilotOpportunity, eventType, idempotency string, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `INSERT INTO exploratory_paper_opportunity_events(event_id,opportunity_id,event_type,idempotency_key,payload,created_at) VALUES($1,$2,$3,$4,$5,$6) ON CONFLICT(idempotency_key) DO NOTHING`, contentHash(opportunity.OpportunityID+"|"+idempotency), opportunity.OpportunityID, eventType, idempotency, data, s.now().UTC())
	return err
}

func (s *PilotPostgresStore) updateOpportunity(ctx context.Context, opportunity PilotOpportunity, eventType, idempotency string, eventPayload any) error {
	opportunity.UpdatedAt = s.now().UTC()
	payload, err := json.Marshal(opportunity)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `UPDATE exploratory_paper_opportunities SET payload=$3,final_classification=$4,updated_at=$5 WHERE pilot_id=$1 AND opportunity_id=$2 AND formal_evidence_eligible=FALSE`, opportunity.PilotID, opportunity.OpportunityID, payload, opportunity.FinalClassification, opportunity.UpdatedAt)
	if err != nil {
		return failClosed("update opportunity projection", err)
	}
	return s.appendOpportunityEvent(ctx, opportunity, eventType, idempotency, eventPayload)
}

func (s *PilotPostgresStore) RecordDecision(ctx context.Context, pilotID, opportunityID string, decision OpportunityDecision) (PilotOpportunity, error) {
	if _, err := s.admission(ctx, pilotID); err != nil {
		return PilotOpportunity{}, err
	}
	opportunity, err := s.loadOpportunity(ctx, pilotID, opportunityID)
	if err != nil {
		return PilotOpportunity{}, err
	}
	if decision.DecisionAt.IsZero() || decision.DecisionAt.Location() != time.UTC || decision.CandidateDecision == "" {
		return PilotOpportunity{}, fmt.Errorf("opportunity decision timestamp and classification are required")
	}
	opportunity.CandidateDecision = decision.CandidateDecision
	opportunity.CandidateDecisionAt = timePointer(decision.DecisionAt)
	opportunity.RiskDecision, opportunity.RiskDecisionAt = decision.RiskDecision, utcPointer(decision.RiskDecisionAt)
	opportunity.HumanDecision, opportunity.HumanReviewer, opportunity.HumanDecisionAt, opportunity.HumanRationale = decision.HumanDecision, decision.HumanReviewer, utcPointer(decision.HumanDecisionAt), decision.HumanRationale
	if decision.HumanDecision == "APPROVED" && decision.TradeLifecycleID != "" && !decision.PaperOnlyAcknowledged {
		return PilotOpportunity{}, fmt.Errorf("exploratory approval requires explicit PAPER-only acknowledgement")
	}
	opportunity.ThesisHash, opportunity.EvidenceSnapshotHash, opportunity.Direction = decision.ThesisHash, decision.EvidenceSnapshotHash, decision.Direction
	opportunity.ProtectiveStop, opportunity.Target, opportunity.HorizonSessions = decision.ProtectiveStop, decision.Target, decision.HorizonSessions
	opportunity.Confidence, opportunity.Uncertainty, opportunity.PaperOnlyAcknowledged = decision.Confidence, decision.Uncertainty, decision.PaperOnlyAcknowledged
	opportunity.CandidateID, opportunity.ThesisID, opportunity.TradeLifecycleID = decision.CandidateID, decision.ThesisID, decision.TradeLifecycleID
	opportunity.EntryEvidenceIDs, opportunity.EntryAt = append([]string(nil), decision.EntryEvidenceIDs...), utcPointer(decision.EntryAt)
	switch {
	case decision.CandidateDecision == DecisionWatch:
		opportunity.FinalClassification = PilotClassificationWatch
	case decision.CandidateDecision == DecisionNoTrade:
		opportunity.FinalClassification = PilotClassificationNoTrade
	case decision.RiskDecision == "REJECTED":
		opportunity.FinalClassification = PilotClassificationRejectedRisk
	case decision.EvidenceRejected:
		opportunity.FinalClassification = PilotClassificationRejectedEvidence
	case strings.EqualFold(decision.HumanDecision, "REJECTED"):
		opportunity.FinalClassification = PilotClassificationRejectedHuman
	case strings.EqualFold(decision.HumanDecision, "APPROVED") && decision.TradeLifecycleID != "":
		opportunity.FinalClassification = PilotClassificationApprovedExploratoryTrade
	default:
		opportunity.FinalClassification = PilotClassificationCandidate
	}
	if decision.Reason != "" && opportunity.EvidenceState == "" {
		opportunity.EvidenceState = decision.Reason
	}
	if err := s.updateOpportunity(ctx, opportunity, "DECISION", "decision:"+decision.DecisionAt.UTC().Format(time.RFC3339Nano), decision); err != nil {
		return PilotOpportunity{}, err
	}
	return opportunity, nil
}

func (s *PilotPostgresStore) LinkLifecycle(ctx context.Context, pilotID, opportunityID string, lifecycleID, candidateID, thesisID string, entryAt time.Time, entryEvidenceIDs []string) error {
	if _, err := s.admission(ctx, pilotID); err != nil {
		return err
	}
	opportunity, err := s.loadOpportunity(ctx, pilotID, opportunityID)
	if err != nil {
		return err
	}
	if lifecycleID == "" || candidateID == "" || thesisID == "" || entryAt.IsZero() || entryAt.Location() != time.UTC {
		return fmt.Errorf("PAPER-01 lifecycle link is incomplete")
	}
	if opportunity.FinalClassification != PilotClassificationCandidate && opportunity.FinalClassification != PilotClassificationApprovedExploratoryTrade {
		return fmt.Errorf("only a candidate can link to an exploratory lifecycle")
	}
	if strings.EqualFold(opportunity.HumanDecision, "APPROVED") && !opportunity.PaperOnlyAcknowledged {
		return fmt.Errorf("exploratory lifecycle link requires PAPER-only human acknowledgement")
	}
	opportunity.TradeLifecycleID, opportunity.CandidateID, opportunity.ThesisID = lifecycleID, candidateID, thesisID
	opportunity.EntryAt, opportunity.EntryEvidenceIDs = timePointer(entryAt), append([]string(nil), entryEvidenceIDs...)
	opportunity.FinalClassification = PilotClassificationApprovedExploratoryTrade
	return s.updateOpportunity(ctx, opportunity, "LIFECYCLE_LINK", "lifecycle:"+lifecycleID, map[string]any{"lifecycleId": lifecycleID, "candidateId": candidateID, "thesisId": thesisID, "entryAt": entryAt, "entryEvidenceIds": entryEvidenceIDs})
}

func (s *PilotPostgresStore) RecordEvidence(ctx context.Context, pilotID, opportunityID string, input EvidenceInput) (PilotEvidenceRecord, error) {
	if _, err := s.admission(ctx, pilotID); err != nil {
		return PilotEvidenceRecord{}, err
	}
	opportunity, err := s.loadOpportunity(ctx, pilotID, opportunityID)
	if err != nil {
		return PilotEvidenceRecord{}, err
	}
	phase, err := classifyPilotEvidence(opportunity, input)
	if err != nil {
		return PilotEvidenceRecord{}, err
	}
	record := PilotEvidenceRecord{PilotID: pilotID, OpportunityID: opportunityID, EvidenceID: input.EvidenceID, FirstSeenAt: input.FirstSeenAt, PublicationAt: utcPointer(input.PublicationAt), IngestionAt: input.IngestionAt, SourceID: input.SourceID, SourceURL: input.SourceURL, Relevance: input.Relevance, Signal: input.Signal, PolicyVersion: input.PolicyVersion, Phase: phase, Latency: buildEvidenceLatency(input.PublicationAt, input.IngestionAt)}
	payload, err := json.Marshal(record)
	if err != nil {
		return PilotEvidenceRecord{}, err
	}
	result, err := s.pool.Exec(ctx, `INSERT INTO exploratory_paper_pilot_evidence(pilot_id,opportunity_id,evidence_id,phase,first_seen_at,payload,formal_evidence_eligible) VALUES($1,$2,$3,$4,$5,$6,FALSE) ON CONFLICT(opportunity_id,evidence_id) DO NOTHING`, pilotID, opportunityID, record.EvidenceID, record.Phase, record.FirstSeenAt, payload)
	if err != nil {
		return PilotEvidenceRecord{}, failClosed("record pilot evidence", err)
	}
	var existing []byte
	if err := s.pool.QueryRow(ctx, `SELECT payload FROM exploratory_paper_pilot_evidence WHERE opportunity_id=$1 AND evidence_id=$2`, opportunityID, record.EvidenceID).Scan(&existing); err != nil {
		return PilotEvidenceRecord{}, err
	}
	if !sameJSON(existing, payload) {
		return PilotEvidenceRecord{}, ErrPilotIdentityConflict
	}
	if result.RowsAffected() == 0 {
		var prior PilotEvidenceRecord
		if err := json.Unmarshal(existing, &prior); err != nil {
			return PilotEvidenceRecord{}, failClosed("decode idempotent pilot evidence", err)
		}
		return prior, nil
	}
	if phase != EvidencePhaseEntry {
		opportunity.NewEvidenceCount++
	}
	if err := s.updateOpportunity(ctx, opportunity, "EVIDENCE", "evidence:"+record.EvidenceID, record); err != nil {
		return PilotEvidenceRecord{}, err
	}
	return record, nil
}

func buildEvidenceLatency(publication *time.Time, ingestion time.Time) LatencyValue {
	if publication == nil {
		return LatencyValue{Known: false, UnknownReason: "source publication timestamp unavailable"}
	}
	return knownLatency(publication.UTC(), ingestion.UTC())
}

func sameJSON(left, right []byte) bool {
	var leftValue, rightValue any
	if json.Unmarshal(left, &leftValue) != nil || json.Unmarshal(right, &rightValue) != nil {
		return string(left) == string(right)
	}
	return reflect.DeepEqual(leftValue, rightValue)
}

func (s *PilotPostgresStore) RecordMarketObservation(ctx context.Context, pilotID, opportunityID string, quality MarketObservationQuality) error {
	if _, err := s.admission(ctx, pilotID); err != nil {
		return err
	}
	if err := quality.Validate(); err != nil {
		return err
	}
	payload, err := json.Marshal(quality)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `INSERT INTO exploratory_paper_pilot_market_observations(observation_id,pilot_id,opportunity_id,kind,observed_at,payload) VALUES($1,$2,$3,$4,$5,$6) ON CONFLICT(observation_id) DO NOTHING`, quality.ObservationID, pilotID, opportunityID, quality.Kind, quality.ObservedAt, payload)
	return err
}

func (s *PilotPostgresStore) RecordOutcome(ctx context.Context, pilotID, opportunityID string, outcome PilotOutcomeSummary) error {
	if _, err := s.admission(ctx, pilotID); err != nil {
		return err
	}
	if outcome.OutcomeID == "" || outcome.CostModelVersion == "" || outcome.ExitReason == "" {
		return fmt.Errorf("pilot outcome summary is incomplete")
	}
	opportunity, err := s.loadOpportunity(ctx, pilotID, opportunityID)
	if err != nil {
		return err
	}
	if opportunity.TradeLifecycleID == "" || opportunity.FinalClassification != PilotClassificationApprovedExploratoryTrade {
		return fmt.Errorf("outcome requires an approved exploratory lifecycle")
	}
	opportunity.Outcome = &outcome
	return s.updateOpportunity(ctx, opportunity, "OUTCOME", "outcome:"+outcome.OutcomeID, outcome)
}

func (s *PilotPostgresStore) RecordIncident(ctx context.Context, incident PilotIncident) error {
	if err := incident.Validate(); err != nil {
		return err
	}
	payload, err := json.Marshal(incident)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `INSERT INTO exploratory_paper_pilot_incidents(incident_id,pilot_id,severity,code,payload,occurred_at,admission_blocked) VALUES($1,$2,$3,$4,$5,$6,$7) ON CONFLICT(incident_id) DO NOTHING`, incident.IncidentID, incident.PilotID, incident.Severity, incident.Code, payload, incident.OccurredAt, incident.AdmissionBlocked)
	if err != nil {
		return failClosed("record pilot incident", err)
	}
	if incident.AdmissionBlocked {
		record, getErr := s.GetPilot(ctx, incident.PilotID)
		if getErr != nil {
			return getErr
		}
		if record.Status == PilotStatusActive {
			nextStatus := PilotStatusAborted
			if strings.EqualFold(incident.Severity, "PAUSE") || strings.EqualFold(incident.Severity, "WARNING") {
				nextStatus = PilotStatusPaused
			}
			record.Status = nextStatus
			if err := s.transition(ctx, incident.PilotID, PilotStatusActive, nextStatus, record); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *PilotPostgresStore) ListOpportunities(ctx context.Context, pilotID string) ([]PilotOpportunity, error) {
	rows, err := s.pool.Query(ctx, `SELECT payload FROM exploratory_paper_opportunities WHERE pilot_id=$1 ORDER BY first_seen_at,opportunity_id`, pilotID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []PilotOpportunity
	for rows.Next() {
		var payload []byte
		if err := rows.Scan(&payload); err != nil {
			return nil, err
		}
		var opportunity PilotOpportunity
		if err := json.Unmarshal(payload, &opportunity); err != nil {
			return nil, failClosed("decode pilot opportunity", err)
		}
		result = append(result, opportunity)
	}
	return result, rows.Err()
}

func (s *PilotPostgresStore) ListIncidents(ctx context.Context, pilotID string) ([]PilotIncident, error) {
	rows, err := s.pool.Query(ctx, `SELECT payload FROM exploratory_paper_pilot_incidents WHERE pilot_id=$1 ORDER BY occurred_at,incident_id`, pilotID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []PilotIncident
	for rows.Next() {
		var payload []byte
		if err := rows.Scan(&payload); err != nil {
			return nil, err
		}
		var incident PilotIncident
		if err := json.Unmarshal(payload, &incident); err != nil {
			return nil, failClosed("decode pilot incident", err)
		}
		result = append(result, incident)
	}
	return result, rows.Err()
}

func (s *PilotPostgresStore) ReadModel(ctx context.Context, pilotID string) (PilotReadModel, error) {
	pilot, err := s.GetPilot(ctx, pilotID)
	if err != nil {
		return PilotReadModel{}, err
	}
	opportunities, err := s.ListOpportunities(ctx, pilotID)
	if err != nil {
		return PilotReadModel{}, err
	}
	incidents, err := s.ListIncidents(ctx, pilotID)
	if err != nil {
		return PilotReadModel{}, err
	}
	return PilotReadModel{Pilot: pilot, Opportunities: opportunities, Metrics: ComputePilotMetrics(opportunities), Incidents: incidents, Exploratory: true, NotFormalEvidence: true}, nil
}

func utcPointer(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	utc := value.UTC()
	return &utc
}

func timePointer(value time.Time) *time.Time {
	value = value.UTC()
	return &value
}

var _ PilotStore = (*PilotPostgresStore)(nil)
