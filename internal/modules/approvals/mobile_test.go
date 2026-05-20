package approvals

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestNewMobileApprovalToken_StoresOnlyHashAndExpiresQuickly(t *testing.T) {
	now := time.Date(2026, 5, 20, 10, 0, 0, 0, time.UTC)
	candidateID := uuid.New()

	token, record, err := NewMobileApprovalToken(candidateID, "telegram", "guardrails:v1", now, 10*time.Minute)
	if err != nil {
		t.Fatalf("NewMobileApprovalToken: %v", err)
	}

	if token == "" {
		t.Fatal("plain token must be returned for button callback construction")
	}
	if record.TokenHash == "" {
		t.Fatal("token hash must be persisted")
	}
	if record.TokenHash == token {
		t.Fatal("plain token must not be stored")
	}
	if record.TokenHash != HashMobileApprovalToken(token) {
		t.Fatal("stored token hash must match generated token")
	}
	if got, want := record.ExpiresAt, now.Add(10*time.Minute); !got.Equal(want) {
		t.Fatalf("expires_at = %s, want %s", got, want)
	}
}

func TestMobileApprovalDecision_ApproveBuildsPaperOnlyCandidateApproval(t *testing.T) {
	now := time.Date(2026, 5, 20, 10, 0, 0, 0, time.UTC)
	token, record, err := NewMobileApprovalToken(uuid.New(), "telegram", "guardrails:v1", now, 10*time.Minute)
	if err != nil {
		t.Fatalf("NewMobileApprovalToken: %v", err)
	}

	req, err := MobileApprovalRequestFromToken(record, MobileApprovalDecisionRequest{
		Token:           token,
		Decision:        MobileDecisionApprove,
		Actor:           "u123",
		Channel:         "telegram",
		GuardrailHash:   "guardrails:v1",
		RuntimeMode:     "paper",
		Now:             now.Add(time.Minute),
		RejectReason:    "ignored for approve",
		CandidateSymbol: "TQQQ",
		OrderOverride:   "live-buy-100",
	})
	if err != nil {
		t.Fatalf("MobileApprovalRequestFromToken: %v", err)
	}

	if req.CandidateID != record.CandidateID {
		t.Fatal("mobile approval must reference the candidate id from the one-time token")
	}
	if req.Decision != DecisionApproved {
		t.Fatalf("decision = %q, want %q", req.Decision, DecisionApproved)
	}
	if req.ApprovedBy != "mobile:telegram:u123" {
		t.Fatalf("approved_by = %q", req.ApprovedBy)
	}
	if req.Notes == nil || !strings.Contains(*req.Notes, "paper-only mobile approval") {
		t.Fatalf("notes should audit paper-only mobile approval, got %#v", req.Notes)
	}
}

func TestMobileApprovalDecision_RejectStoresReason(t *testing.T) {
	now := time.Date(2026, 5, 20, 10, 0, 0, 0, time.UTC)
	token, record, err := NewMobileApprovalToken(uuid.New(), "telegram", "guardrails:v1", now, 10*time.Minute)
	if err != nil {
		t.Fatalf("NewMobileApprovalToken: %v", err)
	}

	req, err := MobileApprovalRequestFromToken(record, MobileApprovalDecisionRequest{
		Token:         token,
		Decision:      MobileDecisionReject,
		Actor:         "u123",
		Channel:       "telegram",
		GuardrailHash: "guardrails:v1",
		RejectReason:  "conflicting Fed headline",
		RuntimeMode:   "paper",
		Now:           now.Add(time.Minute),
	})
	if err != nil {
		t.Fatalf("MobileApprovalRequestFromToken: %v", err)
	}

	if req.Decision != DecisionRejected {
		t.Fatalf("decision = %q, want %q", req.Decision, DecisionRejected)
	}
	if req.Notes == nil || !strings.Contains(*req.Notes, "conflicting Fed headline") {
		t.Fatalf("reject reason missing from notes: %#v", req.Notes)
	}
}

func TestMobileApprovalDecision_RejectsExpiredUsedTamperedAndChangedGuardrails(t *testing.T) {
	now := time.Date(2026, 5, 20, 10, 0, 0, 0, time.UTC)
	token, record, err := NewMobileApprovalToken(uuid.New(), "telegram", "guardrails:v1", now, 10*time.Minute)
	if err != nil {
		t.Fatalf("NewMobileApprovalToken: %v", err)
	}

	cases := []struct {
		name    string
		record  MobileApprovalTokenRecord
		request MobileApprovalDecisionRequest
		wantErr error
	}{
		{
			name:   "expired",
			record: record,
			request: MobileApprovalDecisionRequest{
				Token:         token,
				Decision:      MobileDecisionApprove,
				Actor:         "u123",
				Channel:       "telegram",
				GuardrailHash: "guardrails:v1",
				RuntimeMode:   "paper",
				Now:           now.Add(11 * time.Minute),
			},
			wantErr: ErrMobileApprovalExpired,
		},
		{
			name: "used",
			record: func() MobileApprovalTokenRecord {
				used := record
				usedAt := now.Add(time.Minute)
				used.UsedAt = &usedAt
				return used
			}(),
			request: MobileApprovalDecisionRequest{
				Token:         token,
				Decision:      MobileDecisionApprove,
				Actor:         "u123",
				Channel:       "telegram",
				GuardrailHash: "guardrails:v1",
				RuntimeMode:   "paper",
				Now:           now.Add(2 * time.Minute),
			},
			wantErr: ErrMobileApprovalUsed,
		},
		{
			name:   "tampered token",
			record: record,
			request: MobileApprovalDecisionRequest{
				Token:         token + "x",
				Decision:      MobileDecisionApprove,
				Actor:         "u123",
				Channel:       "telegram",
				GuardrailHash: "guardrails:v1",
				RuntimeMode:   "paper",
				Now:           now.Add(time.Minute),
			},
			wantErr: ErrMobileApprovalInvalid,
		},
		{
			name:   "guardrail changed",
			record: record,
			request: MobileApprovalDecisionRequest{
				Token:         token,
				Decision:      MobileDecisionApprove,
				Actor:         "u123",
				Channel:       "telegram",
				GuardrailHash: "guardrails:v2",
				RuntimeMode:   "paper",
				Now:           now.Add(time.Minute),
			},
			wantErr: ErrMobileApprovalGuardrailChanged,
		},
		{
			name:   "live mode rejected",
			record: record,
			request: MobileApprovalDecisionRequest{
				Token:         token,
				Decision:      MobileDecisionApprove,
				Actor:         "u123",
				Channel:       "telegram",
				GuardrailHash: "guardrails:v1",
				RuntimeMode:   "live",
				Now:           now.Add(time.Minute),
			},
			wantErr: ErrMobileApprovalLiveMode,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := MobileApprovalRequestFromToken(tc.record, tc.request)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("error = %v, want %v", err, tc.wantErr)
			}
		})
	}
}

func TestBuildMobileApprovalNotification_IncludesRequiredTelegramFields(t *testing.T) {
	expires := time.Date(2026, 5, 20, 10, 10, 0, 0, time.UTC)
	notification := BuildMobileApprovalNotification(MobileApprovalSummary{
		CandidateID:   uuid.New(),
		Symbol:        "QQQ",
		Strategy:      "ETF_NEWS_002_SECTOR_MOMENTUM",
		Action:        "Paper Buy",
		Confidence:    0.82,
		Why:           "semiconductor breadth improved",
		PricedInCheck: "not priced in",
		OtherNews:     "no conflicting macro headlines",
		Entry:         431.25,
		StopLoss:      426.10,
		Target:        439.80,
		Risk:          "0.5R",
		ExpiresAt:     expires,
		PlainToken:    "token-123",
		GuardrailHash: "guardrails:v1",
		RuntimeMode:   "paper",
	})

	for _, required := range []string{
		"ETF: QQQ",
		"Strategy: ETF_NEWS_002_SECTOR_MOMENTUM",
		"Action: Paper Buy",
		"Confidence: 82%",
		"Why: semiconductor breadth improved",
		"Priced-in check: not priced in",
		"Other news: no conflicting macro headlines",
		"Entry: 431.25",
		"Stop-loss: 426.10",
		"Target: 439.80",
		"Risk: 0.5R",
		"Expires: 2026-05-20T10:10:00Z",
	} {
		if !strings.Contains(notification.Message, required) {
			t.Fatalf("notification missing %q:\n%s", required, notification.Message)
		}
	}
	if len(notification.Buttons) != 4 {
		t.Fatalf("buttons = %d, want 4", len(notification.Buttons))
	}
	for _, label := range []string{"Approve", "Reject", "Snooze", "Ask Jax"} {
		if !hasButton(notification.Buttons, label) {
			t.Fatalf("missing %q button: %#v", label, notification.Buttons)
		}
	}
	if notification.Payload["runtimeMode"] != "paper" {
		t.Fatalf("runtime mode payload = %#v, want paper", notification.Payload["runtimeMode"])
	}
}

func hasButton(buttons []MobileApprovalButton, label string) bool {
	for _, button := range buttons {
		if button.Label == label {
			return true
		}
	}
	return false
}
