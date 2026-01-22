package observability

import (
	"reflect"
	"testing"
)

func TestRedactValue_RedactsSensitiveFields(t *testing.T) {
	input := map[string]any{
		"symbol":             "AAPL",
		"broker_credentials": map[string]any{"api_key": "abc"},
		"order_payload": map[string]any{
			"price": 123.45,
		},
		"account_id": "acct-123",
		"nested": map[string]any{
			"password": "secret",
		},
	}

	expected := map[string]any{
		"symbol":             "AAPL",
		"broker_credentials": redactedValue,
		"order_payload":      redactedValue,
		"account_id":         redactedValue,
		"nested": map[string]any{
			"password": redactedValue,
		},
	}

	got := RedactValue(input)
	if !reflect.DeepEqual(got, expected) {
		t.Fatalf("expected %#v, got %#v", expected, got)
	}
}

func TestRedactValue_RedactsSliceValues(t *testing.T) {
	input := []any{
		map[string]any{"token": "secret"},
		map[string]any{"ok": true},
	}

	expected := []any{
		map[string]any{"token": redactedValue},
		map[string]any{"ok": true},
	}

	got := RedactValue(input)
	if !reflect.DeepEqual(got, expected) {
		t.Fatalf("expected %#v, got %#v", expected, got)
	}
}

type samplePayload struct {
	Symbol       string         `json:"symbol"`
	APIKey       string         `json:"api_key"`
	OrderRequest map[string]any `json:"order_request"`
}

func TestRedactValue_DecodesStructs(t *testing.T) {
	input := samplePayload{
		Symbol: "MSFT",
		APIKey: "secret",
		OrderRequest: map[string]any{
			"price": 200.0,
		},
	}

	got := RedactValue(input)
	asMap, ok := got.(map[string]any)
	if !ok {
		t.Fatalf("expected map output, got %#v", got)
	}
	if asMap["api_key"] != redactedValue {
		t.Fatalf("expected api_key to be redacted")
	}
	if asMap["order_request"] != redactedValue {
		t.Fatalf("expected order_request to be redacted")
	}
}
