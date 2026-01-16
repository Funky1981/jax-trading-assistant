package contracts

import (
	"errors"
	"strings"
)

const (
	MinSummaryLength = 1
	MaxSummaryLength = 500
	MaxTags          = 10
)

var (
	ErrMemoryItemNilSource      = errors.New("memory item source required")
	ErrMemoryItemMissingType    = errors.New("memory item type required")
	ErrMemoryItemMissingTS      = errors.New("memory item timestamp required")
	ErrMemoryItemMissingData    = errors.New("memory item data required")
	ErrMemoryItemInvalidTag     = errors.New("memory item tags must be lowercase and non-empty")
	ErrMemoryItemTooManyTags    = errors.New("memory item has too many tags")
	ErrMemoryItemInvalidSummary = errors.New("memory item summary length invalid")
)

// NormalizeMemoryTags lowercases and trims tags, dropping empty tags.
func NormalizeMemoryTags(tags []string) []string {
	if len(tags) == 0 {
		return nil
	}
	out := make([]string, 0, len(tags))
	for _, tag := range tags {
		clean := strings.ToLower(strings.TrimSpace(tag))
		if clean == "" {
			continue
		}
		out = append(out, clean)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// ValidateMemoryItem enforces required fields and basic schema constraints.
func ValidateMemoryItem(item MemoryItem) error {
	if item.TS.IsZero() {
		return ErrMemoryItemMissingTS
	}
	if strings.TrimSpace(item.Type) == "" {
		return ErrMemoryItemMissingType
	}
	summary := strings.TrimSpace(item.Summary)
	if len(summary) < MinSummaryLength || len(summary) > MaxSummaryLength {
		return ErrMemoryItemInvalidSummary
	}
	if item.Data == nil {
		return ErrMemoryItemMissingData
	}
	if item.Source == nil || strings.TrimSpace(item.Source.System) == "" {
		return ErrMemoryItemNilSource
	}
	if len(item.Tags) > MaxTags {
		return ErrMemoryItemTooManyTags
	}
	for _, tag := range item.Tags {
		if tag == "" || tag != strings.ToLower(tag) {
			return ErrMemoryItemInvalidTag
		}
	}
	return nil
}
