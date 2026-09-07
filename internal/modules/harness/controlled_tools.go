package harness

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"regexp"
	"sort"
	"strings"
	"time"
)

const ControlledToolRegistryContractV1 = "jax.controlled_tool_registry/v1"

const (
	ToolPermissionEvidenceRead = "EVIDENCE_READ"
	ToolPermissionDerivedRead  = "DERIVED_READ"
	ToolPermissionMemoryRead   = "MEMORY_READ"
	ToolCostFree               = "FREE"
	ToolCostModelTokens        = "MODEL_TOKENS"
	ToolProvenanceRequired     = "REQUIRED"
)

var ErrControlledUnknownTool = errors.New("controlled tool is not registered")

type JSONSchema struct {
	Type                 string                  `json:"type"`
	Properties           map[string]JSONProperty `json:"properties"`
	Required             []string                `json:"required"`
	AdditionalProperties bool                    `json:"additional_properties"`
	MaxProperties        int                     `json:"max_properties"`
}

type JSONProperty struct {
	Type      string        `json:"type"`
	Enum      []string      `json:"enum,omitempty"`
	Pattern   string        `json:"pattern,omitempty"`
	Format    string        `json:"format,omitempty"`
	MaxLength int           `json:"max_length,omitempty"`
	MaxItems  int           `json:"max_items,omitempty"`
	Minimum   *float64      `json:"minimum,omitempty"`
	Maximum   *float64      `json:"maximum,omitempty"`
	Items     *JSONProperty `json:"items,omitempty"`
}

func (schema JSONSchema) Validate() error {
	if schema.Type != "object" || len(schema.Properties) == 0 || !schemaStringList(schema.Required) || schema.MaxProperties <= 0 {
		return fmt.Errorf("tool input schema must be a bounded object with properties and sorted required fields")
	}
	for name, property := range schema.Properties {
		if strings.TrimSpace(name) == "" {
			return fmt.Errorf("tool schema contains an empty property name")
		}
		if err := property.validate(); err != nil {
			return fmt.Errorf("property %s: %w", name, err)
		}
	}
	for _, required := range schema.Required {
		if _, ok := schema.Properties[required]; !ok {
			return fmt.Errorf("required property %q is not declared", required)
		}
	}
	return nil
}

func (property JSONProperty) validate() error {
	switch property.Type {
	case "string", "integer", "number", "boolean", "array":
	default:
		return fmt.Errorf("unsupported JSON property type %q", property.Type)
	}
	if property.MaxLength < 0 || property.MaxItems < 0 || property.Type != "string" && (property.MaxLength > 0 || property.Pattern != "" || property.Format != "") || property.Type != "array" && property.MaxItems > 0 {
		return fmt.Errorf("property constraints do not match type")
	}
	if property.Minimum != nil && (!finiteNumber(*property.Minimum) || property.Maximum != nil && *property.Minimum > *property.Maximum) || property.Maximum != nil && !finiteNumber(*property.Maximum) {
		return fmt.Errorf("numeric bounds are invalid")
	}
	if property.Pattern != "" {
		if _, err := regexp.Compile(property.Pattern); err != nil {
			return fmt.Errorf("invalid pattern: %w", err)
		}
	}
	if property.Type == "array" && property.Items == nil {
		return fmt.Errorf("array property requires item schema")
	}
	return nil
}

type ControlledToolSpec struct {
	ID                  string                `json:"id"`
	Version             string                `json:"version"`
	Description         string                `json:"description"`
	InputSchema         JSONSchema            `json:"input_schema"`
	OutputContract      string                `json:"output_contract"`
	PermissionTier      string                `json:"permission_tier"`
	Timeout             time.Duration         `json:"timeout"`
	CostClass           string                `json:"cost_class"`
	Provider            string                `json:"provider"`
	ProvenanceBehaviour string                `json:"provenance_behaviour"`
	Deterministic       bool                  `json:"deterministic"`
	ReadOnly            bool                  `json:"read_only"`
	Handler             ControlledToolHandler `json:"-"`
}

type ControlledToolHandler func(context.Context, json.RawMessage) (json.RawMessage, error)

func (spec ControlledToolSpec) Validate() error {
	if !validControlledToolID(spec.ID) || strings.TrimSpace(spec.Version) == "" || strings.TrimSpace(spec.Description) == "" || strings.TrimSpace(spec.OutputContract) == "" || spec.Timeout <= 0 || spec.Timeout > 5*time.Minute || strings.TrimSpace(spec.Provider) == "" || spec.ProvenanceBehaviour != ToolProvenanceRequired || !spec.ReadOnly || spec.Handler == nil {
		return fmt.Errorf("controlled tool requires bounded versioned read-only identity, handler and provenance")
	}
	switch spec.PermissionTier {
	case ToolPermissionEvidenceRead, ToolPermissionDerivedRead, ToolPermissionMemoryRead:
	default:
		return fmt.Errorf("unsupported controlled tool permission tier %q", spec.PermissionTier)
	}
	switch spec.CostClass {
	case ToolCostFree, ToolCostModelTokens:
	default:
		return fmt.Errorf("unsupported controlled tool cost class %q", spec.CostClass)
	}
	return spec.InputSchema.Validate()
}

type ControlledToolRequest struct {
	ToolID  string          `json:"tool_id"`
	Version string          `json:"version"`
	RunID   string          `json:"run_id"`
	StepID  string          `json:"step_id"`
	Args    json.RawMessage `json:"args"`
}

func (request ControlledToolRequest) Validate(registry *ControlledToolRegistry) (ControlledToolSpec, error) {
	if registry == nil {
		return ControlledToolSpec{}, fmt.Errorf("controlled tool registry is required")
	}
	if !validControlledToolID(request.ToolID) || strings.TrimSpace(request.Version) == "" || !validControlledID(request.RunID) || !validControlledID(request.StepID) {
		return ControlledToolSpec{}, fmt.Errorf("tool request requires valid tool, version, run and step identities")
	}
	spec, ok := registry.Get(request.ToolID)
	if !ok {
		return ControlledToolSpec{}, fmt.Errorf("%w: %s", ErrControlledUnknownTool, request.ToolID)
	}
	if request.Version != spec.Version {
		return ControlledToolSpec{}, fmt.Errorf("unsupported tool version %s/%s", request.ToolID, request.Version)
	}
	if err := validateJSONArguments(spec.InputSchema, request.Args); err != nil {
		return ControlledToolSpec{}, err
	}
	return spec, nil
}

type ControlledToolRegistry struct {
	ContractVersion string
	tools           map[string]ControlledToolSpec
}

func NewControlledToolRegistry() *ControlledToolRegistry {
	return &ControlledToolRegistry{ContractVersion: ControlledToolRegistryContractV1, tools: map[string]ControlledToolSpec{}}
}

func (registry *ControlledToolRegistry) Register(spec ControlledToolSpec) error {
	if registry == nil || registry.ContractVersion != ControlledToolRegistryContractV1 {
		return fmt.Errorf("controlled tool registry contract is invalid")
	}
	if err := spec.Validate(); err != nil {
		return err
	}
	if _, exists := registry.tools[spec.ID]; exists {
		return fmt.Errorf("duplicate controlled tool: %s", spec.ID)
	}
	registry.tools[spec.ID] = spec
	return nil
}

func (registry *ControlledToolRegistry) Get(toolID string) (ControlledToolSpec, bool) {
	if registry == nil {
		return ControlledToolSpec{}, false
	}
	spec, ok := registry.tools[toolID]
	return spec, ok
}

func (registry *ControlledToolRegistry) All() []ControlledToolSpec {
	if registry == nil {
		return nil
	}
	ids := make([]string, 0, len(registry.tools))
	for id := range registry.tools {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	result := make([]ControlledToolSpec, 0, len(ids))
	for _, id := range ids {
		result = append(result, registry.tools[id])
	}
	return result
}

func validateJSONArguments(schema JSONSchema, raw json.RawMessage) error {
	if len(raw) == 0 || len(raw) > 64*1024 {
		return fmt.Errorf("tool arguments are missing or oversized")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return fmt.Errorf("tool arguments are not valid JSON: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return fmt.Errorf("tool arguments contain trailing JSON")
	}
	return validateJSONObject(schema, value)
}

func validateJSONObject(schema JSONSchema, value any) error {
	object, ok := value.(map[string]any)
	if !ok {
		return fmt.Errorf("tool arguments must be a JSON object")
	}
	if len(object) > schema.MaxProperties {
		return fmt.Errorf("tool arguments exceed maximum property count")
	}
	for _, required := range schema.Required {
		if _, ok := object[required]; !ok {
			return fmt.Errorf("tool arguments missing required field %q", required)
		}
	}
	for name, value := range object {
		property, declared := schema.Properties[name]
		if !declared {
			if !schema.AdditionalProperties {
				return fmt.Errorf("tool arguments contain unknown field %q", name)
			}
			continue
		}
		if err := validateJSONProperty(property, value); err != nil {
			return fmt.Errorf("tool argument %q: %w", name, err)
		}
	}
	return nil
}

func validateJSONProperty(schema JSONProperty, value any) error {
	switch schema.Type {
	case "string":
		text, ok := value.(string)
		if !ok || schema.MaxLength > 0 && len([]rune(text)) > schema.MaxLength || schema.Pattern != "" && !regexp.MustCompile(schema.Pattern).MatchString(text) {
			return fmt.Errorf("expected bounded string")
		}
		if len(schema.Enum) > 0 && !containsExact(schema.Enum, text) {
			return fmt.Errorf("value is not an allowed enum")
		}
		if schema.Format == "identifier" && !validArgumentIdentifier(text) || schema.Format == "utc-date-time" && !validUTCDateTime(text) {
			return fmt.Errorf("value has invalid format")
		}
	case "integer":
		number, err := parseJSONNumber(value)
		if err != nil || number != math.Trunc(number) || !withinNumberBounds(schema, number) {
			return fmt.Errorf("expected bounded integer")
		}
	case "number":
		number, err := parseJSONNumber(value)
		if err != nil || !withinNumberBounds(schema, number) {
			return fmt.Errorf("expected bounded number")
		}
	case "boolean":
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("expected boolean")
		}
	case "array":
		values, ok := value.([]any)
		if !ok || schema.MaxItems > 0 && len(values) > schema.MaxItems {
			return fmt.Errorf("expected bounded array")
		}
		for _, item := range values {
			if err := validateJSONProperty(*schema.Items, item); err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("unsupported JSON property type")
	}
	return nil
}

func parseJSONNumber(value any) (float64, error) {
	number, ok := value.(json.Number)
	if !ok {
		return 0, fmt.Errorf("not a JSON number")
	}
	parsed, err := number.Float64()
	if err != nil || !finiteNumber(parsed) {
		return 0, fmt.Errorf("number is not finite")
	}
	return parsed, nil
}

func withinNumberBounds(schema JSONProperty, value float64) bool {
	return (schema.Minimum == nil || value >= *schema.Minimum) && (schema.Maximum == nil || value <= *schema.Maximum)
}

func validUTCDateTime(value string) bool {
	parsed, err := time.Parse(time.RFC3339, value)
	return err == nil && parsed.Location() == time.UTC
}

func validControlledToolID(value string) bool {
	if len(value) < 3 || len(value) > 128 {
		return false
	}
	for index, character := range value {
		if index == 0 && (character < 'a' || character > 'z') {
			return false
		}
		if (character < 'a' || character > 'z') && (character < '0' || character > '9') && character != '.' && character != '_' && character != '-' {
			return false
		}
	}
	return true
}

func validControlledID(value string) bool {
	return validControlledToolID(value)
}

func validArgumentIdentifier(value string) bool {
	if len(value) == 0 || len(value) > 64 {
		return false
	}
	for _, character := range value {
		if (character < 'a' || character > 'z') && (character < 'A' || character > 'Z') && (character < '0' || character > '9') && character != '.' && character != '_' && character != '-' {
			return false
		}
	}
	return true
}

func finiteNumber(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}

func containsExact(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func schemaStringList(values []string) bool {
	if len(values) == 0 {
		return false
	}
	for index, value := range values {
		if strings.TrimSpace(value) == "" || index > 0 && values[index-1] >= value {
			return false
		}
	}
	return true
}
