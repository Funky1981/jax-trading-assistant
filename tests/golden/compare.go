package golden

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"

	"os"

	"github.com/google/uuid"
)

// CompareResult represents the outcome of comparing two snapshots
type CompareResult struct {
	Match        bool     `json:"match"`
	Differences  []string `json:"differences,omitempty"`
	ExpectedFile string   `json:"expected_file"`
	ActualFile   string   `json:"actual_file"`
}

// Snapshot represents a captured golden snapshot
type Snapshot struct {
	Captured string                 `json:"captured"`
	Service  string                 `json:"service"`
	Endpoint string                 `json:"endpoint"`
	Method   string                 `json:"method"`
	Request  interface{}            `json:"request,omitempty"`
	Response interface{}            `json:"response"`
	Metadata map[string]interface{} `json:"metadata"`
}

// LoadSnapshot loads a snapshot from a JSON file
func LoadSnapshot(path string) (*Snapshot, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var snapshot Snapshot
	if err := json.NewDecoder(f).Decode(&snapshot); err != nil {
		return nil, err
	}

	return &snapshot, nil
}

// CompareSnapshots compares two snapshots and returns differences
func CompareSnapshots(expected, actual *Snapshot) *CompareResult {
	result := &CompareResult{
		Match:       true,
		Differences: []string{},
	}

	// Compare service
	if expected.Service != actual.Service {
		result.Match = false
		result.Differences = append(result.Differences,
			fmt.Sprintf("Service mismatch: expected %s, got %s", expected.Service, actual.Service))
	}

	// Compare endpoint
	if expected.Endpoint != actual.Endpoint {
		result.Match = false
		result.Differences = append(result.Differences,
			fmt.Sprintf("Endpoint mismatch: expected %s, got %s", expected.Endpoint, actual.Endpoint))
	}

	// Compare response structure (deep comparison)
	if !deepEqual(expected.Response, actual.Response) {
		result.Match = false
		diffs := findDifferences(expected.Response, actual.Response, "response")
		result.Differences = append(result.Differences, diffs...)
	}

	return result
}

// deepEqual performs a deep comparison ignoring timestamps and IDs
func deepEqual(expected, actual interface{}) bool {
	normalizedExpected := normalizeForComparison(expected, "")
	normalizedActual := normalizeForComparison(actual, "")
	return reflect.DeepEqual(normalizedExpected, normalizedActual)
}

// findDifferences recursively finds differences between two values
func findDifferences(expected, actual interface{}, path string) []string {
	differences := []string{}

	switch expVal := expected.(type) {
	case map[string]interface{}:
		actMap, ok := actual.(map[string]interface{})
		if !ok {
			differences = append(differences, fmt.Sprintf("%s: type mismatch (expected map, got %T)", path, actual))
			return differences
		}

		// Check for missing keys
		for key := range expVal {
			if isVolatileFieldName(key) {
				continue
			}
			if _, exists := actMap[key]; !exists {
				differences = append(differences, fmt.Sprintf("%s.%s: key missing in actual", path, key))
			}
		}

		// Check for extra keys
		for key := range actMap {
			if isVolatileFieldName(key) {
				continue
			}
			if _, exists := expVal[key]; !exists {
				differences = append(differences, fmt.Sprintf("%s.%s: unexpected key in actual", path, key))
			}
		}

		// Recursively compare values
		for key, expValue := range expVal {
			if isVolatileFieldName(key) {
				continue
			}
			if actValue, exists := actMap[key]; exists {
				subDiffs := findDifferences(expValue, actValue, fmt.Sprintf("%s.%s", path, key))
				differences = append(differences, subDiffs...)
			}
		}

	case []interface{}:
		actSlice, ok := actual.([]interface{})
		if !ok {
			differences = append(differences, fmt.Sprintf("%s: type mismatch (expected slice, got %T)", path, actual))
			return differences
		}

		if len(expVal) != len(actSlice) {
			differences = append(differences, fmt.Sprintf("%s: length mismatch (expected %d, got %d)", path, len(expVal), len(actSlice)))
			// Still compare elements up to shorter length
		}

		minLen := len(expVal)
		if len(actSlice) < minLen {
			minLen = len(actSlice)
		}

		for i := 0; i < minLen; i++ {
			subDiffs := findDifferences(expVal[i], actSlice[i], fmt.Sprintf("%s[%d]", path, i))
			differences = append(differences, subDiffs...)
		}

	default:
		// Primitive types - direct comparison
		if !reflect.DeepEqual(expected, actual) {
			differences = append(differences, fmt.Sprintf("%s: value mismatch (expected %v, got %v)", path, expected, actual))
		}
	}

	return differences
}

// ShouldIgnoreField determines if a field should be ignored in comparison
func ShouldIgnoreField(fieldName string) bool {
	ignoredFields := map[string]bool{
		"created_at": true,
		"updated_at": true,
		"timestamp":  true,
		"captured":   true,
		"id":         true, // UUIDs will differ
		"run_id":     true,
		"request_id": true,
	}

	return ignoredFields[fieldName]
}

func normalizeForComparison(value interface{}, fieldName string) interface{} {
	if isVolatileFieldName(fieldName) {
		return "<ignored>"
	}

	switch v := value.(type) {
	case map[string]interface{}:
		out := make(map[string]interface{}, len(v))
		for key, child := range v {
			out[key] = normalizeForComparison(child, key)
		}
		return out
	case []interface{}:
		out := make([]interface{}, len(v))
		for i := range v {
			out[i] = normalizeForComparison(v[i], fieldName)
		}
		return out
	case string:
		if looksLikeVolatileValue(v) {
			return "<ignored>"
		}
		return v
	default:
		return v
	}
}

func isVolatileFieldName(fieldName string) bool {
	name := strings.ToLower(strings.TrimSpace(fieldName))
	if name == "" {
		return false
	}
	if ShouldIgnoreField(name) {
		return true
	}
	return strings.HasSuffix(name, "_at") ||
		strings.HasSuffix(name, "_time") ||
		strings.HasSuffix(name, "_timestamp") ||
		strings.HasSuffix(name, "_uuid")
}

func looksLikeVolatileValue(value string) bool {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return false
	}
	if _, err := uuid.Parse(trimmed); err == nil {
		return true
	}
	if _, err := time.Parse(time.RFC3339, trimmed); err == nil {
		return true
	}
	if _, err := time.Parse(time.RFC3339Nano, trimmed); err == nil {
		return true
	}
	return false
}
