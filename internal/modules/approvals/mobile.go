package approvals

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	MobileDecisionApprove = "approve"
	MobileDecisionReject  = "reject"
	MobileDecisionSnooze  = "snooze"
	MobileDecisionAskJax  = "ask_jax"
)

var (
	ErrMobileApprovalExpired          = errors.New("mobile approval token expired")
	ErrMobileApprovalUsed             = errors.New("mobile approval token already used")
	ErrMobileApprovalInvalid          = errors.New("mobile approval token invalid")
	ErrMobileApprovalGuardrailChanged = errors.New("mobile approval guardrails changed")
	ErrMobileApprovalLiveMode         = errors.New("mobile approval cannot create live execution")
	ErrMobileApprovalDecisionInvalid  = errors.New("mobile approval decision invalid")
)

// MobileApprovalTokenRecord is the DB-safe representation of a one-time mobile
// approval token. The plain token is only returned to the notification builder.
type MobileApprovalTokenRecord struct {
	ID             uuid.UUID
	NotificationID *uuid.UUID
	CandidateID    uuid.UUID
	Channel        string
	TokenHash      string
	GuardrailHash  string
	ExpiresAt      time.Time
	UsedAt         *time.Time
	Decision       string
	UsedBy         string
	CreatedAt      time.Time
}

type MobileApprovalDecisionRequest struct {
	Token         string
	Decision      string
	Actor         string
	Channel       string
	GuardrailHash string
	RejectReason  string
	RuntimeMode   string
	Now           time.Time

	// Ignored by design. Mobile callbacks may only decide on an existing
	// candidate token, never override the candidate's symbol or order shape.
	CandidateSymbol string
	OrderOverride   string
}

type MobileApprovalSummary struct {
	CandidateID   uuid.UUID
	Symbol        string
	Strategy      string
	Action        string
	Confidence    float64
	Why           string
	PricedInCheck string
	OtherNews     string
	Entry         float64
	StopLoss      float64
	Target        float64
	Risk          string
	ExpiresAt     time.Time
	PlainToken    string
	GuardrailHash string
	RuntimeMode   string
}

type MobileApprovalButton struct {
	Label        string `json:"label"`
	Decision     string `json:"decision"`
	CallbackData string `json:"callbackData"`
}

type MobileApprovalNotification struct {
	Channel     string                 `json:"channel"`
	CandidateID uuid.UUID              `json:"candidateId"`
	Message     string                 `json:"message"`
	Buttons     []MobileApprovalButton `json:"buttons"`
	Payload     map[string]any         `json:"payload"`
}

type NotificationOutboxItem struct {
	ID          uuid.UUID
	Channel     string
	Recipient   *string
	CandidateID uuid.UUID
	Message     string
	Payload     map[string]any
	Status      string
	SendAfter   time.Time
	SentAt      *time.Time
	Error       *string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func NewMobileApprovalToken(candidateID uuid.UUID, channel, guardrailHash string, now time.Time, ttl time.Duration) (string, MobileApprovalTokenRecord, error) {
	if candidateID == uuid.Nil {
		return "", MobileApprovalTokenRecord{}, fmt.Errorf("%w: candidate id required", ErrMobileApprovalInvalid)
	}
	if strings.TrimSpace(channel) == "" {
		return "", MobileApprovalTokenRecord{}, fmt.Errorf("%w: channel required", ErrMobileApprovalInvalid)
	}
	if strings.TrimSpace(guardrailHash) == "" {
		return "", MobileApprovalTokenRecord{}, fmt.Errorf("%w: guardrail hash required", ErrMobileApprovalInvalid)
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	if ttl <= 0 {
		ttl = 10 * time.Minute
	}

	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", MobileApprovalTokenRecord{}, fmt.Errorf("generate mobile approval token: %w", err)
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	record := MobileApprovalTokenRecord{
		ID:            uuid.New(),
		CandidateID:   candidateID,
		Channel:       channel,
		TokenHash:     HashMobileApprovalToken(token),
		GuardrailHash: guardrailHash,
		ExpiresAt:     now.UTC().Add(ttl),
		CreatedAt:     now.UTC(),
	}
	return token, record, nil
}

func HashMobileApprovalToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func MobileApprovalRequestFromToken(record MobileApprovalTokenRecord, mobileReq MobileApprovalDecisionRequest) (ApprovalRequest, error) {
	now := mobileReq.Now.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}
	if HashMobileApprovalToken(mobileReq.Token) != record.TokenHash {
		return ApprovalRequest{}, ErrMobileApprovalInvalid
	}
	if record.UsedAt != nil {
		return ApprovalRequest{}, ErrMobileApprovalUsed
	}
	if now.After(record.ExpiresAt) {
		return ApprovalRequest{}, ErrMobileApprovalExpired
	}
	if mobileReq.GuardrailHash != record.GuardrailHash {
		return ApprovalRequest{}, ErrMobileApprovalGuardrailChanged
	}
	if strings.EqualFold(strings.TrimSpace(mobileReq.RuntimeMode), "live") {
		return ApprovalRequest{}, ErrMobileApprovalLiveMode
	}

	decision, err := mapMobileDecision(mobileReq.Decision)
	if err != nil {
		return ApprovalRequest{}, err
	}
	actor := strings.TrimSpace(mobileReq.Actor)
	if actor == "" {
		actor = "unknown"
	}
	channel := strings.TrimSpace(mobileReq.Channel)
	if channel == "" {
		channel = record.Channel
	}

	notes := buildMobileApprovalNotes(decision, mobileReq.RejectReason)
	return ApprovalRequest{
		CandidateID: record.CandidateID,
		Decision:    decision,
		ApprovedBy:  fmt.Sprintf("mobile:%s:%s", channel, actor),
		Notes:       &notes,
		SnoozeHours: defaultMobileSnoozeHours(decision),
	}, nil
}

func (s *Service) SubmitMobileDecision(ctx context.Context, mobileReq MobileApprovalDecisionRequest) (*Approval, error) {
	record, err := s.store.GetMobileApprovalTokenByHash(ctx, HashMobileApprovalToken(mobileReq.Token))
	if err != nil {
		if isNoRows(err) {
			return nil, ErrMobileApprovalInvalid
		}
		return nil, err
	}
	req, err := MobileApprovalRequestFromToken(*record, mobileReq)
	if err != nil {
		return nil, err
	}
	approval, err := s.Decide(ctx, req)
	if err != nil {
		return nil, err
	}
	if err := s.store.MarkMobileApprovalTokenUsed(ctx, record.ID, approval.Decision, req.ApprovedBy, approval.DecidedAt); err != nil {
		return nil, err
	}
	return approval, nil
}

func BuildMobileApprovalNotification(summary MobileApprovalSummary) MobileApprovalNotification {
	runtimeMode := strings.TrimSpace(summary.RuntimeMode)
	if runtimeMode == "" {
		runtimeMode = "paper"
	}
	message := strings.Join([]string{
		fmt.Sprintf("ETF: %s", summary.Symbol),
		fmt.Sprintf("Strategy: %s", summary.Strategy),
		fmt.Sprintf("Action: %s", summary.Action),
		fmt.Sprintf("Confidence: %.0f%%", summary.Confidence*100),
		fmt.Sprintf("Why: %s", summary.Why),
		fmt.Sprintf("Priced-in check: %s", summary.PricedInCheck),
		fmt.Sprintf("Other news: %s", summary.OtherNews),
		fmt.Sprintf("Entry: %.2f", summary.Entry),
		fmt.Sprintf("Stop-loss: %.2f", summary.StopLoss),
		fmt.Sprintf("Target: %.2f", summary.Target),
		fmt.Sprintf("Risk: %s", summary.Risk),
		fmt.Sprintf("Expires: %s", summary.ExpiresAt.UTC().Format(time.RFC3339)),
	}, "\n")

	return MobileApprovalNotification{
		Channel:     "telegram",
		CandidateID: summary.CandidateID,
		Message:     message,
		Buttons: []MobileApprovalButton{
			button("Approve", MobileDecisionApprove, summary.PlainToken),
			button("Reject", MobileDecisionReject, summary.PlainToken),
			button("Snooze", MobileDecisionSnooze, summary.PlainToken),
			button("Ask Jax", MobileDecisionAskJax, summary.PlainToken),
		},
		Payload: map[string]any{
			"candidateId":   summary.CandidateID.String(),
			"guardrailHash": summary.GuardrailHash,
			"runtimeMode":   runtimeMode,
		},
	}
}

func mapMobileDecision(decision string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(decision)) {
	case MobileDecisionApprove:
		return DecisionApproved, nil
	case MobileDecisionReject:
		return DecisionRejected, nil
	case MobileDecisionSnooze:
		return DecisionSnoozed, nil
	case MobileDecisionAskJax:
		return DecisionReanalysisRequested, nil
	default:
		return "", ErrMobileApprovalDecisionInvalid
	}
}

func buildMobileApprovalNotes(decision, reason string) string {
	reason = strings.TrimSpace(reason)
	switch decision {
	case DecisionRejected:
		if reason == "" {
			reason = "no reason supplied"
		}
		return "paper-only mobile rejection: " + reason
	default:
		return "paper-only mobile approval flow"
	}
}

func defaultMobileSnoozeHours(decision string) int {
	if decision == DecisionSnoozed {
		return 1
	}
	return 0
}

func button(label, decision, token string) MobileApprovalButton {
	return MobileApprovalButton{
		Label:        label,
		Decision:     decision,
		CallbackData: fmt.Sprintf("%s:%s", decision, token),
	}
}
