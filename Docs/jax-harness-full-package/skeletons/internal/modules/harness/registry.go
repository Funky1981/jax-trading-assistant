package harness

import (
    "context"
    "encoding/json"
    "fmt"
)

type EvidenceLevel string

const (
    EvidenceHardInternal    EvidenceLevel = "hard_internal_data"
    EvidenceDerivedInternal EvidenceLevel = "derived_internal_data"
    EvidenceWeakInference   EvidenceLevel = "weak_inference"
)

type ToolDefinition struct {
    Name                string
    Description         string
    ReadOnly            bool
    InputSchemaHint     string
    OutputKind          string
    EvidenceLevel       EvidenceLevel
    FreshnessExpectation string
    Handler             func(context.Context, json.RawMessage) (json.RawMessage, error)
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
    r.tools[def.Name] = def
    return nil
}

func (r *Registry) Get(name string) (ToolDefinition, bool) {
    def, ok := r.tools[name]
    return def, ok
}

func (r *Registry) List() []ToolDefinition {
    out := make([]ToolDefinition, 0, len(r.tools))
    for _, t := range r.tools {
        out = append(out, t)
    }
    return out
}
