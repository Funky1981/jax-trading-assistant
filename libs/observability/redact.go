package observability

import (
	"encoding/json"
	"strings"
)

const redactedValue = "[REDACTED]"

func RedactValue(value any) any {
	if value == nil {
		return nil
	}
	switch typed := value.(type) {
	case map[string]any:
		return redactMap(typed)
	case []any:
		return redactSlice(typed)
	case string, bool,
		int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64,
		float32, float64,
		json.Number:
		return typed
	default:
		decoded, ok := decodeToInterface(value)
		if ok {
			return RedactValue(decoded)
		}
		return value
	}
}

func redactMap(input map[string]any) map[string]any {
	out := make(map[string]any, len(input))
	for key, value := range input {
		if isSensitiveKey(key) {
			out[key] = redactedValue
			continue
		}
		switch typed := value.(type) {
		case map[string]any:
			out[key] = redactMap(typed)
		case []any:
			out[key] = redactSlice(typed)
		default:
			out[key] = RedactValue(typed)
		}
	}
	return out
}

func redactSlice(input []any) []any {
	out := make([]any, len(input))
	for i, value := range input {
		out[i] = RedactValue(value)
	}
	return out
}

func decodeToInterface(value any) (any, bool) {
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, false
	}
	var decoded any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return nil, false
	}
	return decoded, true
}

func isSensitiveKey(key string) bool {
	if key == "" {
		return false
	}
	normalized := strings.ToLower(strings.TrimSpace(key))
	switch normalized {
	case "order_payload", "order_request", "raw_order":
		return true
	case "account_id", "accountid", "account-id", "acct_id":
		return true
	}
	if strings.Contains(normalized, "password") {
		return true
	}
	if strings.Contains(normalized, "secret") {
		return true
	}
	if strings.Contains(normalized, "token") {
		return true
	}
	if strings.Contains(normalized, "api_key") || strings.Contains(normalized, "apikey") {
		return true
	}
	if strings.Contains(normalized, "credential") {
		return true
	}
	if strings.Contains(normalized, "broker") && strings.Contains(normalized, "key") {
		return true
	}
	return false
}
