package provider

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"unicode/utf8"
)

// EncodeNormalizerDescriptorJSON validates before emitting stable struct-ordered
// JSON. The descriptor contains identity and schema metadata, never payloads or
// credentials.
func EncodeNormalizerDescriptorJSON(descriptor NormalizerDescriptor) ([]byte, error) {
	if err := descriptor.Validate(); err != nil {
		return nil, fmt.Errorf("normalizer descriptor JSON encode: %w", err)
	}
	raw, err := json.Marshal(descriptor)
	if err != nil {
		return nil, fmt.Errorf("normalizer descriptor JSON encode: %w", err)
	}
	return raw, nil
}

// DecodeNormalizerDescriptorJSON rejects unknown/duplicate fields, nulls,
// invalid UTF-8, trailing values, and unsupported versions.
func DecodeNormalizerDescriptorJSON(data []byte, destination *NormalizerDescriptor) error {
	if destination == nil {
		return fmt.Errorf("normalizer descriptor JSON decode: destination is nil")
	}
	if !utf8.Valid(data) {
		return fmt.Errorf("normalizer descriptor JSON decode: input must be valid UTF-8")
	}
	if err := rejectDuplicateObjectProperties(data); err != nil {
		return fmt.Errorf("normalizer descriptor JSON decode: %w", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return fmt.Errorf("normalizer descriptor JSON decode: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return fmt.Errorf("normalizer descriptor JSON decode: trailing JSON value")
		}
		return fmt.Errorf("normalizer descriptor JSON decode: trailing data: %w", err)
	}
	if err := destination.Validate(); err != nil {
		return fmt.Errorf("normalizer descriptor JSON decode: %w", err)
	}
	return nil
}
