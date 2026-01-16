package contracts

import (
	"context"
	"time"
)

type MemoryID string

type MemorySource struct {
	System string `json:"system"`
	Ref    string `json:"ref,omitempty"`
}

type MemoryItem struct {
	ID      string         `json:"id,omitempty"`
	TS      time.Time      `json:"ts"`
	Type    string         `json:"type"`
	Symbol  string         `json:"symbol,omitempty"`
	Tags    []string       `json:"tags,omitempty"`
	Summary string         `json:"summary"`
	Data    map[string]any `json:"data,omitempty"`
	Source  *MemorySource  `json:"source,omitempty"`
}

type MemoryQuery struct {
	Q      string     `json:"q,omitempty"`
	Symbol string     `json:"symbol,omitempty"`
	Types  []string   `json:"types,omitempty"`
	From   *time.Time `json:"from,omitempty"`
	To     *time.Time `json:"to,omitempty"`
	Tags   []string   `json:"tags,omitempty"`
	Limit  int        `json:"limit,omitempty"`
}

type ReflectionParams struct {
	Query      string `json:"query"`
	WindowDays int    `json:"window_days,omitempty"`
	PromptHint string `json:"prompt_hint,omitempty"`
}

type MemoryStore interface {
	Ping(ctx context.Context) error
	Retain(ctx context.Context, bank string, item MemoryItem) (MemoryID, error)
	Recall(ctx context.Context, bank string, query MemoryQuery) ([]MemoryItem, error)
	Reflect(ctx context.Context, bank string, params ReflectionParams) ([]MemoryItem, error)
}
