package harness

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
)

type ToolDefinition struct {
	Name                 string
	Description          string
	ReadOnly             bool
	InputSchemaHint      string
	OutputKind           string
	EvidenceLevel        EvidenceLevel
	FreshnessExpectation string
	Handler              func(context.Context, json.RawMessage) (json.RawMessage, error)
}

type Registry struct {
	tools map[string]ToolDefinition
}

func NewRegistry() *Registry {
	return &Registry{tools: make(map[string]ToolDefinition)}
}

func (r *Registry) Register(def ToolDefinition) error {
	if def.Name == "" {
		return fmt.Errorf("tool name required")
	}
	if !def.ReadOnly {
		return fmt.Errorf("non-read-only tool refused by harness registry: %s", def.Name)
	}
	if def.Handler == nil {
		return fmt.Errorf("tool handler required: %s", def.Name)
	}
	if _, exists := r.tools[def.Name]; exists {
		return fmt.Errorf("duplicate tool registration: %s", def.Name)
	}
	r.tools[def.Name] = def
	return nil
}

func (r *Registry) Get(name string) (ToolDefinition, bool) {
	def, ok := r.tools[name]
	return def, ok
}

func (r *Registry) AllTools() []ToolDefinition {
	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}
	slices.Sort(names)

	out := make([]ToolDefinition, 0, len(names))
	for _, name := range names {
		out = append(out, r.tools[name])
	}
	return out
}
