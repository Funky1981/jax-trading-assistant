package planner

import (
	"context"
	"errors"
	"strings"
	"testing"
)

type fakeProvider struct {
	result Result
	err    error
}

func (p fakeProvider) Generate(context.Context, Request) (Result, error) {
	return p.result, p.err
}

func validResult() Result {
	return Result{Summary: "bounded", Steps: []string{"review"}, Action: "HOLD", Confidence: 0.5}
}

func TestServiceValidatesProviderOutput(t *testing.T) {
	service, err := NewService(fakeProvider{result: validResult()})
	if err != nil {
		t.Fatal(err)
	}
	result, err := service.Plan(context.Background(), Request{Symbol: "AAPL"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Action != "HOLD" {
		t.Fatalf("unexpected action %q", result.Action)
	}
}

func TestServiceRejectsExecutionAuthorityLabels(t *testing.T) {
	for _, action := range []string{"APPROVE", "EXECUTE", "ORDER", "BROKER", "SUBMIT", "TRADE"} {
		service, err := NewService(fakeProvider{result: Result{Summary: "bounded", Action: action, Confidence: 0.5}})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := service.Plan(context.Background(), Request{Symbol: "AAPL"}); !errors.Is(err, ErrInvalidResult) {
			t.Fatalf("action %q was not rejected: %v", action, err)
		}
	}
}

func TestServiceRejectsInvalidConfidenceAndUnboundedSteps(t *testing.T) {
	for _, result := range []Result{
		{Summary: "bounded", Action: "HOLD", Confidence: 1.1},
		{Summary: "bounded", Action: "HOLD", Confidence: 0.5, Steps: make([]string, maxSteps+1)},
	} {
		service, err := NewService(fakeProvider{result: result})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := service.Plan(context.Background(), Request{Symbol: "AAPL"}); !errors.Is(err, ErrInvalidResult) {
			t.Fatalf("invalid result was accepted: %v", err)
		}
	}
}

func TestServiceHonorsCancellation(t *testing.T) {
	service, err := NewService(fakeProvider{result: validResult()})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := service.Plan(ctx, Request{Symbol: "AAPL"}); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected cancellation, got %v", err)
	}
}

func TestDeterministicProviderIsAdvisory(t *testing.T) {
	result, err := (DeterministicProvider{}).Generate(context.Background(), Request{Symbol: " aapl "})
	if err != nil {
		t.Fatal(err)
	}
	if result.Action != "HOLD" || !strings.Contains(result.Summary, "AAPL") {
		t.Fatalf("unexpected deterministic result: %+v", result)
	}
	if err := ValidateResult(result); err != nil {
		t.Fatal(err)
	}
}
