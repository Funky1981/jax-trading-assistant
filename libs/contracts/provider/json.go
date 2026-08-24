package provider

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"unicode/utf8"
)

// EncodeDefinitionJSON validates before producing stable struct-ordered JSON.
func EncodeDefinitionJSON(definition ProviderDefinition) ([]byte, error) {
	if err := definition.Validate(); err != nil {
		return nil, fmt.Errorf("provider JSON encode: %w", err)
	}
	raw, err := json.Marshal(definition)
	if err != nil {
		return nil, fmt.Errorf("provider JSON encode: %w", err)
	}
	return raw, nil
}

// DecodeDefinitionJSON rejects unknown fields (including attempted credential
// material), trailing values, invalid UTF-8, unsupported versions, and invalid
// definitions. Registry contracts contain no secret-bearing extension map.
func DecodeDefinitionJSON(data []byte, destination *ProviderDefinition) error {
	if destination == nil {
		return fmt.Errorf("provider JSON decode: destination is nil")
	}
	if !utf8.Valid(data) {
		return fmt.Errorf("provider JSON decode: input must be valid UTF-8")
	}
	if err := rejectDuplicateObjectProperties(data); err != nil {
		return fmt.Errorf("provider JSON decode: %w", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return fmt.Errorf("provider JSON decode: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return fmt.Errorf("provider JSON decode: trailing JSON value")
		}
		return fmt.Errorf("provider JSON decode: trailing data: %w", err)
	}
	if err := destination.Validate(); err != nil {
		return fmt.Errorf("provider JSON decode: %w", err)
	}
	return nil
}

func rejectDuplicateObjectProperties(data []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	var walk func() error
	walk = func() error {
		token, err := decoder.Token()
		if err != nil {
			return err
		}
		if token == nil {
			return fmt.Errorf("null values are not valid provider definitions")
		}
		delimiter, ok := token.(json.Delim)
		if !ok {
			return nil
		}
		switch delimiter {
		case '{':
			seen := make(map[string]struct{})
			for decoder.More() {
				nameToken, err := decoder.Token()
				if err != nil {
					return err
				}
				name, ok := nameToken.(string)
				if !ok {
					return fmt.Errorf("object property name is not a string")
				}
				if _, exists := seen[name]; exists {
					return fmt.Errorf("duplicate object property %q", name)
				}
				seen[name] = struct{}{}
				if err := walk(); err != nil {
					return err
				}
			}
			_, err = decoder.Token()
			return err
		case '[':
			for decoder.More() {
				if err := walk(); err != nil {
					return err
				}
			}
			_, err = decoder.Token()
			return err
		default:
			return fmt.Errorf("unexpected JSON delimiter %q", delimiter)
		}
	}
	return walk()
}
