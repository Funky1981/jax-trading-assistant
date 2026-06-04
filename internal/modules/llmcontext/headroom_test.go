package llmcontext

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHeadroomTrialDisabledByDefault(t *testing.T) {
	trial := NewHeadroomTrial(HeadroomConfig{})
	result, err := trial.Compress(context.Background(), HeadroomRequest{
		FieldName:    "article_body",
		TaskType:     TaskHistoricalSummary,
		Text:         "long article body",
		SourceIDs:    []string{"src-1"},
		RetrievalKey: "raw/src-1",
	})
	if err != nil {
		t.Fatalf("Compress returned error: %v", err)
	}
	if result.Used {
		t.Fatalf("expected disabled trial not to run: %#v", result)
	}
	if result.DisabledReason != "headroom_disabled" {
		t.Fatalf("unexpected disabled reason: %#v", result)
	}
}

func TestHeadroomTrialRejectsNonZoneCAndLiveApproval(t *testing.T) {
	trial := NewHeadroomTrial(HeadroomConfig{Enabled: true, BaseURL: "http://headroom.test", HTTP: http.DefaultClient})

	tests := []HeadroomRequest{
		{FieldName: "stop_loss", TaskType: TaskHistoricalSummary, Text: "505.00", SourceIDs: []string{"src-1"}, RetrievalKey: "raw/src-1"},
		{FieldName: "candles", TaskType: TaskHistoricalSummary, Text: "many candles", SourceIDs: []string{"src-1"}, RetrievalKey: "raw/src-1"},
		{FieldName: "article_body", TaskType: TaskApprovalSummary, LiveApprovalWorkflow: true, Text: "article", SourceIDs: []string{"src-1"}, RetrievalKey: "raw/src-1"},
	}

	for _, tt := range tests {
		result, err := trial.Compress(context.Background(), tt)
		if err != nil {
			t.Fatalf("Compress returned error: %v", err)
		}
		if result.Used {
			t.Fatalf("expected compression to be skipped for %#v, got %#v", tt, result)
		}
	}
}

func TestHeadroomTrialCompressesZoneCAndRecordsSavings(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/compress" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"compressed_text":"short summary"}`))
	}))
	defer server.Close()
	trial := NewHeadroomTrial(HeadroomConfig{Enabled: true, BaseURL: server.URL, HTTP: server.Client()})

	result, err := trial.Compress(context.Background(), HeadroomRequest{
		FieldName:    "article_body",
		TaskType:     TaskHistoricalSummary,
		Text:         strings.Repeat("macro surprise ", 120),
		SourceIDs:    []string{"src-1", "src-2"},
		RetrievalKey: "raw/event-1",
	})
	if err != nil {
		t.Fatalf("Compress returned error: %v", err)
	}
	if !result.Used {
		t.Fatalf("expected Headroom to be used: %#v", result)
	}
	if result.Envelope.CompressedText != "short summary" || result.Envelope.CompressionZone != CompressionZoneSafe {
		t.Fatalf("unexpected envelope: %#v", result.Envelope)
	}
	if result.OriginalTokens <= result.CompressedTokens {
		t.Fatalf("expected token savings, got original=%d compressed=%d", result.OriginalTokens, result.CompressedTokens)
	}
	if result.SavingPercent <= 0 || result.LatencyMillis < 0 {
		t.Fatalf("unexpected metrics: %#v", result)
	}
}
