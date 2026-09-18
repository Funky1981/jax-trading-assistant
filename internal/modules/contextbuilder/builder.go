package contextbuilder

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"jax-trading-assistant/internal/modules/harnesscontracts"
)

// Build constructs one bounded, reference-only ContextPackage. It performs no
// model call, does not claim sufficiency, and does not write any repository or
// trading state.
func Build(ctx context.Context, request BuildRequest) (BuildResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if len(request.Policy.PolicyVersions) == 0 {
		request.Policy.PolicyVersions = append([]harnesscontracts.PolicyReference(nil), request.Objective.PolicyVersions...)
	}
	if err := request.Validate(); err != nil {
		return BuildResult{}, err
	}

	retrievalRequest := RetrievalRequest{
		Objective:      request.Objective,
		TaskState:      request.TaskState,
		PolicyVersions: append([]harnesscontracts.PolicyReference(nil), request.Policy.PolicyVersions...),
		ReferenceTime:  request.ReferenceTime,
	}
	if err := retrievalRequest.Validate(); err != nil {
		return BuildResult{}, err
	}

	evidenceResult, err := request.EvidenceRetriever.Retrieve(ctx, retrievalRequest)
	if err != nil {
		return BuildResult{}, fmt.Errorf("evidence retrieval failed: %w", err)
	}
	if err := evidenceResult.Validate(); err != nil {
		return BuildResult{}, fmt.Errorf("evidence retrieval result: %w", err)
	}
	if evidenceResult.SearchStatus == SearchFailed {
		return BuildResult{}, fmt.Errorf("evidence retrieval reported failure")
	}

	counterResult, err := request.ContradictionRetriever.RetrieveContradictions(ctx, retrievalRequest)
	if err != nil {
		return BuildResult{}, fmt.Errorf("counter-evidence retrieval failed: %w", err)
	}
	if err := counterResult.Validate(); err != nil {
		return BuildResult{}, fmt.Errorf("counter-evidence retrieval result: %w", err)
	}
	if counterResult.SearchStatus == SearchFailed {
		return BuildResult{}, fmt.Errorf("counter-evidence retrieval reported failure")
	}

	memoryResult := MemoryRetrievalResult{
		RetrieverID:      "memory-not-requested",
		RetrieverVersion: "v1",
		RetrievedAt:      request.ReferenceTime,
		SearchStatus:     SearchNotPerformed,
	}
	if request.MemoryRetriever != nil {
		memoryResult, err = request.MemoryRetriever.RetrieveMemory(ctx, retrievalRequest)
		if err != nil {
			return BuildResult{}, fmt.Errorf("memory retrieval failed: %w", err)
		}
		if err := memoryResult.Validate(); err != nil {
			return BuildResult{}, fmt.Errorf("memory retrieval result: %w", err)
		}
		if memoryResult.SearchStatus == SearchFailed {
			return BuildResult{}, fmt.Errorf("memory retrieval reported failure")
		}
	}

	report := newBuildReport(request, evidenceResult, counterResult, memoryResult)
	records := make(map[string]RetrievalRecord)
	evidenceSelections := make(map[string]EvidenceCandidate)
	counterSelections := make(map[string]EvidenceCandidate)
	memorySelections := make(map[string]MemoryCandidate)

	evidenceCandidates := make([]rankedEvidence, 0, len(evidenceResult.Candidates))
	for _, candidate := range evidenceResult.Candidates {
		record := newEvidenceRecord(candidate, EvidenceItemKind, evidenceResult.RetrieverID, evidenceResult.RetrieverVersion, evidenceResult.RetrievedAt)
		records[record.RecordID] = record
		evidenceCandidates = append(evidenceCandidates, rankedEvidence{candidate: candidate, recordID: record.RecordID})
	}
	counterCandidates := make([]rankedEvidence, 0, len(counterResult.Candidates))
	for _, candidate := range counterResult.Candidates {
		record := newEvidenceRecord(candidate, CounterEvidenceItemKind, counterResult.RetrieverID, counterResult.RetrieverVersion, counterResult.RetrievedAt)
		records[record.RecordID] = record
		counterCandidates = append(counterCandidates, rankedEvidence{candidate: candidate, recordID: record.RecordID})
	}
	memoryCandidates := make([]rankedMemory, 0, len(memoryResult.Candidates))
	for _, candidate := range memoryResult.Candidates {
		record := newMemoryRecord(candidate, memoryResult.RetrieverID, memoryResult.RetrieverVersion, memoryResult.RetrievedAt)
		records[record.RecordID] = record
		memoryCandidates = append(memoryCandidates, rankedMemory{candidate: candidate, recordID: record.RecordID})
	}

	evidenceSelected, evidenceOmissions, evidenceDeferred, err := selectEvidence(evidenceCandidates, request, records, evidenceSelections)
	if err != nil {
		return BuildResult{}, err
	}
	counterSelected, counterOmissions, counterDeferred, err := selectCounterEvidence(counterCandidates, request, records, counterSelections)
	if err != nil {
		return BuildResult{}, err
	}
	memorySelected, memoryOmissions, memoryDeferred := selectMemory(memoryCandidates, request, records, memorySelections)

	for _, warning := range evidenceResult.Warnings {
		report.Warnings = append(report.Warnings, warning)
	}
	for _, warning := range counterResult.Warnings {
		report.Warnings = append(report.Warnings, warning)
	}
	for _, warning := range memoryResult.Warnings {
		report.Warnings = append(report.Warnings, warning)
	}
	report.KnownMissing = append(report.KnownMissing, counterResult.KnownMissing...)
	report.Omissions = append(report.Omissions, evidenceOmissions...)
	report.Omissions = append(report.Omissions, counterOmissions...)
	report.Omissions = append(report.Omissions, memoryOmissions...)
	report.DeferredReferences = append(report.DeferredReferences, evidenceDeferred...)
	report.DeferredReferences = append(report.DeferredReferences, counterDeferred...)
	report.DeferredReferences = append(report.DeferredReferences, memoryDeferred...)

	if counterResult.SearchStatus == SearchNotPerformed {
		report.Warnings = append(report.Warnings, "COUNTER_EVIDENCE_NOT_SEARCHED")
	} else if counterResult.SearchStatus == SearchNoResults {
		report.Warnings = append(report.Warnings, "NO_COUNTER_EVIDENCE_FOUND")
	}
	if evidenceResult.SearchStatus == SearchNoResults {
		report.Warnings = append(report.Warnings, "NO_SUPPORTING_EVIDENCE_FOUND")
	}
	if memoryResult.SearchStatus == SearchNotPerformed {
		report.Warnings = append(report.Warnings, "MEMORY_NOT_SEARCHED")
	}

	selectedEvidence := referencesFromEvidence(evidenceSelected)
	selectedCounter := referencesFromEvidence(counterSelected)
	selectedMemory := referencesFromMemory(memorySelected)
	constraints := uniqueSorted(append(append([]string(nil), request.Objective.Constraints...), request.TaskState.Constraints...))
	openQuestions := uniqueSorted(request.TaskState.OpenQuestions)
	knownUnknowns := uniqueSorted(append(append([]string(nil), request.TaskState.KnownUnknowns...), report.KnownMissing...))
	selectionReasons := selectionReasonList(records)

	packageValue := harnesscontracts.ContextPackage{
		ContractVersion:     harnesscontracts.ContextPackageContractV1,
		ContextPackageID:    "context-pending",
		TaskID:              request.TaskState.TaskID,
		TaskStateVersion:    request.TaskState.StateVersion,
		ObjectiveID:         request.Objective.ObjectiveID,
		ObjectiveVersion:    request.Objective.Version,
		CreatedAt:           request.ReferenceTime,
		ModelPurpose:        buildModelPurpose(request.Objective),
		TokenBudget:         request.Budget,
		Constraints:         constraints,
		PolicyVersions:      append([]harnesscontracts.PolicyReference(nil), request.Policy.PolicyVersions...),
		SelectedEvidence:    selectedEvidence,
		CounterEvidence:     selectedCounter,
		SelectedMemory:      selectedMemory,
		CurrentState:        request.TaskState,
		OpenQuestions:       openQuestions,
		KnownUnknowns:       knownUnknowns,
		Assumptions:         append([]string(nil), request.Policy.Assumptions...),
		ToolReferences:      append([]harnesscontracts.ToolReference(nil), request.ToolReferences...),
		CheckpointReference: request.CheckpointReference,
		Provenance: harnesscontracts.ContextProvenance{
			ContractVersion:          harnesscontracts.ContextProvenanceContractV1,
			Objective:                harnesscontracts.ObjectiveReference{ObjectiveID: request.Objective.ObjectiveID, ObjectiveVersion: request.Objective.Version},
			TaskState:                harnesscontracts.TaskStateReference{TaskID: request.TaskState.TaskID, StateVersion: request.TaskState.StateVersion},
			ContextPackageID:         "context-pending",
			SelectedEvidenceIDs:      evidenceIDs(selectedEvidence),
			CounterEvidenceIDs:       evidenceIDs(selectedCounter),
			SelectedMemoryIDs:        memoryIDs(selectedMemory),
			ToolInvocationIDs:        toolIDs(request.ToolReferences),
			Model:                    request.Policy.Model,
			PromptVersion:            request.Policy.PromptVersion,
			PolicyVersions:           append([]harnesscontracts.PolicyReference(nil), request.Policy.PolicyVersions...),
			AssemblyAlgorithmVersion: request.Policy.AssemblyAlgorithmVersion,
			AssembledAt:              request.ReferenceTime,
		},
		Audit: harnesscontracts.ContextPackageAudit{
			BuilderVersion:      request.Policy.BuilderVersion,
			OmittedEvidenceIDs:  omissionIDs(report.Omissions, EvidenceItemKind),
			DeferredEvidenceIDs: deferredIDs(report.DeferredReferences, EvidenceItemKind),
			OmittedMemoryIDs:    omissionIDs(report.Omissions, MemoryItemKind),
			SelectionReasons:    selectionReasons,
			AssembledAt:         request.ReferenceTime,
		},
	}
	packageValue, err = harnesscontracts.NewContextPackage(packageValue)
	if err != nil {
		return BuildResult{}, fmt.Errorf("construct context package: %w", err)
	}
	packageValue.ContextPackageID = "context-" + packageValue.ContentHash[len("sha256:"):len("sha256:")+24]
	packageValue.Provenance.ContextPackageID = packageValue.ContextPackageID
	packageValue, err = harnesscontracts.NewContextPackage(packageValue)
	if err != nil {
		return BuildResult{}, fmt.Errorf("finalize context package: %w", err)
	}

	report.SelectedEvidenceCount = len(selectedEvidence)
	report.SelectedCounterCount = len(selectedCounter)
	report.SelectedMemoryCount = len(selectedMemory)
	report.BudgetUsed = BudgetUsage{EvidenceUsed: sumEvidenceUnits(evidenceSelected), CounterEvidenceUsed: sumEvidenceUnits(counterSelected), MemoryUsed: sumMemoryUnits(memorySelected)}
	report.FinalContextPackageID = packageValue.ContextPackageID
	report.FinalContextPackageHash = packageValue.ContentHash
	report.RetrievalRecords = mapToSortedRecords(records)
	report.RejectedCount, report.DeferredCount = countDecisions(report.RetrievalRecords)
	if err := report.Validate(); err != nil {
		return BuildResult{}, fmt.Errorf("build report: %w", err)
	}
	result := BuildResult{Package: packageValue, Report: report}
	if err := result.Validate(); err != nil {
		return BuildResult{}, err
	}
	return result, nil
}

// BuildContext is the descriptive API name retained for callers that prefer
// the architecture terminology.
func BuildContext(ctx context.Context, request BuildRequest) (BuildResult, error) {
	return Build(ctx, request)
}

type rankedEvidence struct {
	candidate EvidenceCandidate
	recordID  string
}

type rankedMemory struct {
	candidate MemoryCandidate
	recordID  string
}

func newBuildReport(request BuildRequest, evidence EvidenceRetrievalResult, counter ContradictionRetrievalResult, memory MemoryRetrievalResult) BuildReport {
	objectiveHash, _ := harnesscontracts.CanonicalHash(request.Objective)
	taskHash, _ := harnesscontracts.CanonicalHash(request.TaskState)
	budgetHash, _ := harnesscontracts.CanonicalHash(request.Budget)
	return BuildReport{
		ContractVersion:                BuildReportContractV1,
		BuilderVersion:                 request.Policy.BuilderVersion,
		Objective:                      harnesscontracts.ObjectiveReference{ObjectiveID: request.Objective.ObjectiveID, ObjectiveVersion: request.Objective.Version},
		TaskState:                      harnesscontracts.TaskStateReference{TaskID: request.TaskState.TaskID, StateVersion: request.TaskState.StateVersion},
		PolicyVersions:                 append([]harnesscontracts.PolicyReference(nil), request.Policy.PolicyVersions...),
		ReferenceTime:                  request.ReferenceTime,
		ObjectiveInputHash:             objectiveHash,
		TaskStateInputHash:             taskHash,
		BudgetInputHash:                budgetHash,
		EvidenceCandidateCount:         len(evidence.Candidates),
		CounterCandidateCount:          len(counter.Candidates),
		MemoryCandidateCount:           len(memory.Candidates),
		BudgetRequested:                request.Budget,
		EvidenceSearchStatus:           evidence.SearchStatus,
		CounterEvidenceSearchStatus:    counter.SearchStatus,
		CounterEvidenceSearchPerformed: counter.SearchStatus == SearchPerformed || counter.SearchStatus == SearchNoResults,
	}
}

func selectEvidence(candidates []rankedEvidence, request BuildRequest, records map[string]RetrievalRecord, selected map[string]EvidenceCandidate) ([]rankedEvidence, []Omission, []DeferredReference, error) {
	return selectEvidenceLike(candidates, request, records, selected, false)
}

func selectCounterEvidence(candidates []rankedEvidence, request BuildRequest, records map[string]RetrievalRecord, selected map[string]EvidenceCandidate) ([]rankedEvidence, []Omission, []DeferredReference, error) {
	if request.Policy.RequireCounterEvidence && request.Policy.MinimumCounterEvidenceUnits == 0 {
		request.Policy.MinimumCounterEvidenceUnits = request.Budget.CounterEvidenceReserve
	}
	chosen, omissions, deferred, err := selectEvidenceLike(candidates, request, records, selected, true)
	if err != nil {
		return nil, omissions, deferred, err
	}
	used := sumEvidenceUnits(chosen)
	if request.Policy.RequireCounterEvidence && (request.Policy.MinimumCounterEvidenceUnits <= 0 || used < request.Policy.MinimumCounterEvidenceUnits) {
		return nil, omissions, deferred, fmt.Errorf("mandatory counter-evidence reserve could not be satisfied: used=%d required=%d", used, request.Policy.MinimumCounterEvidenceUnits)
	}
	return chosen, omissions, deferred, nil
}

func selectEvidenceLike(candidates []rankedEvidence, request BuildRequest, records map[string]RetrievalRecord, selected map[string]EvidenceCandidate, counter bool) ([]rankedEvidence, []Omission, []DeferredReference, error) {
	budget := request.Budget.Evidence
	maxItems := request.Policy.MaxEvidenceItems
	requiredIDs := make(map[string]struct{}, len(request.Policy.RequiredEvidenceIDs))
	for _, id := range request.Policy.RequiredEvidenceIDs {
		requiredIDs[id] = struct{}{}
	}
	if counter {
		budget = request.Budget.CounterEvidence
		maxItems = request.Policy.MaxCounterEvidenceItems
		requiredIDs = map[string]struct{}{}
	}
	for index := range candidates {
		if _, required := requiredIDs[candidates[index].candidate.Reference.EvidenceID]; required {
			candidates[index].candidate.PolicyRequired = true
		}
	}
	ordered, duplicateOmissions := uniqueAndRankEvidence(candidates, records)
	omissions := append([]Omission(nil), duplicateOmissions...)
	deferred := []DeferredReference{}
	chosen := []rankedEvidence{}
	used := int64(0)
	seenSource := map[string]struct{}{}
	selectedRequired := map[string]struct{}{}

	sort.SliceStable(ordered, func(i, j int) bool {
		leftRequired := ordered[i].candidate.PolicyRequired
		rightRequired := ordered[j].candidate.PolicyRequired
		if leftRequired != rightRequired {
			return leftRequired
		}
		return evidenceLess(ordered[i], ordered[j])
	})

	for pass := 0; pass < 2; pass++ {
		for _, candidate := range ordered {
			if _, already := selected[candidate.candidate.Reference.EvidenceID]; already {
				continue
			}
			if pass == 0 && request.Policy.PreferIndependentSources && sourceKey(candidate.candidate) != "" {
				if _, exists := seenSource[sourceKey(candidate.candidate)]; exists && !candidate.candidate.PolicyRequired {
					continue
				}
			}
			if len(chosen) >= maxItems || used+candidate.candidate.EstimatedUnits > budget {
				continue
			}
			record := records[candidate.recordID]
			if !eligibleEvidence(candidate.candidate, request.Policy, request.ReferenceTime, counter, &record, &omissions, &deferred) {
				records[candidate.recordID] = record
				continue
			}
			records[candidate.recordID] = record
			chosen = append(chosen, candidate)
			selected[candidate.candidate.Reference.EvidenceID] = candidate.candidate
			used += candidate.candidate.EstimatedUnits
			seenSource[sourceKey(candidate.candidate)] = struct{}{}
			selectedRequired[candidate.candidate.Reference.EvidenceID] = struct{}{}
			markSelected(records, candidate.recordID, counter, candidate.candidate, request.ReferenceTime)
		}
	}

	for _, candidate := range ordered {
		id := candidate.candidate.Reference.EvidenceID
		if _, exists := selectedRequired[id]; exists {
			continue
		}
		if _, required := requiredIDs[id]; required {
			return chosen, omissions, deferred, fmt.Errorf("required evidence %q could not fit the evidence budget", id)
		}
		if records[candidate.recordID].Decision == RetrievalRejected && records[candidate.recordID].ReasonCode == "" {
			if request.Policy.AllowOnDemand && candidate.candidate.OnDemandReference != "" {
				markDeferred(records, candidate.recordID, ReasonDeferredOnDemand, candidate.candidate.OnDemandReference)
				deferred = append(deferred, DeferredReference{Kind: itemKind(records[candidate.recordID]), CanonicalReference: id, OnDemandReference: candidate.candidate.OnDemandReference, ReasonCode: ReasonDeferredOnDemand, RetrieverID: records[candidate.recordID].RetrieverID})
			} else {
				markRejected(records, candidate.recordID, ReasonOmittedBudget, "candidate did not fit the reserved evidence budget")
				omissions = append(omissions, Omission{Kind: itemKind(records[candidate.recordID]), CanonicalReference: id, ReasonCode: ReasonOmittedBudget, Reason: "candidate did not fit the reserved evidence budget"})
			}
		}
	}
	for requiredID := range requiredIDs {
		if _, exists := selected[requiredID]; !exists {
			return chosen, omissions, deferred, fmt.Errorf("required evidence %q was not returned by the retriever", requiredID)
		}
	}
	return chosen, omissions, deferred, nil
}

func selectMemory(candidates []rankedMemory, request BuildRequest, records map[string]RetrievalRecord, selected map[string]MemoryCandidate) ([]rankedMemory, []Omission, []DeferredReference) {
	ordered, omissions := uniqueAndRankMemory(candidates, records)
	deferred := []DeferredReference{}
	chosen := []rankedMemory{}
	used := int64(0)
	for _, candidate := range ordered {
		id := candidate.candidate.Reference.MemoryID
		if !candidate.candidate.PolicyEligible {
			markRejected(records, candidate.recordID, ReasonOmittedPolicy, candidate.candidate.PolicyReason)
			omissions = append(omissions, Omission{Kind: MemoryItemKind, CanonicalReference: id, ReasonCode: ReasonOmittedPolicy, Reason: nonEmptyReason(candidate.candidate.PolicyReason, "memory is not policy eligible")})
			continue
		}
		if request.Policy.ExcludeRegimeMismatchMemory && candidate.candidate.RegimeStatus == RegimeMismatch {
			markRejected(records, candidate.recordID, ReasonOmittedPolicy, "memory regime mismatch")
			omissions = append(omissions, Omission{Kind: MemoryItemKind, CanonicalReference: id, ReasonCode: ReasonOmittedPolicy, Reason: "memory regime mismatch"})
			continue
		}
		if len(chosen) >= request.Policy.MaxMemoryItems || used+candidate.candidate.EstimatedUnits > request.Budget.Memory {
			if request.Policy.AllowOnDemand && candidate.candidate.OnDemandReference != "" {
				markDeferred(records, candidate.recordID, ReasonDeferredOnDemand, candidate.candidate.OnDemandReference)
				deferred = append(deferred, DeferredReference{Kind: MemoryItemKind, CanonicalReference: id, OnDemandReference: candidate.candidate.OnDemandReference, ReasonCode: ReasonDeferredOnDemand, RetrieverID: records[candidate.recordID].RetrieverID})
			} else {
				markRejected(records, candidate.recordID, ReasonOmittedBudget, "memory did not fit the memory budget")
				omissions = append(omissions, Omission{Kind: MemoryItemKind, CanonicalReference: id, ReasonCode: ReasonOmittedBudget, Reason: "memory did not fit the memory budget"})
			}
			continue
		}
		chosen = append(chosen, candidate)
		selected[id] = candidate.candidate
		used += candidate.candidate.EstimatedUnits
		markSelectedMemory(records, candidate.recordID, candidate.candidate)
	}
	return chosen, omissions, deferred
}

func eligibleEvidence(candidate EvidenceCandidate, policy BuildPolicy, referenceTime time.Time, counter bool, record *RetrievalRecord, omissions *[]Omission, deferred *[]DeferredReference) bool {
	kind := EvidenceItemKind
	if counter {
		kind = CounterEvidenceItemKind
	}
	id := candidate.Reference.EvidenceID
	if !candidate.PolicyEligible {
		markRejectedRecord(record, ReasonOmittedPolicy, nonEmptyReason(candidate.PolicyReason, "evidence is not policy eligible"))
		*omissions = append(*omissions, Omission{Kind: kind, CanonicalReference: id, ReasonCode: ReasonOmittedPolicy, Reason: nonEmptyReason(candidate.PolicyReason, "evidence is not policy eligible")})
		return false
	}
	status := temporalStatus(candidate.Reference, referenceTime, policy.StaleAfter)
	record.TemporalStatus = status
	if status == TemporalFuture && policy.ExcludeFutureEvidence {
		*omissions = append(*omissions, Omission{Kind: kind, CanonicalReference: id, ReasonCode: ReasonOmittedFuture, Reason: "evidence is after the explicit reference time"})
		return false
	}
	if status == TemporalStale && policy.ExcludeStaleEvidence {
		*omissions = append(*omissions, Omission{Kind: kind, CanonicalReference: id, ReasonCode: ReasonOmittedStale, Reason: "evidence exceeded the explicit staleness interval"})
		return false
	}
	if policy.RequireKnownObservationTime && candidate.Reference.ObservedAt == nil {
		*omissions = append(*omissions, Omission{Kind: kind, CanonicalReference: id, ReasonCode: ReasonOmittedUnknownTiming, Reason: "observation time is unknown"})
		return false
	}
	return true
}

func uniqueAndRankEvidence(candidates []rankedEvidence, records map[string]RetrievalRecord) ([]rankedEvidence, []Omission) {
	best := map[string]rankedEvidence{}
	omissions := []Omission{}
	for _, candidate := range candidates {
		key := candidate.candidate.DeduplicationKey
		if key == "" {
			key = "id:" + candidate.candidate.Reference.EvidenceID
		}
		if existing, ok := best[key]; ok {
			if evidenceLess(candidate, existing) {
				markRejected(records, existing.recordID, ReasonOmittedDuplicate, "duplicate underlying source was superseded by a higher-ranked reference")
				omissions = append(omissions, Omission{Kind: itemKind(records[existing.recordID]), CanonicalReference: existing.candidate.Reference.EvidenceID, ReasonCode: ReasonOmittedDuplicate, Reason: "duplicate underlying source was superseded by a higher-ranked reference"})
				best[key] = candidate
			} else {
				markRejected(records, candidate.recordID, ReasonOmittedDuplicate, "duplicate underlying source")
				omissions = append(omissions, Omission{Kind: itemKind(records[candidate.recordID]), CanonicalReference: candidate.candidate.Reference.EvidenceID, ReasonCode: ReasonOmittedDuplicate, Reason: "duplicate underlying source"})
			}
			continue
		}
		best[key] = candidate
	}
	ordered := make([]rankedEvidence, 0, len(best))
	for _, candidate := range best {
		ordered = append(ordered, candidate)
	}
	sort.SliceStable(ordered, func(i, j int) bool { return evidenceLess(ordered[i], ordered[j]) })
	return ordered, omissions
}

func uniqueAndRankMemory(candidates []rankedMemory, records map[string]RetrievalRecord) ([]rankedMemory, []Omission) {
	best := map[string]rankedMemory{}
	omissions := []Omission{}
	for _, candidate := range candidates {
		key := candidate.candidate.Reference.MemoryID
		if existing, ok := best[key]; ok {
			if memoryLess(candidate, existing) {
				markRejected(records, existing.recordID, ReasonOmittedDuplicate, "duplicate memory reference was superseded")
				omissions = append(omissions, Omission{Kind: MemoryItemKind, CanonicalReference: key, ReasonCode: ReasonOmittedDuplicate, Reason: "duplicate memory reference was superseded"})
				best[key] = candidate
			} else {
				markRejected(records, candidate.recordID, ReasonOmittedDuplicate, "duplicate memory reference")
				omissions = append(omissions, Omission{Kind: MemoryItemKind, CanonicalReference: key, ReasonCode: ReasonOmittedDuplicate, Reason: "duplicate memory reference"})
			}
			continue
		}
		best[key] = candidate
	}
	ordered := make([]rankedMemory, 0, len(best))
	for _, candidate := range best {
		ordered = append(ordered, candidate)
	}
	sort.SliceStable(ordered, func(i, j int) bool { return memoryLess(ordered[i], ordered[j]) })
	return ordered, omissions
}

func evidenceLess(left, right rankedEvidence) bool {
	if left.candidate.PolicyRequired != right.candidate.PolicyRequired {
		return left.candidate.PolicyRequired
	}
	if left.candidate.Factors.RankScore() != right.candidate.Factors.RankScore() {
		return left.candidate.Factors.RankScore() > right.candidate.Factors.RankScore()
	}
	if left.candidate.EstimatedUnits != right.candidate.EstimatedUnits {
		return left.candidate.EstimatedUnits < right.candidate.EstimatedUnits
	}
	if left.candidate.Reference.EvidenceID != right.candidate.Reference.EvidenceID {
		return left.candidate.Reference.EvidenceID < right.candidate.Reference.EvidenceID
	}
	return left.recordID < right.recordID
}

func memoryLess(left, right rankedMemory) bool {
	if left.candidate.Factors.RankScore() != right.candidate.Factors.RankScore() {
		return left.candidate.Factors.RankScore() > right.candidate.Factors.RankScore()
	}
	if left.candidate.EstimatedUnits != right.candidate.EstimatedUnits {
		return left.candidate.EstimatedUnits < right.candidate.EstimatedUnits
	}
	return left.candidate.Reference.MemoryID < right.candidate.Reference.MemoryID
}

func sourceKey(candidate EvidenceCandidate) string {
	if candidate.SourceGroup != "" {
		return candidate.SourceGroup
	}
	return candidate.Reference.ProviderIdentity
}

func temporalStatus(reference harnesscontracts.EvidenceReference, asOf time.Time, staleAfter time.Duration) string {
	times := []*time.Time{reference.ObservedAt, reference.PublishedAt, reference.FirstSeenAt}
	var latest *time.Time
	for _, value := range times {
		if value == nil {
			continue
		}
		if value.After(asOf) {
			return TemporalFuture
		}
		if latest == nil || value.After(*latest) {
			copyValue := *value
			latest = &copyValue
		}
	}
	if latest != nil && staleAfter > 0 && asOf.Sub(*latest) > staleAfter {
		return TemporalStale
	}
	if reference.ObservedAt == nil {
		return TemporalObservationUnknown
	}
	if reference.PublishedAt == nil {
		return TemporalPublicationUnknown
	}
	if reference.FirstSeenAt == nil {
		return TemporalFirstSeenUnknown
	}
	return TemporalFresh
}

func newEvidenceRecord(candidate EvidenceCandidate, kind, retrieverID, retrieverVersion string, retrievedAt time.Time) RetrievalRecord {
	classification := candidate.Reference.Classification
	return RetrievalRecord{ContractVersion: RetrievalRecordContractV1, RecordID: makeRecordID(kind, candidate.Reference.EvidenceID, candidate.Reference.SourceReference, retrieverID), ItemKind: kind, EvidenceReference: copyEvidenceReference(candidate.Reference), RetrieverID: retrieverID, RetrieverVersion: retrieverVersion, RetrievedAt: retrievedAt, Factors: candidate.Factors, Classification: classification, Decision: RetrievalRejected, ReasonCode: "", Reason: "considered but not selected", PolicyEligible: candidate.PolicyEligible, EstimatedUnits: candidate.EstimatedUnits, RankScore: candidate.Factors.RankScore()}
}

func newMemoryRecord(candidate MemoryCandidate, retrieverID, retrieverVersion string, retrievedAt time.Time) RetrievalRecord {
	return RetrievalRecord{ContractVersion: RetrievalRecordContractV1, RecordID: makeRecordID(MemoryItemKind, candidate.Reference.MemoryID, candidate.Reference.SourceTaskReference, retrieverID), ItemKind: MemoryItemKind, MemoryReference: copyMemoryReference(candidate.Reference), RetrieverID: retrieverID, RetrieverVersion: retrieverVersion, RetrievedAt: retrievedAt, Factors: candidate.Factors, Classification: "memory", Decision: RetrievalRejected, ReasonCode: "", Reason: "considered but not selected", PolicyEligible: candidate.PolicyEligible, EstimatedUnits: candidate.EstimatedUnits, RankScore: candidate.Factors.RankScore()}
}

func markSelected(records map[string]RetrievalRecord, recordID string, counter bool, candidate EvidenceCandidate, referenceTime time.Time) {
	record := records[recordID]
	record.Decision = RetrievalSelected
	record.ReasonCode = ReasonSelectedHighRelevance
	if counter {
		record.ReasonCode = ReasonSelectedCounterEvidence
	} else if candidate.PolicyRequired {
		record.ReasonCode = ReasonSelectedPolicyRequired
	}
	record.Reason = "selected by deterministic multi-factor ranking under the reserved budget"
	record.BudgetImpact = candidate.EstimatedUnits
	record.TemporalStatus = temporalStatus(candidate.Reference, referenceTime, 0)
	record.DeferredReference = ""
	records[recordID] = record
}

func markSelectedMemory(records map[string]RetrievalRecord, recordID string, candidate MemoryCandidate) {
	record := records[recordID]
	record.Decision = RetrievalSelected
	record.ReasonCode = ReasonSelectedHighRelevance
	record.Reason = "selected by deterministic memory ranking under the memory budget"
	record.BudgetImpact = candidate.EstimatedUnits
	records[recordID] = record
}

func markRejected(records map[string]RetrievalRecord, recordID, reasonCode, reason string) {
	record := records[recordID]
	record.Decision = RetrievalRejected
	record.ReasonCode = reasonCode
	record.Reason = nonEmptyReason(reason, "not selected")
	records[recordID] = record
}

func markRejectedRecord(record *RetrievalRecord, reasonCode, reason string) {
	record.Decision = RetrievalRejected
	record.ReasonCode = reasonCode
	record.Reason = nonEmptyReason(reason, "not selected")
}

func markDeferred(records map[string]RetrievalRecord, recordID, reasonCode, reference string) {
	record := records[recordID]
	record.Decision = RetrievalDeferred
	record.ReasonCode = reasonCode
	record.Reason = "deferred with a stable on-demand reference because the budget was exhausted"
	record.DeferredReference = reference
	records[recordID] = record
}

func copyEvidenceReference(reference harnesscontracts.EvidenceReference) *harnesscontracts.EvidenceReference {
	copyValue := reference
	return &copyValue
}
func copyMemoryReference(reference harnesscontracts.MemoryReference) *harnesscontracts.MemoryReference {
	copyValue := reference
	return &copyValue
}

func makeRecordID(kind, identity, source, retriever string) string {
	hash, _ := harnesscontracts.CanonicalHash(map[string]string{"kind": kind, "identity": identity, "source": source, "retriever": retriever})
	return strings.ToLower(kind) + ":" + identity + ":" + hash[len("sha256:"):len("sha256:")+16]
}

func itemKind(record RetrievalRecord) string { return record.ItemKind }
func nonEmptyReason(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func mapToSortedRecords(records map[string]RetrievalRecord) []RetrievalRecord {
	result := make([]RetrievalRecord, 0, len(records))
	for _, record := range records {
		if record.ReasonCode == "" {
			record.ReasonCode = ReasonUnknownMetadata
		}
		result = append(result, record)
	}
	sortRetrievalRecords(result)
	return result
}

func selectionReasonList(records map[string]RetrievalRecord) []string {
	seen := map[string]struct{}{}
	for _, record := range records {
		if record.Decision == RetrievalSelected {
			seen[record.ReasonCode] = struct{}{}
		}
	}
	result := make([]string, 0, len(seen))
	for reason := range seen {
		result = append(result, reason)
	}
	sort.Strings(result)
	return result
}

func omissionIDs(omissions []Omission, kind string) []string {
	result := []string{}
	for _, omission := range omissions {
		if omission.Kind == kind {
			result = append(result, omission.CanonicalReference)
		}
	}
	return uniqueSorted(result)
}

func deferredIDs(deferred []DeferredReference, kind string) []string {
	result := []string{}
	for _, reference := range deferred {
		if reference.Kind == kind {
			result = append(result, reference.CanonicalReference)
		}
	}
	return uniqueSorted(result)
}

func countDecisions(records []RetrievalRecord) (rejected, deferred int) {
	for _, record := range records {
		switch record.Decision {
		case RetrievalRejected:
			rejected++
		case RetrievalDeferred:
			deferred++
		}
	}
	return rejected, deferred
}

func sumEvidenceUnits(candidates []rankedEvidence) int64 {
	var total int64
	for _, candidate := range candidates {
		total += candidate.candidate.EstimatedUnits
	}
	return total
}
func sumMemoryUnits(candidates []rankedMemory) int64 {
	var total int64
	for _, candidate := range candidates {
		total += candidate.candidate.EstimatedUnits
	}
	return total
}
func evidenceIDs(references []harnesscontracts.EvidenceReference) []string {
	result := make([]string, 0, len(references))
	for _, reference := range references {
		result = append(result, reference.EvidenceID)
	}
	return uniqueSorted(result)
}
func memoryIDs(references []harnesscontracts.MemoryReference) []string {
	result := make([]string, 0, len(references))
	for _, reference := range references {
		result = append(result, reference.MemoryID)
	}
	return uniqueSorted(result)
}
func toolIDs(references []harnesscontracts.ToolReference) []string {
	result := make([]string, 0, len(references))
	for _, reference := range references {
		result = append(result, reference.ToolInvocationID)
	}
	return uniqueSorted(result)
}

func referencesFromEvidence(candidates []rankedEvidence) []harnesscontracts.EvidenceReference {
	result := make([]harnesscontracts.EvidenceReference, 0, len(candidates))
	for _, candidate := range candidates {
		result = append(result, candidate.candidate.Reference)
	}
	return result
}
func referencesFromMemory(candidates []rankedMemory) []harnesscontracts.MemoryReference {
	result := make([]harnesscontracts.MemoryReference, 0, len(candidates))
	for _, candidate := range candidates {
		result = append(result, candidate.candidate.Reference)
	}
	return result
}

func uniqueSorted(values []string) []string {
	seen := map[string]struct{}{}
	result := []string{}
	for _, value := range values {
		if strings.TrimSpace(value) == "" {
			continue
		}
		if _, ok := seen[value]; !ok {
			seen[value] = struct{}{}
			result = append(result, value)
		}
	}
	sort.Strings(result)
	return result
}

func buildModelPurpose(objective harnesscontracts.ResearchObjective) string {
	purpose := objective.Purpose + " Question: " + objective.Question
	if len(purpose) > 4096 {
		return purpose[:4096]
	}
	return purpose
}
