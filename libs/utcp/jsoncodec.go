package utcp

import (
	"encoding/json"
	"fmt"
)

func decodeJSONLike(input any, output any) error {
	raw, err := json.Marshal(input)
	if err != nil {
		return fmt.Errorf("marshal input: %w", err)
	}
	if err := json.Unmarshal(raw, output); err != nil {
		return fmt.Errorf("unmarshal input: %w", err)
	}
	return nil
}
