package worldmonitorintelligence

import "testing"

func TestValidatePageAcceptsEmptyAndOrderedPages(t *testing.T) {
	if err := ValidatePage(nil, "12", 12, 100); err != nil {
		t.Fatalf("empty page: %v", err)
	}
	if err := ValidatePage([]CursorEvent{{EventID: "evt-a", PersistenceSeq: "13"}, {EventID: "evt-b", PersistenceSeq: "14"}}, "14", 12, 100); err != nil {
		t.Fatalf("ordered page: %v", err)
	}
}

func TestValidatePageRejectsCursorAndIdentityFailures(t *testing.T) {
	cases := []struct {
		name     string
		events   []CursorEvent
		next     string
		after    int64
		pageSize int
	}{
		{name: "empty event id", events: []CursorEvent{{PersistenceSeq: "1"}}, next: "1", after: 0, pageSize: 10},
		{name: "duplicate sequence", events: []CursorEvent{{EventID: "a", PersistenceSeq: "1"}, {EventID: "b", PersistenceSeq: "1"}}, next: "1", after: 0, pageSize: 10},
		{name: "cursor skips page", events: []CursorEvent{{EventID: "a", PersistenceSeq: "1"}}, next: "2", after: 0, pageSize: 10},
		{name: "empty cursor advances", events: nil, next: "1", after: 0, pageSize: 10},
		{name: "page exceeds bound", events: []CursorEvent{{EventID: "a", PersistenceSeq: "1"}}, next: "1", after: 0, pageSize: 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := ValidatePage(tc.events, tc.next, tc.after, tc.pageSize); err == nil {
				t.Fatal("expected validation failure")
			}
		})
	}
}
