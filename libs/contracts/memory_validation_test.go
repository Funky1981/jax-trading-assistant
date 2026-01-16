package contracts

import (
	"testing"
	"time"
)

func TestValidateMemoryItem_Valid(t *testing.T) {
	item := MemoryItem{
		TS:      time.Now().UTC(),
		Type:    "decision",
		Summary: "Entered on earnings gap with tight stop.",
		Tags:    []string{"earnings", "gap"},
		Data:    map[string]any{"confidence": 0.62},
		Source:  &MemorySource{System: "dexter"},
	}

	if err := ValidateMemoryItem(item); err != nil {
		t.Fatalf("expected valid item, got error: %v", err)
	}
}

func TestValidateMemoryItem_MissingRequiredFields(t *testing.T) {
	tests := []struct {
		name string
		item MemoryItem
	}{
		{
			name: "missing type",
			item: MemoryItem{
				TS:      time.Now().UTC(),
				Summary: "summary",
				Data:    map[string]any{"a": 1},
				Source:  &MemorySource{System: "dexter"},
			},
		},
		{
			name: "missing summary",
			item: MemoryItem{
				TS:     time.Now().UTC(),
				Type:   "decision",
				Data:   map[string]any{"a": 1},
				Source: &MemorySource{System: "dexter"},
			},
		},
		{
			name: "missing data",
			item: MemoryItem{
				TS:      time.Now().UTC(),
				Type:    "decision",
				Summary: "summary",
				Source:  &MemorySource{System: "dexter"},
			},
		},
		{
			name: "missing source",
			item: MemoryItem{
				TS:      time.Now().UTC(),
				Type:    "decision",
				Summary: "summary",
				Data:    map[string]any{"a": 1},
			},
		},
	}

	for _, tc := range tests {
		if err := ValidateMemoryItem(tc.item); err == nil {
			t.Fatalf("expected error for %s", tc.name)
		}
	}
}

func TestValidateMemoryItem_InvalidTimestamp(t *testing.T) {
	item := MemoryItem{
		Type:    "decision",
		Summary: "summary",
		Data:    map[string]any{"a": 1},
		Source:  &MemorySource{System: "dexter"},
	}

	if err := ValidateMemoryItem(item); err == nil {
		t.Fatalf("expected error for invalid timestamp")
	}
}

func TestValidateMemoryItem_TagsValidation(t *testing.T) {
	tooMany := make([]string, MaxTags+1)
	for i := 0; i < len(tooMany); i++ {
		tooMany[i] = "tag"
	}

	item := MemoryItem{
		TS:      time.Now().UTC(),
		Type:    "decision",
		Summary: "summary",
		Tags:    []string{"Earnings"},
		Data:    map[string]any{"a": 1},
		Source:  &MemorySource{System: "dexter"},
	}

	if err := ValidateMemoryItem(item); err == nil {
		t.Fatalf("expected error for uppercase tag")
	}

	item.Tags = tooMany
	if err := ValidateMemoryItem(item); err == nil {
		t.Fatalf("expected error for too many tags")
	}
}

func TestNormalizeMemoryTags(t *testing.T) {
	input := []string{" Earnings ", "", "Gap"}
	got := NormalizeMemoryTags(input)
	if len(got) != 2 {
		t.Fatalf("expected 2 tags, got %d", len(got))
	}
	if got[0] != "earnings" || got[1] != "gap" {
		t.Fatalf("unexpected tags: %#v", got)
	}
}
