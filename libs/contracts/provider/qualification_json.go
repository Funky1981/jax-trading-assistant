package provider

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"unicode/utf8"
)

func EncodeQualificationDecisionJSON(decision QualificationDecision) ([]byte, error) {
	if err := decision.Validate(); err != nil {
		return nil, fmt.Errorf("qualification JSON encode: %w", err)
	}
	raw, err := json.Marshal(decision)
	if err != nil {
		return nil, fmt.Errorf("qualification JSON encode: %w", err)
	}
	return raw, nil
}

// DecodeQualificationDecisionJSON is strict so unknown extension fields,
// duplicate properties, nulls, and future versions cannot silently acquire
// qualification meaning.
func DecodeQualificationDecisionJSON(data []byte, destination *QualificationDecision) error {
	if destination == nil {
		return fmt.Errorf("qualification JSON decode: destination is nil")
	}
	if !utf8.Valid(data) {
		return fmt.Errorf("qualification JSON decode: input must be valid UTF-8")
	}
	if err := rejectDuplicateObjectProperties(data); err != nil {
		return fmt.Errorf("qualification JSON decode: %w", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return fmt.Errorf("qualification JSON decode: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return fmt.Errorf("qualification JSON decode: trailing JSON value")
		}
		return fmt.Errorf("qualification JSON decode: trailing data: %w", err)
	}
	if err := destination.Validate(); err != nil {
		return fmt.Errorf("qualification JSON decode: %w", err)
	}
	return nil
}
