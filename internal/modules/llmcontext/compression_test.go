package llmcontext

import (
	"strings"
	"testing"
)

func TestCompressionPolicyNeverCompressesTradingTruth(t *testing.T) {
	policy := DefaultCompressionPolicy()

	for _, field := range []string{"stop_loss", "take_profit", "risk_amount", "approval_token", "priced_in_verdict", "guardrail_results"} {
		if zone := policy.ZoneForField(field); zone != CompressionZoneNever {
			t.Fatalf("field %s zone = %s, want never", field, zone)
		}
	}
}

func TestCompressionServiceRejectsZoneAAndPreservesSourceIDs(t *testing.T) {
	service := NewCompressionService(DefaultCompressionPolicy(), NoopCompressor{})

	_, err := service.Compress(CompressRequest{
		FieldName:    "stop_loss",
		Text:         "stop at 505",
		SourceIDs:    []string{"src-1"},
		RetrievalKey: "raw/src-1",
	})
	if err == nil {
		t.Fatal("expected Zone A compression to fail")
	}

	env, err := service.Compress(CompressRequest{
		FieldName:    "article_body",
		Text:         strings.Repeat("macro surprise ", 20),
		SourceIDs:    []string{"src-1", "src-2"},
		RetrievalKey: "raw/event-1",
	})
	if err != nil {
		t.Fatalf("Compress returned error: %v", err)
	}
	if env.CompressionZone != CompressionZoneSafe || len(env.SourceIDs) != 2 || !env.OriginalAvailable {
		t.Fatalf("compressed envelope missing audit metadata: %#v", env)
	}
	if env.ContentHash == "" {
		t.Fatal("expected content hash")
	}
}

func TestApprovalPacketKeepsTradingTruthUncompressed(t *testing.T) {
	packet := ApprovalPacket{
		Symbol:           "SPY",
		Entry:            "510.00",
		StopLoss:         "505.00",
		TakeProfit:       "520.00",
		RiskAmount:       "100.00",
		PricedInVerdict:  PricedInVerdictNotPricedIn,
		GuardrailResults: "pass",
		SupportingContext: []CompressedEnvelope{{
			CompressionAllowed: true,
			CompressionZone:    CompressionZoneSafe,
			SourceIDs:          []string{"src-1"},
			OriginalAvailable:  true,
			RetrievalKey:       "raw/src-1",
			ContentHash:        "abc",
			CompressedText:     "compressed article",
		}},
	}

	if err := ValidateApprovalPacket(packet); err != nil {
		t.Fatalf("ValidateApprovalPacket returned error: %v", err)
	}
	packet.StopLoss = ""
	if err := ValidateApprovalPacket(packet); err == nil {
		t.Fatal("expected missing stop loss to fail")
	}
}
