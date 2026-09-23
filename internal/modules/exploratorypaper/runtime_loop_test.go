package exploratorypaper

import (
	"context"
	"testing"
	"time"

	"jax-trading-assistant/internal/modules/papertrading"
)

type runtimeEntrySource struct{ entries []EntryRequest }

func (s runtimeEntrySource) LoadApprovedEntries(context.Context) ([]EntryRequest, error) {
	return append([]EntryRequest(nil), s.entries...), nil
}

type runtimeReviewSource struct{}

func (runtimeReviewSource) LoadReviewObservation(context.Context, LifecycleRecord, Review) (ReviewObservation, error) {
	return ReviewObservation{}, ErrReviewInputsUnavailable
}

type availableReviewSource struct{ observation ReviewObservation }

func (s availableReviewSource) LoadReviewObservation(context.Context, LifecycleRecord, Review) (ReviewObservation, error) {
	return s.observation, nil
}

type runtimeStore struct {
	record            LifecycleRecord
	found             bool
	processed         int
	scheduled         int
	due               []ReviewRecord
	missing           int
	checkpoints       int
	recommended       int
	approved          int
	rejected          int
	checkpointIDs     map[int]bool
	recommendationIDs map[string]bool
	ledger            papertrading.PaperAccount
	outcome           *Outcome
}

func (s *runtimeStore) FindByCandidate(_ context.Context, candidateID string) (LifecycleRecord, bool, error) {
	if s.found && s.record.CandidateID == candidateID {
		return s.record, true, nil
	}
	return LifecycleRecord{}, false, nil
}

func (s *runtimeStore) SaveApprovedEntry(_ context.Context, candidateID string, thesis TradeThesis, binding EntryBinding, approval ApprovalSnapshot, position Position, entry EntrySnapshot) (string, error) {
	frozen, err := FreezeThesis(thesis, binding.BoundAt)
	if err != nil {
		return "", err
	}
	s.record = LifecycleRecord{LifecycleID: "runtime-lifecycle", CandidateID: candidateID, Thesis: frozen, Binding: binding, Position: position, Workflow: approval.Workflow, PaperIntent: approval.PaperIntent, EntryOrder: entry.Order, EntryFill: entry.Fill}
	s.ledger = entry.Ledger
	s.found = true
	return s.record.LifecycleID, nil
}

func (s *runtimeStore) PersistReviewSchedule(_ context.Context, _ string, schedule ReviewSchedule) error {
	s.scheduled = len(schedule.Sessions)
	return nil
}

func (s *runtimeStore) MarkEntryProcessed(context.Context, string) error { s.processed++; return nil }
func (s *runtimeStore) ListDueReviews(context.Context, time.Time) ([]ReviewRecord, error) {
	return append([]ReviewRecord(nil), s.due...), nil
}
func (s *runtimeStore) RecordReviewUnavailable(context.Context, string, int, time.Time, string) error {
	s.missing++
	return nil
}
func (s *runtimeStore) ApplyEvidence(context.Context, string, RelevantEvidence, time.Time, string) (Position, error) {
	return s.record.Position, nil
}
func (s *runtimeStore) PersistCheckpoint(context.Context, Checkpoint) error {
	if s.checkpointIDs == nil {
		s.checkpointIDs = map[int]bool{}
	}
	if !s.checkpointIDs[1] {
		s.checkpointIDs[1] = true
		s.checkpoints++
	}
	return nil
}

func (s *runtimeStore) PersistExitRecommendation(_ context.Context, _ string, _ ExitDecision, review Review) error {
	if s.recommendationIDs == nil {
		s.recommendationIDs = map[string]bool{}
	}
	if !s.recommendationIDs[review.ReviewID] {
		s.recommendationIDs[review.ReviewID] = true
		s.recommended++
	}
	return nil
}
func (s *runtimeStore) PersistExitApproval(context.Context, string, Review, ApprovalSnapshot) error {
	s.approved++
	return nil
}
func (s *runtimeStore) PersistExitRejection(context.Context, string, Review, ApprovalSnapshot) error {
	s.rejected++
	return nil
}
func (s *runtimeStore) PersistOutcome(context.Context, string, ExitSnapshot, Outcome) error {
	// The production store performs the durable close transaction. The fake
	// retains the same observable state for the orchestration proof.
	return nil

}

type recordingRuntimeStore struct{ *runtimeStore }

func (s *recordingRuntimeStore) ApplyEvidence(_ context.Context, _ string, evidence RelevantEvidence, at time.Time, policy string) (Position, error) {
	position := s.record.Position
	if err := position.Reassess(evidence, at, policy); err != nil {
		return Position{}, err
	}
	s.record.Position = position
	return position, nil
}

func (s *recordingRuntimeStore) PersistOutcome(_ context.Context, _ string, _ ExitSnapshot, outcome Outcome) error {
	s.outcome = &outcome
	s.record.Position.State = StateClosed
	s.record.Position.OperationalState = StateClosed
	return nil
}

type runtimeExitApprover struct{ approval ApprovalSnapshot }

func (a runtimeExitApprover) ApproveExit(context.Context, LifecycleRecord, ExitDecision, ReviewObservation) (ApprovalSnapshot, error) {
	return a.approval, nil
}

func TestRuntimeLoopUsesCanonicalEntryApprovalPaperVenueAndIdempotentRetry(t *testing.T) {
	now := time.Date(2026, 1, 2, 10, 0, 0, 0, time.UTC)
	thesis := fixtureThesis()
	risk := acceptedRisk(t, "recommendation-runtime", now, 1000)
	approvalWorkflow, approvalIntent := approvedIntent(t, "runtime", "LONG", now)
	input := reviewedCandidateInput(now)
	input.EventID, input.IssuerID, input.InstrumentID = thesis.EventID, thesis.IssuerID, thesis.InstrumentID
	input.EventCategory, input.EventTimestamp, input.GeneratedAt = thesis.EventCategory, thesis.EventTimestamp, thesis.CandidateGeneratedAt
	input.Evidence = thesis.Evidence
	input.ReviewedEvidence.EvidenceSetFingerprint = EvidenceSetFingerprint(input.Evidence)
	input.ReviewedEvidence.ReviewedAt = now
	input.SourceURL = thesis.Evidence[0].SourceURL
	input.CausalMechanism, input.QuantContext, input.RiskAssessment = thesis.ExpectedMechanism, thesis.QuantTechnicalContext, thesis.RiskAssessment
	input.TechnicalConfirmation = "confirmed supporting trend"
	input.Direction = thesis.Direction
	if candidateResult, candidateErr := GenerateCandidate(input); candidateErr != nil || candidateResult.Decision != DecisionCandidate {
		t.Fatalf("fixture candidate=%#v err=%v", candidateResult, candidateErr)
	}
	calendar := SessionCalendar{Sessions: map[string]bool{"2026-01-02": true, "2026-01-03": false, "2026-01-04": false, "2026-01-05": true, "2026-01-06": true, "2026-01-07": true, "2026-01-08": true}, Timezone: "UTC", OpenTime: "09:00", CloseTime: "17:00"}
	costs := papertrading.DefaultCostModel()
	venue, err := papertrading.NewPaperVenue(papertrading.DefaultPaperCapabilityContract(), costs)
	if err != nil {
		t.Fatal(err)
	}
	ledger, err := papertrading.NewPaperLedger("runtime-account", "USD", 10000)
	if err != nil {
		t.Fatal(err)
	}
	entry := EntryRequest{CandidateID: "candidate-1", Candidate: input, Evidence: *input.ReviewedEvidence, Thesis: thesis, RiskDecision: risk, Approval: ApprovalSnapshot{Workflow: approvalWorkflow, PaperIntent: approvalIntent}, PositionID: "runtime-position", Quantity: 10, Tick: papertrading.MarketTick{TickID: "runtime-entry-tick", InstrumentID: thesis.InstrumentID, Bid: 99, Ask: 101, Last: 100, AvailableQuantity: 10, Timestamp: now.Add(2 * time.Second), ReceivedAt: now.Add(2 * time.Second), Session: papertrading.SessionOpen, Source: "canonical-market"}, Venue: papertrading.DefaultPaperCapabilityContract(), CostModel: costs, Ledger: ledger.Snapshot(), Calendar: calendar, Now: now}
	store := &recordingRuntimeStore{runtimeStore: &runtimeStore{}}
	runtime := &Runtime{Mode: "PAPER", AccountID: "runtime-account", Store: store, Entries: runtimeEntrySource{entries: []EntryRequest{entry}}, Reviews: runtimeReviewSource{}, Venue: venue, CostModel: costs, Now: func() time.Time { return now }}
	mismatchedEntry := entry
	mismatchedEntry.CandidateID = "candidate-mismatched-account"
	mismatchedEntry.Ledger.AccountID = "other-account"
	mismatchedRuntime := *runtime
	mismatchedRuntime.Entries = runtimeEntrySource{entries: []EntryRequest{mismatchedEntry}}
	beforeMismatch := len(venue.Snapshot().Orders)
	if err := mismatchedRuntime.RunEntryCycle(context.Background()); err == nil {
		t.Fatal("entry with a different paper account was accepted")
	}
	if got := len(venue.Snapshot().Orders); got != beforeMismatch {
		t.Fatalf("mismatched entry mutated venue: before=%d after=%d", beforeMismatch, got)
	}
	if err := runtime.RunEntryCycle(context.Background()); err != nil {
		t.Fatalf("runtime entry: %v", err)
	}
	if !store.found || store.record.EntryFill.FillID == "" || store.record.Position.PositionID != entry.PositionID || store.scheduled != 5 || store.processed != 1 {
		t.Fatalf("runtime entry was not durably handed off: %#v", store)
	}
	if err := runtime.RunEntryCycle(context.Background()); err != nil {
		t.Fatal(err)
	}
	if store.processed != 2 {
		t.Fatalf("retry did not remain idempotent: processed=%d", store.processed)
	}
}

func TestRuntimeLoopRejectsNonPaperModeBeforeAnyVenueAction(t *testing.T) {
	runtime := &Runtime{Mode: "LIVE", Store: &runtimeStore{}, Entries: runtimeEntrySource{}, Reviews: runtimeReviewSource{}}
	if err := runtime.RunEntryCycle(context.Background()); err != ErrRuntimeMode {
		t.Fatalf("mode error=%v", err)
	}
}

func TestRuntimeLoopRejectsReviewForDifferentAccountBeforeVenueAction(t *testing.T) {
	now := time.Date(2026, 1, 2, 10, 0, 0, 0, time.UTC)
	venue, err := papertrading.NewPaperVenue(papertrading.DefaultPaperCapabilityContract(), papertrading.DefaultCostModel())
	if err != nil {
		t.Fatal(err)
	}
	runtime := &Runtime{
		Mode:      "PAPER",
		AccountID: "account-a",
		Store:     &runtimeStore{},
		Entries:   runtimeEntrySource{},
		Reviews: availableReviewSource{observation: ReviewObservation{
			Ledger: papertrading.PaperAccount{AccountID: "account-b"},
		}},
		Venue: venue,
		Now:   func() time.Time { return now },
	}
	before := len(venue.Snapshot().Orders)
	err = runtime.processReview(context.Background(), ReviewRecord{Review: Review{PositionID: "position-account-scope", SessionNumber: 1}})
	if err == nil {
		t.Fatal("review with a different paper account was accepted")
	}
	if after := len(venue.Snapshot().Orders); after != before {
		t.Fatalf("mismatched review mutated venue: before=%d after=%d", before, after)
	}
}

func TestRuntimeLoopProcessesDueReviewAndPreservesMissingData(t *testing.T) {
	now := time.Date(2026, 1, 2, 10, 0, 0, 0, time.UTC)
	thesis := fixtureThesis()
	risk := acceptedRisk(t, "recommendation-review", now, 1000)
	wf, intent := approvedIntent(t, "review", "LONG", now)
	input := reviewedCandidateInput(now)
	input.EventID, input.IssuerID, input.InstrumentID = thesis.EventID, thesis.IssuerID, thesis.InstrumentID
	input.EventCategory, input.EventTimestamp, input.GeneratedAt = thesis.EventCategory, thesis.EventTimestamp, thesis.CandidateGeneratedAt
	input.Evidence = thesis.Evidence
	input.ReviewedEvidence.EvidenceSetFingerprint = EvidenceSetFingerprint(input.Evidence)
	input.ReviewedEvidence.ReviewedAt = now
	input.SourceURL, input.CausalMechanism = thesis.Evidence[0].SourceURL, thesis.ExpectedMechanism
	input.QuantContext, input.RiskAssessment, input.TechnicalConfirmation, input.Direction = thesis.QuantTechnicalContext, thesis.RiskAssessment, "confirmed supporting trend", thesis.Direction
	calendar := SessionCalendar{Sessions: map[string]bool{"2026-01-02": true, "2026-01-03": false, "2026-01-04": false, "2026-01-05": true, "2026-01-06": true, "2026-01-07": true, "2026-01-08": true}, Timezone: "UTC", OpenTime: "09:00", CloseTime: "17:00"}
	costs := papertrading.DefaultCostModel()
	venue, err := papertrading.NewPaperVenue(papertrading.DefaultPaperCapabilityContract(), costs)
	if err != nil {
		t.Fatal(err)
	}
	ledger, err := papertrading.NewPaperLedger("review-account", "USD", 10000)
	if err != nil {
		t.Fatal(err)
	}
	entry := EntryRequest{CandidateID: "candidate-review", Candidate: input, Evidence: *input.ReviewedEvidence, Thesis: thesis, RiskDecision: risk, Approval: ApprovalSnapshot{Workflow: wf, PaperIntent: intent}, PositionID: "review-position", Quantity: 10, Tick: papertrading.MarketTick{TickID: "review-entry", InstrumentID: thesis.InstrumentID, Bid: 99, Ask: 101, Last: 100, AvailableQuantity: 10, Timestamp: now.Add(2 * time.Second), ReceivedAt: now.Add(2 * time.Second), Session: papertrading.SessionOpen, Source: "canonical-market"}, Venue: papertrading.DefaultPaperCapabilityContract(), CostModel: costs, Ledger: ledger.Snapshot(), Calendar: calendar, Now: now}
	store := &recordingRuntimeStore{runtimeStore: &runtimeStore{}}
	entryRuntime := &Runtime{Mode: "PAPER", AccountID: "review-account", Store: store, Entries: runtimeEntrySource{entries: []EntryRequest{entry}}, Reviews: runtimeReviewSource{}, Venue: venue, CostModel: costs, Now: func() time.Time { return now }}
	if err := entryRuntime.RunEntryCycle(context.Background()); err != nil {
		t.Fatal(err)
	}
	store.due = []ReviewRecord{{Lifecycle: store.record, Review: Review{ReviewID: "review-1", PositionID: entry.PositionID, SessionNumber: 1, ScheduledAt: now.Add(time.Hour), Status: "PENDING"}}}
	exitTime := now.Add(2 * time.Hour)
	exitWorkflow, exitIntent := approvedIntent(t, "review-exit", "SHORT", exitTime)
	observation := ReviewObservation{Tick: papertrading.MarketTick{TickID: "review-tick", InstrumentID: thesis.InstrumentID, Bid: 94, Ask: 95, Last: 94, AvailableQuantity: 10, Timestamp: exitTime, ReceivedAt: exitTime, Session: papertrading.SessionOpen, Source: "canonical-market"}, Price: 94, PriceSource: "canonical-market", Evidence: []RelevantEvidence{{Reference: EvidenceReference{EvidenceID: "runtime-invalidation", SourceID: "issuer-source", SourceURL: "https://example.test/invalidation", Quality: "high", ObservedAt: exitTime}, IssuerID: thesis.IssuerID, InstrumentID: thesis.InstrumentID, Signal: EvidenceInvalidates, Reason: "issuer correction invalidates the mechanism"}}, Calendar: calendar, Ledger: store.ledger, Path: []PriceObservation{{ObservationID: "review-path", At: exitTime, Price: 94, Source: "canonical-market"}}, ThesisInvalidated: true}
	runtime := &Runtime{Mode: "PAPER", AccountID: "review-account", Store: store, Entries: runtimeEntrySource{}, Reviews: availableReviewSource{observation: observation}, ExitApprover: runtimeExitApprover{approval: ApprovalSnapshot{Workflow: exitWorkflow, PaperIntent: exitIntent}}, Venue: venue, CostModel: costs, Now: func() time.Time { return now.Add(3 * time.Hour) }}
	noApprovalRuntime := *runtime
	noApprovalRuntime.ExitApprover = nil
	if err := noApprovalRuntime.RunDueReviews(context.Background()); err != nil {
		t.Fatalf("recommendation without approver returned unexpected error: %v", err)
	}
	if store.recommended != 1 || store.outcome != nil {
		t.Fatalf("exit recommendation was not durable before approval: recommended=%d outcome=%#v", store.recommended, store.outcome)
	}
	if err := runtime.RunDueReviews(context.Background()); err != nil {
		t.Fatal(err)
	}
	if store.checkpoints != 1 || store.missing != 0 || store.recommended != 1 || store.approved != 1 || store.outcome == nil || store.outcome.ExitReason != ExitThesisInvalidated || store.outcome.ExcursionStatus != "INCOMPLETE" || store.outcome.ExcursionKnown {
		t.Fatalf("due review was not processed through approved exit: checkpoints=%d missing=%d outcome=%#v", store.checkpoints, store.missing, store.outcome)
	}
	store.due = []ReviewRecord{{Lifecycle: store.record, Review: Review{ReviewID: "review-2", PositionID: entry.PositionID, SessionNumber: 2, ScheduledAt: now.Add(time.Hour), Status: "PENDING"}}}
	runtime.Reviews = runtimeReviewSource{}
	if err := runtime.RunDueReviews(context.Background()); err != nil {
		t.Fatal(err)
	}
	if store.missing != 1 {
		t.Fatalf("missing review inputs were not retained: missing=%d", store.missing)
	}
}
