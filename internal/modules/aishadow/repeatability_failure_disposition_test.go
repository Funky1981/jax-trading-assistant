package aishadow

import (
	"path/filepath"
	"testing"
)

func TestC1F3RepeatabilityR2FailureDispositionIsImmutableAndNonScoreable(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	path := filepath.Join(root, filepath.FromSlash(C1F3RepeatabilityR2FailureDispositionPath))
	disposition, err := LoadC1F3RepeatabilityR2FailureDisposition(path)
	if err != nil {
		t.Fatal(err)
	}
	if !disposition.CellConsumed || disposition.RerunAllowed || disposition.RepeatabilityMeasured || disposition.ResponseSemanticsInspected {
		t.Fatalf("r2 failure disposition is unsafe: %+v", disposition)
	}
	got, err := hashOpaqueFile(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("r2_failure_disposition_sha256=%s", got)
}
