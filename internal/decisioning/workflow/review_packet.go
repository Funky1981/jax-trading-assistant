package workflow

import (
	"fmt"
	"time"

	"jax-trading-assistant/internal/decisioning/triage"
)

type ReviewPacket struct {
	PacketID               string            `json:"packet_id"`
	TriageItemID           string            `json:"triage_item_id"`
	SourceType             triage.SourceType `json:"source_type"`
	Priority               triage.Priority   `json:"priority"`
	Status                 triage.Status     `json:"status"`
	EventID                string            `json:"event_id"`
	Asset                  string            `json:"asset"`
	SetupFamily            string            `json:"setup_family"`
	Reason                 string            `json:"reason"`
	EvidenceRefs           []string          `json:"evidence_refs"`
	SuggestedAction        string            `json:"suggested_action"`
	AllowedFollowUpActions []string          `json:"allowed_follow_up_actions"`
	ForbiddenActions       []string          `json:"forbidden_actions"`
	RequiresHumanApproval  bool              `json:"requires_human_approval"`
	AutoApplyAllowed       bool              `json:"auto_apply_allowed"`
	DueAt                  time.Time         `json:"due_at"`
	CreatedAt              time.Time         `json:"created_at"`
}

func BuildReviewPacket(item triage.Item) (ReviewPacket, error) {
	if err := triage.ValidateItem(item); err != nil {
		return ReviewPacket{}, err
	}
	if item.TriageItemID == "" {
		return ReviewPacket{}, fmt.Errorf("triage item id is required")
	}
	return ReviewPacket{
		PacketID:               "packet_" + item.TriageItemID,
		TriageItemID:           item.TriageItemID,
		SourceType:             item.SourceType,
		Priority:               item.Priority,
		Status:                 item.Status,
		EventID:                item.EventID,
		Asset:                  item.Asset,
		SetupFamily:            item.SetupFamily,
		Reason:                 item.Reason,
		EvidenceRefs:           append([]string{}, item.EvidenceRefs...),
		SuggestedAction:        item.SuggestedAction,
		AllowedFollowUpActions: append([]string{}, item.AllowedFollowUpActions...),
		ForbiddenActions:       append([]string{}, item.ForbiddenActions...),
		RequiresHumanApproval:  true,
		AutoApplyAllowed:       false,
		DueAt:                  item.DueAt,
		CreatedAt:              item.CreatedAt,
	}, nil
}
