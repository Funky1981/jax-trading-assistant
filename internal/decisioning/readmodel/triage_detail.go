package readmodel

import (
	"time"

	"jax-trading-assistant/internal/decisioning/triage"
)

type TriageItemDetail struct {
	TriageItemID           string            `json:"triage_item_id"`
	SourceType             triage.SourceType `json:"source_type"`
	SourceID               string            `json:"source_id"`
	EventID                string            `json:"event_id"`
	Asset                  string            `json:"asset"`
	SetupFamily            string            `json:"setup_family"`
	Priority               triage.Priority   `json:"priority"`
	Status                 triage.Status     `json:"status"`
	Reason                 string            `json:"reason"`
	EvidenceRefs           []string          `json:"evidence_refs"`
	SuggestedAction        string            `json:"suggested_action"`
	AllowedFollowUpActions []string          `json:"allowed_follow_up_actions"`
	ForbiddenActions       []string          `json:"forbidden_actions"`
	RequiresHumanApproval  bool              `json:"requires_human_approval"`
	AutoApplyAllowed       bool              `json:"auto_apply_allowed"`
	DueAt                  time.Time         `json:"due_at"`
	CreatedAt              time.Time         `json:"created_at"`
	UpdatedAt              time.Time         `json:"updated_at"`
}

func BuildTriageItemDetail(item triage.Item) (TriageItemDetail, error) {
	if err := triage.ValidateItem(item); err != nil {
		return TriageItemDetail{}, err
	}
	return TriageItemDetail{
		TriageItemID:           item.TriageItemID,
		SourceType:             item.SourceType,
		SourceID:               item.SourceID,
		EventID:                item.EventID,
		Asset:                  item.Asset,
		SetupFamily:            item.SetupFamily,
		Priority:               item.Priority,
		Status:                 item.Status,
		Reason:                 item.Reason,
		EvidenceRefs:           append([]string{}, item.EvidenceRefs...),
		SuggestedAction:        item.SuggestedAction,
		AllowedFollowUpActions: append([]string{}, item.AllowedFollowUpActions...),
		ForbiddenActions:       append([]string{}, item.ForbiddenActions...),
		RequiresHumanApproval:  true,
		AutoApplyAllowed:       false,
		DueAt:                  item.DueAt,
		CreatedAt:              item.CreatedAt,
		UpdatedAt:              item.UpdatedAt,
	}, nil
}
