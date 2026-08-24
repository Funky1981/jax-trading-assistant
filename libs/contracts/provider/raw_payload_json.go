package provider

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"unicode/utf8"
)

// EncodeRawPayloadRefJSON validates before serializing a reference. Raw bytes,
// request headers, and credentials are not fields of this contract.
func EncodeRawPayloadRefJSON(ref RawPayloadRef) ([]byte, error) {
	if err := ref.Validate(); err != nil {
		return nil, fmt.Errorf("raw payload reference JSON encode: %w", err)
	}
	raw, err := json.Marshal(ref)
	if err != nil {
		return nil, fmt.Errorf("raw payload reference JSON encode: %w", err)
	}
	return raw, nil
}

// DecodeRawPayloadRefJSON fails closed for unknown/duplicate fields, nulls,
// invalid UTF-8, trailing values, unsupported versions, and invalid metadata.
func DecodeRawPayloadRefJSON(data []byte, destination *RawPayloadRef) error {
	if destination == nil {
		return fmt.Errorf("raw payload reference JSON decode: destination is nil")
	}
	if !utf8.Valid(data) {
		return fmt.Errorf("raw payload reference JSON decode: input must be valid UTF-8")
	}
	if err := rejectDuplicateObjectProperties(data); err != nil {
		return fmt.Errorf("raw payload reference JSON decode: %w", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return fmt.Errorf("raw payload reference JSON decode: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return fmt.Errorf("raw payload reference JSON decode: trailing JSON value")
		}
		return fmt.Errorf("raw payload reference JSON decode: trailing data: %w", err)
	}
	if err := destination.Validate(); err != nil {
		return fmt.Errorf("raw payload reference JSON decode: %w", err)
	}
	return nil
}
