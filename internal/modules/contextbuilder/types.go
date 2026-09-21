// Package contextbuilder implements the offline, read-only HARNESS-02
// retrieval and bounded context-assembly seam.
//
// It consumes canonical references and caller-supplied retrieval results. It
// does not own evidence or memory storage, call a model, invoke JaxMind, write
// task state, or import any trading runtime package.
package contextbuilder

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"jax-trading-assistant/internal/modules/harnesscontracts"
)

const (
	ContextBuilderContractV1  = "jax.harness.context_builder/v1"
	BuildReportContractV1     = "jax.harness.context_build_report/v1"
	RetrievalRecordContractV1 = "jax.harness.retrieval_record/v1"
)

const (
	SearchPerformed    = "PERFORMED"
	SearchNoResults    = "NO_RESULTS"
	SearchNotPerformed = "NOT_PERFORMED"
	SearchFailed       = "FAILED"

	EvidenceItemKind        = "EVIDENCE"
	CounterEvidenceItemKind = "COUNTER_EVIDENCE"
	MemoryItemKind          = "MEMORY"

	RetrievalSelected = "SELECTED"
	RetrievalRejected = "REJECTED"
	RetrievalDeferred = "DEFERRED"

	ReasonSelectedRequired        = "SELECTED_REQUIRED"
	ReasonSelectedHighRelevance   = "SELECTED_HIGH_RELEVANCE"
	ReasonSelectedCounterEvidence = "SELECTED_COUNTER_EVIDENCE"
	ReasonSelectedPolicyRequired  = "SELECTED_POLICY_REQUIRED"
	ReasonOmittedBudget           = "OMITTED_BUDGET"
	ReasonOmittedLowRelevance     = "OMITTED_LOW_RELEVANCE"
	ReasonOmittedDuplicate        = "OMITTED_DUPLICATE"
	ReasonOmittedStale            = "OMITTED_STALE"
	ReasonOmittedFuture           = "OMITTED_FUTURE"
	ReasonOmittedPolicy           = "OMITTED_POLICY"
	ReasonOmittedUnknownTiming    = "OMITTED_UNKNOWN_TIMING"
	ReasonDeferredOnDemand        = "DEFERRED_ON_DEMAND"
	ReasonUnknownMetadata         = "UNKNOWN_METADATA"

	TemporalFresh              = "FRESH"
	TemporalStale              = "STALE"
	TemporalFuture             = "FUTURE"
	TemporalPublicationUnknown = "PUBLICATION_TIME_UNKNOWN"
	TemporalFirstSeenUnknown   = "FIRST_SEEN_TIME_UNKNOWN"
	TemporalObservationUnknown = "OBSERVATION_TIME_UNKNOWN"

	RegimeMatch    = "MATCH"
	RegimeMismatch = "MISMATCH"
	RegimeUnknown  = "UNKNOWN"
)

var builderIdentifierPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:/-]{0,127}$`)

// RankingFactors preserve separate, auditable dimensions. RankScore is a
// deterministic ordering aid and is never a sufficiency or similarity claim.
type RankingFactors struct {
	ObjectiveRelevance int `json:"objective_relevance"`
	SubjectRelevance   int `json:"subject_relevance"`
	EventRelevance     int `json:"event_relevance"`
	CausalRelevance    int `json:"causal_relevance"`
	SourceQuality      int `json:"source_quality"`
	Recency            int `json:"recency"`
	Independence       int `json:"independence"`
	Corroboration      int `json:"corroboration"`
	StalenessPenalty   int `json:"staleness_penalty"`
	UncertaintyPenalty int `json:"uncertainty_penalty"`
}

func (f RankingFactors) Validate() error {
	values := []int{f.ObjectiveRelevance, f.SubjectRelevance, f.EventRelevance, f.CausalRelevance, f.SourceQuality, f.Recency, f.Independence, f.Corroboration, f.StalenessPenalty, f.UncertaintyPenalty}
	for _, value := range values {
		if value < 0 || value > 100 {
			return fmt.Errorf("ranking factors must be between 0 and 100")
		}
	}
	return nil
}

func (f RankingFactors) RankScore() int {
	// The weights deliberately retain multiple source, causal, quality, and
	// temporal dimensions. No single similarity value can dominate selection.
	return 30*f.ObjectiveRelevance + 18*f.SubjectRelevance + 14*f.EventRelevance +
		14*f.CausalRelevance + 10*f.SourceQuality + 6*f.Recency +
		5*f.Independence + 5*f.Corroboration - 8*f.StalenessPenalty - 8*f.UncertaintyPenalty
}

type EvidenceCandidate struct {
	Reference         harnesscontracts.EvidenceReference `json:"reference"`
	Factors           RankingFactors                     `json:"factors"`
	EstimatedUnits    int64                              `json:"estimated_units"`
	SubjectKey        string                             `json:"subject_key"`
	EventType         string                             `json:"event_type"`
	SourceGroup       string                             `json:"source_group"`
	DeduplicationKey  string                             `json:"deduplication_key"`
	OnDemandReference string                             `json:"on_demand_reference,omitempty"`
	PolicyEligible    bool                               `json:"policy_eligible"`
	PolicyReason      string                             `json:"policy_reason,omitempty"`
	PolicyRequired    bool                               `json:"policy_required"`
}

func (candidate EvidenceCandidate) Validate() error {
	if err := candidate.Reference.Validate(); err != nil {
		return err
	}
	switch candidate.Reference.Classification {
	case harnesscontracts.EvidenceCandidate, harnesscontracts.EvidenceSupporting, harnesscontracts.EvidenceContradictory, harnesscontracts.EvidenceInvalidating, harnesscontracts.EvidenceUnknownMissing:
	default:
		return fmt.Errorf("unsupported evidence classification %q", candidate.Reference.Classification)
	}
	if err := candidate.Factors.Validate(); err != nil {
		return err
	}
	if candidate.EstimatedUnits <= 0 || candidate.EstimatedUnits > 1_000_000 || !validOptionalBuilderReference(candidate.OnDemandReference) || !validOptionalText(candidate.SubjectKey, 256) || !validOptionalText(candidate.EventType, 256) || !validOptionalText(candidate.SourceGroup, 256) || !validOptionalText(candidate.DeduplicationKey, 512) || !validOptionalText(candidate.PolicyReason, 4096) {
		return fmt.Errorf("evidence candidate is incomplete or unbounded")
	}
	if containsSecretMaterial(candidate.OnDemandReference) || containsSecretMaterial(candidate.PolicyReason) {
		return fmt.Errorf("evidence candidate contains secret material")
	}
	return nil
}

type MemoryCandidate struct {
	Reference         harnesscontracts.MemoryReference `json:"reference"`
	Factors           RankingFactors                   `json:"factors"`
	EstimatedUnits    int64                            `json:"estimated_units"`
	EventType         string                           `json:"event_type"`
	SubjectKey        string                           `json:"subject_key"`
	RegimeStatus      string                           `json:"regime_status"`
	OnDemandReference string                           `json:"on_demand_reference,omitempty"`
	PolicyEligible    bool                             `json:"policy_eligible"`
	PolicyReason      string                           `json:"policy_reason,omitempty"`
}

func (candidate MemoryCandidate) Validate() error {
	if err := candidate.Reference.Validate(); err != nil {
		return err
	}
	if err := candidate.Factors.Validate(); err != nil {
		return err
	}
	if candidate.EstimatedUnits <= 0 || candidate.EstimatedUnits > 1_000_000 || !validOptionalBuilderReference(candidate.OnDemandReference) || !validOptionalText(candidate.EventType, 256) || !validOptionalText(candidate.SubjectKey, 256) || !validOptionalText(candidate.PolicyReason, 4096) {
		return fmt.Errorf("memory candidate is incomplete or unbounded")
	}
	switch candidate.RegimeStatus {
	case RegimeMatch, RegimeMismatch, RegimeUnknown:
	default:
		return fmt.Errorf("unsupported memory regime status %q", candidate.RegimeStatus)
	}
	if containsSecretMaterial(candidate.OnDemandReference) || containsSecretMaterial(candidate.PolicyReason) {
		return fmt.Errorf("memory candidate contains secret material")
	}
	return nil
}

type RetrievalRequest struct {
	Objective      harnesscontracts.ResearchObjective `json:"objective"`
	TaskState      harnesscontracts.TaskState         `json:"task_state"`
	PolicyVersions []harnesscontracts.PolicyReference `json:"policy_versions"`
	ReferenceTime  time.Time                          `json:"reference_time"`
}

func (request RetrievalRequest) Validate() error {
	if err := request.Objective.Validate(); err != nil {
		return err
	}
	if err := request.TaskState.Validate(); err != nil {
		return err
	}
	if request.TaskState.ObjectiveID != request.Objective.ObjectiveID || request.TaskState.ObjectiveVersion != request.Objective.Version || !validUTCTime(request.ReferenceTime) || !validPolicyReferences(request.PolicyVersions) {
		return fmt.Errorf("retrieval request identity or reference time is invalid")
	}
	return nil
}

type EvidenceRetrievalResult struct {
	RetrieverID      string              `json:"retriever_id"`
	RetrieverVersion string              `json:"retriever_version"`
	RetrievedAt      time.Time           `json:"retrieved_at"`
	SearchStatus     string              `json:"search_status"`
	Candidates       []EvidenceCandidate `json:"candidates"`
	Warnings         []string            `json:"warnings"`
}

func (result EvidenceRetrievalResult) Validate() error {
	if !validBuilderIdentifier(result.RetrieverID) || !validBuilderIdentifier(result.RetrieverVersion) || !validUTCTime(result.RetrievedAt) || !validSearchStatus(result.SearchStatus) || !validStringList(result.Warnings, 128, 4096) || len(result.Candidates) > 4096 {
		return fmt.Errorf("evidence retrieval result is malformed")
	}
	for _, candidate := range result.Candidates {
		if err := candidate.Validate(); err != nil {
			return err
		}
		if candidate.Reference.Classification != harnesscontracts.EvidenceCandidate && candidate.Reference.Classification != harnesscontracts.EvidenceSupporting {
			return fmt.Errorf("evidence retriever returned counter-evidence in supporting stream")
		}
	}
	return nil
}

type ContradictionRetrievalResult struct {
	RetrieverID      string              `json:"retriever_id"`
	RetrieverVersion string              `json:"retriever_version"`
	RetrievedAt      time.Time           `json:"retrieved_at"`
	SearchStatus     string              `json:"search_status"`
	Candidates       []EvidenceCandidate `json:"candidates"`
	KnownMissing     []string            `json:"known_missing"`
	Warnings         []string            `json:"warnings"`
}

func (result ContradictionRetrievalResult) Validate() error {
	if !validBuilderIdentifier(result.RetrieverID) || !validBuilderIdentifier(result.RetrieverVersion) || !validUTCTime(result.RetrievedAt) || !validSearchStatus(result.SearchStatus) || !validStringList(result.KnownMissing, 128, 4096) || !validStringList(result.Warnings, 128, 4096) || len(result.Candidates) > 4096 {
		return fmt.Errorf("contradiction retrieval result is malformed")
	}
	for _, candidate := range result.Candidates {
		if err := candidate.Validate(); err != nil {
			return fmt.Errorf("contradiction candidate: %w", err)
		}
		if candidate.Reference.Classification != harnesscontracts.EvidenceContradictory && candidate.Reference.Classification != harnesscontracts.EvidenceInvalidating && candidate.Reference.Classification != harnesscontracts.EvidenceUnknownMissing {
			return fmt.Errorf("contradiction retriever returned non-counter evidence")
		}
	}
	return nil
}

type MemoryRetrievalResult struct {
	RetrieverID      string            `json:"retriever_id"`
	RetrieverVersion string            `json:"retriever_version"`
	RetrievedAt      time.Time         `json:"retrieved_at"`
	SearchStatus     string            `json:"search_status"`
	Candidates       []MemoryCandidate `json:"candidates"`
	Warnings         []string          `json:"warnings"`
}

func (result MemoryRetrievalResult) Validate() error {
	if !validBuilderIdentifier(result.RetrieverID) || !validBuilderIdentifier(result.RetrieverVersion) || !validUTCTime(result.RetrievedAt) || !validSearchStatus(result.SearchStatus) || !validStringList(result.Warnings, 128, 4096) || len(result.Candidates) > 4096 {
		return fmt.Errorf("memory retrieval result is malformed")
	}
	for _, candidate := range result.Candidates {
		if err := candidate.Validate(); err != nil {
			return err
		}
	}
	return nil
}

type EvidenceRetriever interface {
	Retrieve(context.Context, RetrievalRequest) (EvidenceRetrievalResult, error)
}

type ContradictionRetriever interface {
	RetrieveContradictions(context.Context, RetrievalRequest) (ContradictionRetrievalResult, error)
}

type MemoryRetriever interface {
	RetrieveMemory(context.Context, RetrievalRequest) (MemoryRetrievalResult, error)
}

type BuildPolicy struct {
	BuilderVersion              string                             `json:"builder_version"`
	PolicyVersions              []harnesscontracts.PolicyReference `json:"policy_versions"`
	Model                       harnesscontracts.ModelReference    `json:"model"`
	PromptVersion               string                             `json:"prompt_version"`
	AssemblyAlgorithmVersion    string                             `json:"assembly_algorithm_version"`
	RequiredEvidenceIDs         []string                           `json:"required_evidence_ids"`
	RequireCounterEvidence      bool                               `json:"require_counter_evidence"`
	MinimumCounterEvidenceUnits int64                              `json:"minimum_counter_evidence_units"`
	ExcludeFutureEvidence       bool                               `json:"exclude_future_evidence"`
	ExcludeStaleEvidence        bool                               `json:"exclude_stale_evidence"`
	StaleAfter                  time.Duration                      `json:"stale_after"`
	RequireKnownObservationTime bool                               `json:"require_known_observation_time"`
	ExcludeRegimeMismatchMemory bool                               `json:"exclude_regime_mismatch_memory"`
	PreferIndependentSources    bool                               `json:"prefer_independent_sources"`
	AllowOnDemand               bool                               `json:"allow_on_demand"`
	MaxEvidenceItems            int                                `json:"max_evidence_items"`
	MaxCounterEvidenceItems     int                                `json:"max_counter_evidence_items"`
	MaxMemoryItems              int                                `json:"max_memory_items"`
	Assumptions                 []string                           `json:"assumptions"`
}

func DefaultBuildPolicy() BuildPolicy {
	return BuildPolicy{BuilderVersion: "contextbuilder-v1", PromptVersion: "research-prompt-v1", AssemblyAlgorithmVersion: "assembly-v1", RequireCounterEvidence: true, ExcludeFutureEvidence: true, ExcludeStaleEvidence: true, PreferIndependentSources: true, AllowOnDemand: true, MaxEvidenceItems: 512, MaxCounterEvidenceItems: 512, MaxMemoryItems: 512}
}

func (policy BuildPolicy) Validate(budget harnesscontracts.ContextBudget) error {
	if !validBuilderIdentifier(policy.BuilderVersion) || !validBuilderIdentifier(policy.PromptVersion) || !validBuilderIdentifier(policy.AssemblyAlgorithmVersion) || policy.Model.Validate() != nil || !validPolicyReferences(policy.PolicyVersions) || !validStringList(policy.RequiredEvidenceIDs, 512, 256) || !validStringList(policy.Assumptions, 128, 4096) || policy.MinimumCounterEvidenceUnits < 0 || policy.StaleAfter < 0 || policy.MaxEvidenceItems < 1 || policy.MaxCounterEvidenceItems < 1 || policy.MaxMemoryItems < 1 || !policy.ExcludeFutureEvidence {
		return fmt.Errorf("context build policy is incomplete or permits future evidence")
	}
	if policy.RequireCounterEvidence && (!budget.RequireCounterEvidence || budget.CounterEvidenceReserve <= 0) {
		return fmt.Errorf("counter-evidence policy requires a budget reserve")
	}
	return nil
}

type BuildRequest struct {
	Objective              harnesscontracts.ResearchObjective    `json:"objective"`
	TaskState              harnesscontracts.TaskState            `json:"task_state"`
	Budget                 harnesscontracts.ContextBudget        `json:"budget"`
	Policy                 BuildPolicy                           `json:"policy"`
	ReferenceTime          time.Time                             `json:"reference_time"`
	EvidenceRetriever      EvidenceRetriever                     `json:"-"`
	ContradictionRetriever ContradictionRetriever                `json:"-"`
	MemoryRetriever        MemoryRetriever                       `json:"-"`
	ToolReferences         []harnesscontracts.ToolReference      `json:"tool_references"`
	CheckpointReference    *harnesscontracts.CheckpointReference `json:"checkpoint_reference,omitempty"`
}

func (request BuildRequest) Validate() error {
	if err := request.Objective.Validate(); err != nil {
		return err
	}
	if err := request.TaskState.Validate(); err != nil {
		return err
	}
	if err := request.Budget.Validate(); err != nil {
		return err
	}
	if request.TaskState.ObjectiveID != request.Objective.ObjectiveID || request.TaskState.ObjectiveVersion != request.Objective.Version || !validUTCTime(request.ReferenceTime) || request.TaskState.CreatedAt.After(request.ReferenceTime) || request.TaskState.UpdatedAt.After(request.ReferenceTime) || request.EvidenceRetriever == nil || request.ContradictionRetriever == nil {
		return fmt.Errorf("context build request is incomplete or identity-mismatched")
	}
	if err := request.Policy.Validate(request.Budget); err != nil {
		return err
	}
	if len(request.ToolReferences) > 128 {
		return fmt.Errorf("too many tool references")
	}
	for _, reference := range request.ToolReferences {
		if err := reference.Validate(); err != nil {
			return err
		}
	}
	if request.CheckpointReference != nil {
		if err := request.CheckpointReference.Validate(); err != nil {
			return err
		}
		if request.CheckpointReference.TaskID != request.TaskState.TaskID || request.CheckpointReference.StateVersion > request.TaskState.StateVersion {
			return fmt.Errorf("checkpoint reference does not belong to build task state")
		}
	}
	return nil
}

type RetrievalRecord struct {
	ContractVersion   string                              `json:"contract_version"`
	RecordID          string                              `json:"record_id"`
	ItemKind          string                              `json:"item_kind"`
	EvidenceReference *harnesscontracts.EvidenceReference `json:"evidence_reference,omitempty"`
	MemoryReference   *harnesscontracts.MemoryReference   `json:"memory_reference,omitempty"`
	RetrieverID       string                              `json:"retriever_id"`
	RetrieverVersion  string                              `json:"retriever_version"`
	RetrievedAt       time.Time                           `json:"retrieved_at"`
	Factors           RankingFactors                      `json:"factors"`
	Classification    string                              `json:"classification"`
	Decision          string                              `json:"decision"`
	ReasonCode        string                              `json:"reason_code"`
	Reason            string                              `json:"reason"`
	PolicyEligible    bool                                `json:"policy_eligible"`
	EstimatedUnits    int64                               `json:"estimated_units"`
	BudgetImpact      int64                               `json:"budget_impact"`
	TemporalStatus    string                              `json:"temporal_status,omitempty"`
	DeferredReference string                              `json:"deferred_reference,omitempty"`
	RankScore         int                                 `json:"rank_score"`
}

func (record RetrievalRecord) Validate() error {
	if record.ContractVersion != RetrievalRecordContractV1 || !validBuilderIdentifier(record.RecordID) || !validBuilderIdentifier(record.RetrieverID) || !validBuilderIdentifier(record.RetrieverVersion) || !validUTCTime(record.RetrievedAt) || !validOptionalText(record.Classification, 256) || !validReasonCode(record.ReasonCode) || !boundedText(record.Reason, 4096) || record.EstimatedUnits <= 0 || record.BudgetImpact < 0 || !validOptionalBuilderReference(record.DeferredReference) || !validTemporalStatus(record.TemporalStatus) || !validDecision(record.Decision) || containsSecretMaterial(record.Reason) {
		return fmt.Errorf("retrieval record is malformed")
	}
	if err := record.Factors.Validate(); err != nil {
		return err
	}
	if (record.EvidenceReference == nil) == (record.MemoryReference == nil) {
		return fmt.Errorf("retrieval record must contain exactly one canonical reference")
	}
	if record.EvidenceReference != nil {
		if err := record.EvidenceReference.Validate(); err != nil {
			return err
		}
		if record.ItemKind == MemoryItemKind {
			return fmt.Errorf("memory item kind cannot carry evidence")
		}
	} else if err := record.MemoryReference.Validate(); err != nil {
		return err
	} else if record.ItemKind != MemoryItemKind {
		return fmt.Errorf("evidence item kind cannot carry memory")
	}
	return nil
}

type BudgetUsage struct {
	EvidenceUsed        int64 `json:"evidence_used"`
	CounterEvidenceUsed int64 `json:"counter_evidence_used"`
	MemoryUsed          int64 `json:"memory_used"`
}

type DeferredReference struct {
	Kind               string `json:"kind"`
	CanonicalReference string `json:"canonical_reference"`
	OnDemandReference  string `json:"on_demand_reference"`
	ReasonCode         string `json:"reason_code"`
	RetrieverID        string `json:"retriever_id"`
}

func (reference DeferredReference) Validate() error {
	if reference.Kind != EvidenceItemKind && reference.Kind != CounterEvidenceItemKind && reference.Kind != MemoryItemKind || !validBuilderReference(reference.CanonicalReference) || !validBuilderReference(reference.OnDemandReference) || !validReasonCode(reference.ReasonCode) || !validBuilderIdentifier(reference.RetrieverID) {
		return fmt.Errorf("deferred reference is malformed")
	}
	return nil
}

type Omission struct {
	Kind               string `json:"kind"`
	CanonicalReference string `json:"canonical_reference"`
	ReasonCode         string `json:"reason_code"`
	Reason             string `json:"reason"`
}

func (omission Omission) Validate() error {
	if omission.Kind != EvidenceItemKind && omission.Kind != CounterEvidenceItemKind && omission.Kind != MemoryItemKind || !validBuilderReference(omission.CanonicalReference) || !validReasonCode(omission.ReasonCode) || !boundedText(omission.Reason, 4096) || containsSecretMaterial(omission.Reason) {
		return fmt.Errorf("omission is malformed")
	}
	return nil
}

type BuildReport struct {
	ContractVersion                string                              `json:"contract_version"`
	BuilderVersion                 string                              `json:"builder_version"`
	Objective                      harnesscontracts.ObjectiveReference `json:"objective"`
	TaskState                      harnesscontracts.TaskStateReference `json:"task_state"`
	PolicyVersions                 []harnesscontracts.PolicyReference  `json:"policy_versions"`
	ReferenceTime                  time.Time                           `json:"reference_time"`
	ObjectiveInputHash             string                              `json:"objective_input_hash"`
	TaskStateInputHash             string                              `json:"task_state_input_hash"`
	BudgetInputHash                string                              `json:"budget_input_hash"`
	EvidenceCandidateCount         int                                 `json:"evidence_candidate_count"`
	CounterCandidateCount          int                                 `json:"counter_candidate_count"`
	MemoryCandidateCount           int                                 `json:"memory_candidate_count"`
	SelectedEvidenceCount          int                                 `json:"selected_evidence_count"`
	SelectedCounterCount           int                                 `json:"selected_counter_count"`
	SelectedMemoryCount            int                                 `json:"selected_memory_count"`
	RejectedCount                  int                                 `json:"rejected_count"`
	DeferredCount                  int                                 `json:"deferred_count"`
	BudgetRequested                harnesscontracts.ContextBudget      `json:"budget_requested"`
	BudgetUsed                     BudgetUsage                         `json:"budget_used"`
	EvidenceSearchStatus           string                              `json:"evidence_search_status"`
	CounterEvidenceSearchStatus    string                              `json:"counter_evidence_search_status"`
	CounterEvidenceSearchPerformed bool                                `json:"counter_evidence_search_performed"`
	RetrievalRecords               []RetrievalRecord                   `json:"retrieval_records"`
	Omissions                      []Omission                          `json:"omissions"`
	DeferredReferences             []DeferredReference                 `json:"deferred_references"`
	KnownMissing                   []string                            `json:"known_missing"`
	Warnings                       []string                            `json:"warnings"`
	FinalContextPackageID          string                              `json:"final_context_package_id"`
	FinalContextPackageHash        string                              `json:"final_context_package_hash"`
}

func (report BuildReport) Validate() error {
	if report.ContractVersion != BuildReportContractV1 || !validBuilderIdentifier(report.BuilderVersion) || report.Objective.Validate() != nil || report.TaskState.Validate() != nil || !validPolicyReferences(report.PolicyVersions) || !validUTCTime(report.ReferenceTime) || !harnesscontracts.IsCanonicalHash(report.ObjectiveInputHash) || !harnesscontracts.IsCanonicalHash(report.TaskStateInputHash) || !harnesscontracts.IsCanonicalHash(report.BudgetInputHash) || report.EvidenceCandidateCount < 0 || report.CounterCandidateCount < 0 || report.MemoryCandidateCount < 0 || report.SelectedEvidenceCount < 0 || report.SelectedCounterCount < 0 || report.SelectedMemoryCount < 0 || report.RejectedCount < 0 || report.DeferredCount < 0 || report.BudgetRequested.Validate() != nil || !validSearchStatus(report.EvidenceSearchStatus) || !validSearchStatus(report.CounterEvidenceSearchStatus) || !validStringList(report.KnownMissing, 512, 4096) || !validStringList(report.Warnings, 512, 4096) || !validIdentifier(report.FinalContextPackageID) || !harnesscontracts.IsCanonicalHash(report.FinalContextPackageHash) {
		return fmt.Errorf("build report is incomplete or malformed")
	}
	for _, record := range report.RetrievalRecords {
		if err := record.Validate(); err != nil {
			return err
		}
	}
	for _, omission := range report.Omissions {
		if err := omission.Validate(); err != nil {
			return err
		}
	}
	for _, deferred := range report.DeferredReferences {
		if err := deferred.Validate(); err != nil {
			return err
		}
	}
	return nil
}

type BuildResult struct {
	Package harnesscontracts.ContextPackage `json:"package"`
	Report  BuildReport                     `json:"report"`
}

func (result BuildResult) Validate() error {
	if err := result.Package.Validate(); err != nil {
		return err
	}
	if err := result.Report.Validate(); err != nil {
		return err
	}
	if result.Report.FinalContextPackageID != result.Package.ContextPackageID || result.Report.FinalContextPackageHash != result.Package.ContentHash {
		return fmt.Errorf("build report does not identify final context package")
	}
	return nil
}

func sortRetrievalRecords(records []RetrievalRecord) {
	sort.SliceStable(records, func(i, j int) bool {
		if records[i].ItemKind != records[j].ItemKind {
			return records[i].ItemKind < records[j].ItemKind
		}
		return records[i].RecordID < records[j].RecordID
	})
}

func validBuilderIdentifier(value string) bool {
	return builderIdentifierPattern.MatchString(value) && !containsSecretMaterial(value)
}
func validIdentifier(value string) bool       { return validBuilderIdentifier(value) }
func validBuilderReference(value string) bool { return validBuilderIdentifier(value) }
func validOptionalBuilderReference(value string) bool {
	return value == "" || validBuilderReference(value)
}
func validUTCTime(value time.Time) bool            { return !value.IsZero() && value.Location() == time.UTC }
func validOptionalText(value string, max int) bool { return value == "" || boundedText(value, max) }
func boundedText(value string, max int) bool {
	return strings.TrimSpace(value) != "" && len(value) <= max
}
func validStringList(values []string, maxItems, maxLength int) bool {
	if len(values) > maxItems {
		return false
	}
	for _, value := range values {
		if !boundedText(value, maxLength) {
			return false
		}
	}
	return true
}
func validPolicyReferences(values []harnesscontracts.PolicyReference) bool {
	if len(values) > 64 {
		return false
	}
	for _, value := range values {
		if value.Validate() != nil {
			return false
		}
	}
	return true
}
func validSearchStatus(value string) bool {
	return value == SearchPerformed || value == SearchNoResults || value == SearchNotPerformed || value == SearchFailed
}
func validTemporalStatus(value string) bool {
	return value == "" || value == TemporalFresh || value == TemporalStale || value == TemporalFuture || value == TemporalPublicationUnknown || value == TemporalFirstSeenUnknown || value == TemporalObservationUnknown
}
func validDecision(value string) bool {
	return value == RetrievalSelected || value == RetrievalRejected || value == RetrievalDeferred
}
func validReasonCode(value string) bool {
	switch value {
	case ReasonSelectedRequired, ReasonSelectedHighRelevance, ReasonSelectedCounterEvidence, ReasonSelectedPolicyRequired, ReasonOmittedBudget, ReasonOmittedLowRelevance, ReasonOmittedDuplicate, ReasonOmittedStale, ReasonOmittedFuture, ReasonOmittedPolicy, ReasonOmittedUnknownTiming, ReasonDeferredOnDemand, ReasonUnknownMetadata:
		return true
	default:
		return false
	}
}
func containsSecretMaterial(value string) bool {
	lower := strings.ToLower(value)
	for _, marker := range []string{"api_key", "apikey", "password", "authorization", "cookie", "database_url", "dsn=", "broker_credential", "client_secret", "refresh_token", "token="} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}
