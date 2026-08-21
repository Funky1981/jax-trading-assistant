package canonical

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
)

// EncodeJSON validates a contract before producing its deterministic JSON
// representation. Canonical contracts contain no maps; struct field order and
// array order therefore determine stable output bytes.
func EncodeJSON(contract Contract) ([]byte, error) {
	if isNilContract(contract) {
		return nil, fmt.Errorf("canonical JSON encode: contract is nil")
	}
	if err := contract.Validate(); err != nil {
		return nil, fmt.Errorf("canonical JSON encode: %w", err)
	}
	data, err := json.Marshal(contract)
	if err != nil {
		return nil, fmt.Errorf("canonical JSON encode: %w", err)
	}
	return data, nil
}

// DecodeJSON rejects unknown fields, trailing JSON values, unsupported
// versions, and invalid contract states. destination must be a non-nil pointer
// to one of the canonical contract types.
func DecodeJSON(data []byte, destination Contract) error {
	if isNilContract(destination) {
		return fmt.Errorf("canonical JSON decode: destination is nil")
	}
	value := reflect.ValueOf(destination)
	if value.Kind() != reflect.Pointer {
		return fmt.Errorf("canonical JSON decode: destination must be a pointer")
	}

	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return fmt.Errorf("canonical JSON decode: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return fmt.Errorf("canonical JSON decode: trailing JSON value")
		}
		return fmt.Errorf("canonical JSON decode: trailing data: %w", err)
	}
	if err := destination.Validate(); err != nil {
		return fmt.Errorf("canonical JSON decode: %w", err)
	}
	return nil
}

func isNilContract(contract Contract) bool {
	if contract == nil {
		return true
	}
	value := reflect.ValueOf(contract)
	return value.Kind() == reflect.Pointer && value.IsNil()
}
