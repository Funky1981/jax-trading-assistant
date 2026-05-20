package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

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
