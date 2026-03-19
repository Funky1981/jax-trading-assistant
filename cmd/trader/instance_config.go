package main

import (
	"encoding/json"
	"fmt"
	"strings"
)

// normalizeInstanceConfig canonicalizes legacy instance config shapes into the
// runtime contract expected by watcher, execution, and frontend APIs.
func normalizeInstanceConfig(raw json.RawMessage) (json.RawMessage, error) {
	if len(raw) == 0 {
		raw = json.RawMessage(`{}`)
	}

	var cfg map[string]any
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return nil, fmt.Errorf("invalid configJson: %w", err)
	}

	symbols := normalizedSymbolList(cfg)
	if len(symbols) > 0 {
		cfg["symbols"] = symbols
	}
	delete(cfg, "universe")

	out, err := json.Marshal(cfg)
	if err != nil {
		return nil, fmt.Errorf("marshal configJson: %w", err)
	}
	return json.RawMessage(out), nil
}

func normalizedSymbolList(cfg map[string]any) []string {
	candidates := []any{cfg["symbols"], cfg["universe"]}
	seen := map[string]struct{}{}
	out := make([]string, 0, 8)
	for _, candidate := range candidates {
		switch values := candidate.(type) {
		case []any:
			for _, value := range values {
				if sym, ok := normalizeSymbolValue(value); ok {
					if _, exists := seen[sym]; exists {
						continue
					}
					seen[sym] = struct{}{}
					out = append(out, sym)
				}
			}
		case []string:
			for _, value := range values {
				if sym, ok := normalizeSymbolValue(value); ok {
					if _, exists := seen[sym]; exists {
						continue
					}
					seen[sym] = struct{}{}
					out = append(out, sym)
				}
			}
		}
	}
	return out
}

func normalizedSymbolListFromRaw(raw json.RawMessage) []string {
	var cfg map[string]any
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return nil
	}
	return normalizedSymbolList(cfg)
}

func normalizeSymbolValue(value any) (string, bool) {
	raw, ok := value.(string)
	if !ok {
		return "", false
	}
	symbol := strings.ToUpper(strings.TrimSpace(raw))
	if symbol == "" {
		return "", false
	}
	return symbol, true
}
