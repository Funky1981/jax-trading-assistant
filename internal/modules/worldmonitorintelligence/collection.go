package worldmonitorintelligence

import (
	"fmt"
	"strconv"
	"strings"
)

// CursorEvent is the provider identity needed to validate an ordered page.
// Provider-specific fields stay outside this package and cannot become a
// canonical event accidentally.
type CursorEvent struct {
	EventID        string
	PersistenceSeq string
}

// ValidatePage verifies page bounds, event identity, strict sequence order,
// and the provider's next-cursor contract before a page is committed.
func ValidatePage(events []CursorEvent, nextCursor string, after int64, pageSize int) error {
	if pageSize < 1 || len(events) > pageSize {
		return fmt.Errorf("invalid page size: events=%d page_size=%d", len(events), pageSize)
	}
	previous := after
	for index, event := range events {
		if strings.TrimSpace(event.EventID) == "" {
			return fmt.Errorf("event %d has an empty provider event ID", index)
		}
		sequence, err := strconv.ParseInt(strings.TrimSpace(event.PersistenceSeq), 10, 64)
		if err != nil || sequence <= previous {
			return fmt.Errorf("event %d has invalid or non-monotonic persistence sequence %q", index, event.PersistenceSeq)
		}
		previous = sequence
	}
	next, err := strconv.ParseInt(strings.TrimSpace(nextCursor), 10, 64)
	if err != nil || next < after || (len(events) > 0 && next != previous) || (len(events) == 0 && next != after) {
		return fmt.Errorf("invalid next cursor %q after=%d", nextCursor, after)
	}
	return nil
}
