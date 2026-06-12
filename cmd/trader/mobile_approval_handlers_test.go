package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	approvalsmod "jax-trading-assistant/internal/modules/approvals"
)

func TestMobileTelegramWebhookRejectsInvalidPayloadBeforeApprovalService(t *testing.T) {
	handler := mobileTelegramWebhookHandler(approvalsmod.NewService(nil))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/mobile/telegram/webhook", strings.NewReader("{"))
	rec := httptest.NewRecorder()

	handler(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestMobileTelegramWebhookBodyParsesTelegramCallbackQuery(t *testing.T) {
	body := mobileTelegramWebhookBody{
		GuardrailHash: "guardrail:v1",
		CallbackQuery: &struct {
			Data string `json:"data"`
			From *struct {
				ID       int64  `json:"id"`
				Username string `json:"username"`
			} `json:"from"`
		}{
			Data: "approve:plain-token",
			From: &struct {
				ID       int64  `json:"id"`
				Username string `json:"username"`
			}{ID: 12345, Username: "operator"},
		},
	}

	req := body.mobileDecisionRequest(time.Date(2026, 6, 12, 10, 0, 0, 0, time.UTC))
	if req.Token != "plain-token" || req.Decision != "approve" {
		t.Fatalf("parsed token/action = %q/%q", req.Token, req.Decision)
	}
	if req.Actor != "operator" || req.Channel != "telegram" || req.RuntimeMode != "paper" {
		t.Fatalf("parsed actor/channel/runtime = %q/%q/%q", req.Actor, req.Channel, req.RuntimeMode)
	}
	if req.GuardrailHash != "guardrail:v1" {
		t.Fatalf("guardrail hash = %q", req.GuardrailHash)
	}
}

func TestMobileApprovalErrorMapping(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want int
	}{
		{name: "expired token", err: approvalsmod.ErrMobileApprovalExpired, want: http.StatusGone},
		{name: "used token", err: approvalsmod.ErrMobileApprovalUsed, want: http.StatusConflict},
		{name: "guardrail drift", err: approvalsmod.ErrMobileApprovalGuardrailChanged, want: http.StatusForbidden},
		{name: "live mode", err: approvalsmod.ErrMobileApprovalLiveMode, want: http.StatusForbidden},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			writeMobileApprovalError(rec, tc.err)
			if rec.Code != tc.want {
				t.Fatalf("status = %d, want %d", rec.Code, tc.want)
			}
		})
	}
}
