package agent0

import (
	"context"
	"testing"
)

func TestMockClient_DefaultPlan(t *testing.T) {
	client := &MockClient{}
	got, err := client.Plan(context.Background(), PlanRequest{Symbol: "AAPL"})
	if err != nil {
		t.Fatalf("Plan returned error: %v", err)
	}
	if got.Action != "HOLD" {
		t.Fatalf("expected HOLD action, got %q", got.Action)
	}
	if got.Confidence <= 0 {
		t.Fatalf("expected positive confidence, got %f", got.Confidence)
	}
}

func TestMockClient_DefaultExecute(t *testing.T) {
	client := &MockClient{}
	got, err := client.Execute(context.Background(), ExecuteRequest{})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if !got.Success {
		t.Fatal("expected success=true")
	}
}
