package harness

import (
	"context"
	"testing"
	"time"
)

func memoryKey() ResearchValidityKey {
	return ResearchValidityKey{ContractVersion: ResearchMemoryContractV1, TaskType: "market_context", ObjectiveFingerprint: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", EvidenceFingerprint: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", SourceVintages: []string{"source-2026-09-07"}, ToolVersions: []string{"research.market_snapshot/v1"}, PromptSystemVersion: "prompt-v1", OutputContract: "jax.research_report/v1", Provider: "jax-fixture", Model: "local-fixture", ResearchContractVersion: "research-v1", PolicyVersion: "policy-v1", QuantVersions: []string{"quant-v1"}, RuntimeOptionsFingerprint: "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"}
}

func TestExactResearchMemoryRequiresCompleteValidityAndPreservesProvenance(t *testing.T) {
	now := time.Date(2026, 9, 7, 12, 0, 0, 0, time.UTC)
	entry, err := NewResearchMemoryEntry(memoryKey(), MemoryCurrentConclusion, MemoryStatusValid, "Evidence-linked provisional conclusion", []string{"evd_fixture"}, []string{"pvn_fixture"}, now, now.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	store := NewExactResearchMemory()
	if err := store.Put(context.Background(), entry); err != nil {
		t.Fatal(err)
	}
	result, err := store.GetExact(context.Background(), memoryKey(), now.Add(time.Minute))
	if err != nil || result.Label != "EXACT_REUSE" || result.Entry.ID != entry.ID || result.Entry.EvidenceIDs[0] != "evd_fixture" {
		t.Fatalf("exact provenance-safe reuse failed: %#v err=%v", result, err)
	}
	if SemanticResultCacheEnabled {
		t.Fatal("semantic result cache must remain disabled")
	}
}

func TestExactResearchMemoryInvalidatesMaterialChangesAndStaleState(t *testing.T) {
	now := time.Date(2026, 9, 7, 12, 0, 0, 0, time.UTC)
	entry, err := NewResearchMemoryEntry(memoryKey(), MemoryCurrentConclusion, MemoryStatusValid, "conclusion", []string{"evd_fixture"}, []string{"pvn_fixture"}, now, now.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	store := NewExactResearchMemory()
	if err := store.Put(context.Background(), entry); err != nil {
		t.Fatal(err)
	}
	changed := memoryKey()
	changed.Model = "different-model"
	if _, err := store.GetExact(context.Background(), changed, now.Add(time.Minute)); err == nil {
		t.Fatal("expected model validity-key miss")
	}
	digest := entry.ValidityDigest
	if err := store.Invalidate(context.Background(), digest); err != nil {
		t.Fatal(err)
	}
	if stored := store.entries[digest]; stored.Validate() != nil || stored.Status != MemoryStatusInvalidated {
		t.Fatal("invalidated memory must remain integrity-valid with a preserved immutable identity")
	}
	if _, err := store.GetExact(context.Background(), memoryKey(), now.Add(time.Minute)); err == nil {
		t.Fatal("expected invalidated memory rejection")
	}
	entry, err = NewResearchMemoryEntry(memoryKey(), MemoryCurrentConclusion, MemoryStatusValid, "stale", []string{"evd_fixture"}, []string{"pvn_fixture"}, now, now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	store = NewExactResearchMemory()
	if err := store.Put(context.Background(), entry); err != nil {
		t.Fatal(err)
	}
	if _, err := store.GetExact(context.Background(), memoryKey(), now.Add(2*time.Minute)); err == nil {
		t.Fatal("expected stale memory rejection")
	}
}

func TestResearchMemoryRejectsConclusionWithoutEvidenceOrSupersededReuse(t *testing.T) {
	now := time.Date(2026, 9, 7, 12, 0, 0, 0, time.UTC)
	if _, err := NewResearchMemoryEntry(memoryKey(), MemoryCurrentConclusion, MemoryStatusValid, "conclusion", nil, []string{"pvn_fixture"}, now, now.Add(time.Hour)); err == nil {
		t.Fatal("expected evidence provenance requirement")
	}
	entry, err := NewResearchMemoryEntry(memoryKey(), MemorySupersededConclusion, MemoryStatusSuperseded, "old conclusion", []string{"evd_fixture"}, []string{"pvn_fixture"}, now, now.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	store := NewExactResearchMemory()
	if err := store.Put(context.Background(), entry); err != nil {
		t.Fatal(err)
	}
	if _, err := store.GetExact(context.Background(), memoryKey(), now.Add(time.Minute)); err == nil {
		t.Fatal("expected superseded conclusion rejection")
	}
}
