package quant

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"time"
)

func newCalculationRequest(dataset FrozenDataset, inputReferences []InputReference, algorithm, algorithmVersion string, parameters []Parameter) (Request, error) {
	if err := dataset.Validate(); err != nil {
		return Request{}, err
	}
	requestBytes, _ := json.Marshal(struct {
		DatasetID  string           `json:"dataset_id"`
		DatasetSHA string           `json:"dataset_sha"`
		Inputs     []InputReference `json:"inputs"`
		Algorithm  string           `json:"algorithm"`
		Version    string           `json:"version"`
		Parameters []Parameter      `json:"parameters"`
	}{dataset.ID, dataset.ContentSHA256, inputReferences, algorithm, algorithmVersion, parameters})
	digest := sha256.Sum256(requestBytes)
	requestID := "qreq_" + hex.EncodeToString(digest[:])
	return NewRequestWithInputs(requestID, dataset, inputReferences, algorithm, algorithmVersion, parameters)
}

func newCalculationResponse(request Request, values []Value, unknowns []string) (Response, error) {
	return NewResponse(request, request.Dataset.FrozenAt.UTC(), values, unknowns)
}

func valuesForReturns(dataset FrozenDataset, metric string, logarithmic bool) ([]Value, error) {
	if len(dataset.Bars) < 2 {
		return nil, fmt.Errorf("%s requires at least two bars", metric)
	}
	values := make([]Value, 0, len(dataset.Bars)-1)
	for index := 1; index < len(dataset.Bars); index++ {
		previous, current := dataset.Bars[index-1].Close, dataset.Bars[index].Close
		ratio := current / previous
		value := ratio - 1
		if logarithmic {
			value = mathLog(ratio)
		}
		at := dataset.Bars[index].At
		values = append(values, Value{Metric: metric, Value: value, Unit: "decimal_return", ObservedAt: &at})
	}
	return values, nil
}

func mathLog(value float64) float64 {
	return math.Log(value)
}

func ensureFiniteValues(values []Value) error {
	for _, value := range values {
		if !finite(value.Value) || value.ObservedAt == nil || value.ObservedAt.Location() != time.UTC {
			return fmt.Errorf("calculation produced a non-finite or non-UTC result")
		}
	}
	return nil
}

func formatFloat(value float64) string {
	return strconv.FormatFloat(value, 'g', -1, 64)
}
