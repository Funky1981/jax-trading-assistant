package canonical

import (
	"bytes"
	"math"
	"testing"
	"time"
)

func TestCanonicalByteSpecificationExample(t *testing.T) {
	occurred := time.Date(2026, 8, 21, 10, 0, 0, 123400000, time.UTC)
	event := Event{
		ContractVersion: EventContractV1,
		ID:              "evt_byte_spec",
		Type:            EventTypeNews,
		Assertion:       EventAssertionAsserted,
		Title:           "A <tag> & line\nnext",
		Summary:         "café",
		OccurredAt:      &occurred,
		CreatedAt:       occurred,
	}
	got, err := CanonicalContractBytes(event)
	if err != nil {
		t.Fatalf("CanonicalContractBytes() error = %v", err)
	}
	want := []byte(`{"canonicalization":"jax.canonical-contract-json/v1","contract_kind":"event","contract_version":"jax.event/v1","content":{"contract_version":"jax.event/v1","id":"evt_byte_spec","type":"news","assertion":"asserted","title":"A \u003ctag\u003e \u0026 line\nnext","summary":"café","occurred_at":"2026-08-21T10:00:00.1234Z","created_at":"2026-08-21T10:00:00.1234Z"}}`)
	if !bytes.Equal(got, want) {
		t.Fatalf("canonical bytes mismatch:\nwant: %s\n got: %s", want, got)
	}
}

func TestCanonicalJSONRejectsAmbiguousInput(t *testing.T) {
	duplicate := []byte(`{"contract_version":"jax.event/v1","id":"evt_duplicate","type":"news","assertion":"asserted","title":"first","title":"second","occurred_at":"2026-08-21T10:00:00Z","created_at":"2026-08-21T10:00:00Z"}`)
	if err := DecodeJSON(duplicate, &Event{}); err == nil {
		t.Fatal("DecodeJSON() accepted a duplicate property")
	}
	nullOptional := []byte(`{"contract_version":"jax.event/v1","id":"evt_null","type":"news","assertion":"asserted","title":"null is ambiguous","summary":null,"occurred_at":"2026-08-21T10:00:00Z","created_at":"2026-08-21T10:00:00Z"}`)
	if err := DecodeJSON(nullOptional, &Event{}); err == nil {
		t.Fatal("DecodeJSON() accepted an explicit null")
	}
	invalidUTF8 := append([]byte(`{"contract_version":"jax.event/v1","id":"evt_utf8","type":"news","assertion":"asserted","title":"`), 0xff)
	invalidUTF8 = append(invalidUTF8, []byte(`","occurred_at":"2026-08-21T10:00:00Z","created_at":"2026-08-21T10:00:00Z"}`)...)
	if err := DecodeJSON(invalidUTF8, &Event{}); err == nil {
		t.Fatal("DecodeJSON() accepted invalid UTF-8")
	}

	result := validContracts().quant
	result.Values[0].Value = math.Copysign(0, -1)
	assertValidationField(t, result.Validate(), "values[0].value")
}
