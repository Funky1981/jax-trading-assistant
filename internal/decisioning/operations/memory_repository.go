package operations

import (
	"fmt"
	"sort"
	"time"

	"jax-trading-assistant/internal/decisioning/feedback"
	"jax-trading-assistant/internal/decisioning/triage"
)

type MemoryRepository struct {
	triageItems       map[string]triage.Item
	feedbackDecisions map[string]FeedbackDecision
	followUpActions   map[string]FollowUpAction
	auditRecords      []OperationAuditRecord
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		triageItems:       map[string]triage.Item{},
		feedbackDecisions: map[string]FeedbackDecision{},
		followUpActions:   map[string]FollowUpAction{},
	}
}

func (r *MemoryRepository) SaveTriageItem(item triage.Item) error {
	before, hadBefore := r.triageItems[item.TriageItemID]
	normalized := cloneTriageItem(item)
	normalized.ForbiddenActions = mergeForbiddenActions(normalized.ForbiddenActions)
	normalized.RequiresHumanApproval = true
	autoApplyBlocked := normalized.AutoApplyAllowed
	normalized.AutoApplyAllowed = false
	if err := triage.ValidateItem(normalized); err != nil {
		return err
	}
	r.triageItems[normalized.TriageItemID] = normalized

	beforeStatus := ""
	if hadBefore {
		beforeStatus = string(before.Status)
	}
	r.SaveOperationAuditRecord(newAuditRecord(
		AuditActionTriageItemSaved,
		normalized.TriageItemID,
		"",
		"",
		string(normalized.SourceType),
		"system",
		beforeStatus,
		string(normalized.Status),
		"triage item persisted for human review operations",
		auditTime(normalized.UpdatedAt, normalized.CreatedAt),
	))
	if autoApplyBlocked {
		r.SaveOperationAuditRecord(newAuditRecord(
			AuditActionAutoApplyBlocked,
			normalized.TriageItemID,
			"",
			"",
			string(normalized.SourceType),
			"system",
			string(item.Status),
			string(normalized.Status),
			"auto_apply_allowed was normalized to false; suggestions cannot be auto-applied",
			auditTime(normalized.UpdatedAt, normalized.CreatedAt),
		))
	}
	return nil
}

func (r *MemoryRepository) GetTriageItem(id string) (triage.Item, bool) {
	item, ok := r.triageItems[id]
	if !ok {
		return triage.Item{}, false
	}
	return cloneTriageItem(item), true
}

func (r *MemoryRepository) ListTriageItems() []triage.Item {
	items := make([]triage.Item, 0, len(r.triageItems))
	for _, item := range r.triageItems {
		items = append(items, cloneTriageItem(item))
	}
	sortTriageItems(items)
	return items
}

func (r *MemoryRepository) ListOpenTriageItems() []triage.Item {
	items := r.filterTriageItems(func(item triage.Item) bool {
		return item.Status == triage.StatusOpen
	})
	sortTriageItemsByPriorityDueID(items)
	return items
}

func (r *MemoryRepository) ListHighPriorityTriageItems() []triage.Item {
	items := r.filterTriageItems(func(item triage.Item) bool {
		return item.Status == triage.StatusOpen && (item.Priority == triage.PriorityHigh || item.Priority == triage.PriorityCritical)
	})
	sortTriageItemsByPriorityDueID(items)
	return items
}

func (r *MemoryRepository) ListDueTriageItems(asOf time.Time) []triage.Item {
	items := r.filterTriageItems(func(item triage.Item) bool {
		return item.Status == triage.StatusOpen && !item.DueAt.IsZero() && !item.DueAt.After(asOf)
	})
	sortTriageItemsByPriorityDueID(items)
	return items
}

func (r *MemoryRepository) SaveHumanFeedbackDecision(decision FeedbackDecision) error {
	if decision.FeedbackDecisionID == "" {
		return fmt.Errorf("feedback decision id is required")
	}
	if decision.TriageItemID == "" {
		return fmt.Errorf("triage item id is required")
	}
	if decision.HumanReviewer == "" {
		return fmt.Errorf("human reviewer is required")
	}
	if decision.Rationale == "" {
		return fmt.Errorf("rationale is required")
	}
	switch decision.Decision {
	case DecisionAcceptSuggestion, DecisionRejectSuggestion, DecisionDeferDecision, DecisionRequestMoreEvidence, DecisionCloseNoAction:
	default:
		return fmt.Errorf("decision %q is not allowed", decision.Decision)
	}
	copied := cloneFeedbackDecision(decision)
	r.feedbackDecisions[copied.FeedbackDecisionID] = copied
	r.SaveOperationAuditRecord(newAuditRecord(
		AuditActionFeedbackDecisionSaved,
		copied.TriageItemID,
		copied.FeedbackDecisionID,
		"",
		"",
		copied.HumanReviewer,
		"",
		statusForDecision(copied.Decision),
		copied.Rationale,
		copied.CreatedAt,
	))
	return nil
}

func (r *MemoryRepository) GetHumanFeedbackDecision(id string) (FeedbackDecision, bool) {
	decision, ok := r.feedbackDecisions[id]
	if !ok {
		return FeedbackDecision{}, false
	}
	return cloneFeedbackDecision(decision), true
}

func (r *MemoryRepository) ListFeedbackDecisionsForTriageItem(triageItemID string) []FeedbackDecision {
	decisions := make([]FeedbackDecision, 0)
	for _, decision := range r.feedbackDecisions {
		if decision.TriageItemID == triageItemID {
			decisions = append(decisions, cloneFeedbackDecision(decision))
		}
	}
	sort.SliceStable(decisions, func(i, j int) bool {
		if !decisions[i].CreatedAt.Equal(decisions[j].CreatedAt) {
			return decisions[i].CreatedAt.Before(decisions[j].CreatedAt)
		}
		return decisions[i].FeedbackDecisionID < decisions[j].FeedbackDecisionID
	})
	return decisions
}

func (r *MemoryRepository) SaveFollowUpAction(action FollowUpAction) error {
	normalized := cloneFollowUpAction(action)
	normalized.ForbiddenActions = mergeForbiddenActions(normalized.ForbiddenActions)
	normalized.RequiresHumanApproval = true
	autoApplyBlocked := normalized.AutoApplyAllowed
	normalized.AutoApplyAllowed = false
	if err := ValidateFollowUpAction(normalized); err != nil {
		return err
	}
	r.followUpActions[normalized.ActionID] = normalized
	r.SaveOperationAuditRecord(newAuditRecord(
		AuditActionFollowUpActionSaved,
		normalized.TriageItemID,
		"",
		normalized.ActionID,
		"",
		"system",
		"",
		string(normalized.Status),
		"follow-up action persisted for manual review only",
		normalized.CreatedAt,
	))
	if autoApplyBlocked {
		r.SaveOperationAuditRecord(newAuditRecord(
			AuditActionAutoApplyBlocked,
			normalized.TriageItemID,
			"",
			normalized.ActionID,
			"",
			"system",
			string(action.Status),
			string(normalized.Status),
			"auto_apply_allowed was normalized to false; follow-up actions cannot change rules automatically",
			normalized.CreatedAt,
		))
	}
	return nil
}

func (r *MemoryRepository) GetFollowUpAction(id string) (FollowUpAction, bool) {
	action, ok := r.followUpActions[id]
	if !ok {
		return FollowUpAction{}, false
	}
	return cloneFollowUpAction(action), true
}

func (r *MemoryRepository) ListFollowUpActionsForTriageItem(triageItemID string) []FollowUpAction {
	actions := make([]FollowUpAction, 0)
	for _, action := range r.followUpActions {
		if action.TriageItemID == triageItemID {
			actions = append(actions, cloneFollowUpAction(action))
		}
	}
	sortFollowUpActions(actions)
	return actions
}

func (r *MemoryRepository) SaveOperationAuditRecord(record OperationAuditRecord) {
	copied := record
	copied.ForbiddenActions = mergeForbiddenActions(copied.ForbiddenActions)
	r.auditRecords = append(r.auditRecords, copied)
}

func (r *MemoryRepository) ListOperationAuditRecords() []OperationAuditRecord {
	records := append([]OperationAuditRecord(nil), r.auditRecords...)
	sort.SliceStable(records, func(i, j int) bool {
		if !records[i].CreatedAt.Equal(records[j].CreatedAt) {
			return records[i].CreatedAt.Before(records[j].CreatedAt)
		}
		return records[i].AuditID < records[j].AuditID
	})
	for i := range records {
		records[i].ForbiddenActions = append([]string(nil), records[i].ForbiddenActions...)
	}
	return records
}

func (r *MemoryRepository) filterTriageItems(include func(triage.Item) bool) []triage.Item {
	items := make([]triage.Item, 0)
	for _, item := range r.triageItems {
		if include(item) {
			items = append(items, cloneTriageItem(item))
		}
	}
	return items
}

func statusForDecision(decision HumanDecision) string {
	switch decision {
	case DecisionAcceptSuggestion:
		return string(triage.StatusAccepted)
	case DecisionRejectSuggestion:
		return string(triage.StatusRejected)
	case DecisionDeferDecision:
		return string(triage.StatusDeferred)
	case DecisionRequestMoreEvidence:
		return string(triage.StatusNeedsMoreEvidence)
	case DecisionCloseNoAction:
		return string(triage.StatusClosed)
	default:
		return ""
	}
}

func auditTime(preferred, fallback time.Time) time.Time {
	if !preferred.IsZero() {
		return preferred
	}
	return fallback
}

func cloneTriageItem(item triage.Item) triage.Item {
	item.EvidenceRefs = append([]string(nil), item.EvidenceRefs...)
	item.AllowedFollowUpActions = append([]string(nil), item.AllowedFollowUpActions...)
	item.ForbiddenActions = append([]string(nil), item.ForbiddenActions...)
	return item
}

func cloneFeedbackDecision(decision FeedbackDecision) FeedbackDecision {
	decision.AcceptedFollowUpActions = append([]string(nil), decision.AcceptedFollowUpActions...)
	decision.RejectedFollowUpActions = append([]string(nil), decision.RejectedFollowUpActions...)
	decision.RequiredEvidence = append([]string(nil), decision.RequiredEvidence...)
	return decision
}

func cloneFollowUpAction(action FollowUpAction) FollowUpAction {
	action.ForbiddenActions = append([]string(nil), action.ForbiddenActions...)
	return action
}

func sortTriageItems(items []triage.Item) {
	sort.SliceStable(items, func(i, j int) bool {
		return items[i].TriageItemID < items[j].TriageItemID
	})
}

func sortTriageItemsByPriorityDueID(items []triage.Item) {
	sort.SliceStable(items, func(i, j int) bool {
		leftRank := priorityRank(items[i].Priority)
		rightRank := priorityRank(items[j].Priority)
		if leftRank != rightRank {
			return leftRank > rightRank
		}
		if !items[i].DueAt.Equal(items[j].DueAt) {
			return items[i].DueAt.Before(items[j].DueAt)
		}
		return items[i].TriageItemID < items[j].TriageItemID
	})
}

func sortFollowUpActions(actions []FollowUpAction) {
	sort.SliceStable(actions, func(i, j int) bool {
		if !actions[i].CreatedAt.Equal(actions[j].CreatedAt) {
			return actions[i].CreatedAt.Before(actions[j].CreatedAt)
		}
		return actions[i].ActionID < actions[j].ActionID
	})
}

func priorityRank(priority triage.Priority) int {
	switch priority {
	case triage.PriorityCritical:
		return 4
	case triage.PriorityHigh:
		return 3
	case triage.PriorityMedium:
		return 2
	case triage.PriorityLow:
		return 1
	default:
		return 0
	}
}

func forbiddenActions() []string {
	return feedback.ForbiddenActions()
}
