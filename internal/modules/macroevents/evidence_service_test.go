package macroevents

import (
	"context"
	"testing"
)

func TestEvidenceServiceBuildsAndPersistsBundle(t *testing.T) {
	store := &fakeEvidenceStore{}
	service := NewEvidenceService(store)

	bundle, err := service.BuildAndSave(context.Background(), fullEvidenceInput())
	if err != nil {
		t.Fatalf("BuildAndSave returned error: %v", err)
	}

	if bundle.ID == "" {
		t.Fatal("expected persisted bundle id")
	}
	if len(store.saved) != 1 {
		t.Fatalf("saved bundles = %d, want 1", len(store.saved))
	}
}

type fakeEvidenceStore struct {
	saved []EvidenceBundle
}

func (s *fakeEvidenceStore) SaveEvidenceBundle(_ context.Context, bundle EvidenceBundle) (EvidenceBundle, error) {
	bundle.ID = "bundle-1"
	s.saved = append(s.saved, bundle)
	return bundle, nil
}
