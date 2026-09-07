package quant

import (
	"math"
	"testing"
	"time"
)

func testDataset(t *testing.T) FrozenDataset {
	t.Helper()
	now := time.Date(2026, 9, 7, 12, 0, 0, 0, time.UTC)
	dataset, err := NewFrozenDataset("ds_known", "TEST", "1D", "AS_PROVIDED", "frozen-fixture-v1", now, []Bar{{At: now.Add(-24 * time.Hour), Open: 99, High: 101, Low: 98, Close: 100, Volume: 10}, {At: now, Open: 100, High: 103, Low: 99, Close: 102, Volume: 12}})
	if err != nil {
		t.Fatal(err)
	}
	return dataset
}

func TestFrozenDatasetAndResponseAreVersionedAndBoundToInput(t *testing.T) {
	dataset := testDataset(t)
	if err := dataset.Validate(); err != nil {
		t.Fatal(err)
	}
	request, err := NewRequest("req_1", dataset, "returns", "returns-v1", []Parameter{{Name: "price_field", Value: "close"}})
	if err != nil {
		t.Fatal(err)
	}
	response, err := NewResponse(request, time.Date(2026, 9, 7, 12, 1, 0, 0, time.UTC), []Value{{Metric: "known", Value: 2, Unit: "count"}}, []string{"b", "a", "a"})
	if err != nil || len(response.Unknowns) != 2 || response.InputSHA256 != dataset.ContentSHA256 {
		t.Fatalf("response=%+v err=%v", response, err)
	}
	if math.IsNaN(response.Values[0].Value) {
		t.Fatal("NaN accepted")
	}
}

func TestFrozenDatasetRejectsLookalikeInvalidInputs(t *testing.T) {
	dataset := testDataset(t)
	dataset.Bars[1].At = dataset.Bars[0].At
	if err := dataset.Validate(); err == nil {
		t.Fatal("duplicate timestamps accepted")
	}
	dataset = testDataset(t)
	dataset.Bars[0].Close = math.NaN()
	dataset.ContentSHA256 = dataset.contentDigest()
	if err := dataset.Validate(); err == nil {
		t.Fatal("NaN close accepted")
	}
	dataset = testDataset(t)
	dataset.Bars[0].At = dataset.Bars[0].At.In(time.FixedZone("offset", 3600))
	dataset.ContentSHA256 = dataset.contentDigest()
	if err := dataset.Validate(); err == nil {
		t.Fatal("non-UTC timestamp accepted")
	}
}

func TestRequestRejectsMalformedSecondaryInputDigest(t *testing.T) {
	dataset := testDataset(t)
	for _, digest := range []string{"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA", "gggggggggggggggggggggggggggggggggggggggggggggggggggggggggggggggg"} {
		if _, err := NewRequestWithInputs("request", dataset, []InputReference{{ID: "benchmark", ContentSHA256: digest}}, "algorithm", "v1", nil); err == nil {
			t.Fatalf("malformed input digest accepted: %q", digest)
		}
	}
}
