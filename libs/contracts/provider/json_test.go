package provider

import (
	"bytes"
	"reflect"
	"testing"
)

func TestProviderDefinitionJSONRoundTripIsStable(t *testing.T) {
	definition := validDefinition(CapabilityEventFeed)
	first, err := EncodeDefinitionJSON(definition)
	if err != nil {
		t.Fatalf("EncodeDefinitionJSON() error = %v", err)
	}
	second, err := EncodeDefinitionJSON(definition)
	if err != nil {
		t.Fatalf("second EncodeDefinitionJSON() error = %v", err)
	}
	if !bytes.Equal(first, second) {
		t.Fatalf("encoding is not stable:\n%s\n%s", first, second)
	}
	var decoded ProviderDefinition
	if err := DecodeDefinitionJSON(first, &decoded); err != nil {
		t.Fatalf("DecodeDefinitionJSON() error = %v", err)
	}
	if !reflect.DeepEqual(decoded, definition) {
		t.Fatalf("round trip mismatch:\nwant %#v\n got %#v", definition, decoded)
	}
}

func TestProviderDefinitionJSONRejectsSecretMaterialAndInvalidInput(t *testing.T) {
	raw, err := EncodeDefinitionJSON(validDefinition(CapabilityMarketQuote))
	if err != nil {
		t.Fatalf("EncodeDefinitionJSON() error = %v", err)
	}
	withSecret := bytes.Replace(raw, []byte(`"display_name":`), []byte(`"credential_value":"do-not-store","display_name":`), 1)
	duplicateName := bytes.Replace(raw, []byte(`"display_name":`), []byte(`"display_name":"Duplicate","display_name":`), 1)
	withNull := bytes.Replace(raw, []byte(`"display_name":"Market Data Provider"`), []byte(`"display_name":null`), 1)

	tests := []struct {
		name string
		raw  []byte
	}{
		{"secret_unknown_field", withSecret},
		{"duplicate_field", duplicateName},
		{"null_field", withNull},
		{"trailing_value", append(append([]byte(nil), raw...), []byte(` {}`)...)},
		{"unsupported_version", bytes.Replace(raw, []byte(string(ProviderDefinitionV1)), []byte("jax.provider_definition/v99"), 1)},
		{"invalid_utf8", append(append([]byte(nil), raw...), 0xff)},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var decoded ProviderDefinition
			if err := DecodeDefinitionJSON(test.raw, &decoded); err == nil {
				t.Fatal("DecodeDefinitionJSON() accepted invalid input")
			}
		})
	}
	if err := DecodeDefinitionJSON(raw, nil); err == nil {
		t.Fatal("DecodeDefinitionJSON() accepted nil destination")
	}
}
