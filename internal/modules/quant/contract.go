package quant

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
)

const (
	ServiceContractV1      = "jax.quant_service/v1"
	DatasetContractV1      = "jax.quant_frozen_dataset/v1"
	ResultContractV1       = "jax.quant_result/v1"
	CoreAlgorithmNamespace = "jax.quant.algorithm"
)

type Bar struct {
	At     time.Time `json:"at"`
	Open   float64   `json:"open"`
	High   float64   `json:"high"`
	Low    float64   `json:"low"`
	Close  float64   `json:"close"`
	Volume float64   `json:"volume"`
}

type FrozenDataset struct {
	ID                    string    `json:"id"`
	ContractVersion       string    `json:"contract_version"`
	Instrument            string    `json:"instrument"`
	Frequency             string    `json:"frequency"`
	PriceAdjustmentPolicy string    `json:"price_adjustment_policy"`
	SourceReference       string    `json:"source_reference"`
	VolumeSource          string    `json:"volume_source"`
	FrozenAt              time.Time `json:"frozen_at"`
	Bars                  []Bar     `json:"bars"`
	ContentSHA256         string    `json:"content_sha256"`
}

type Parameter struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type InputReference struct {
	ID            string `json:"id"`
	ContentSHA256 string `json:"content_sha256"`
}

type Request struct {
	ContractVersion  string           `json:"contract_version"`
	RequestID        string           `json:"request_id"`
	Dataset          FrozenDataset    `json:"dataset"`
	InputReferences  []InputReference `json:"input_references,omitempty"`
	Algorithm        string           `json:"algorithm"`
	AlgorithmVersion string           `json:"algorithm_version"`
	Parameters       []Parameter      `json:"parameters,omitempty"`
}

type Value struct {
	Metric     string     `json:"metric"`
	Value      float64    `json:"value"`
	Unit       string     `json:"unit"`
	ObservedAt *time.Time `json:"observed_at,omitempty"`
}

type Response struct {
	ContractVersion  string           `json:"contract_version"`
	RequestID        string           `json:"request_id"`
	DatasetID        string           `json:"dataset_id"`
	InputSHA256      string           `json:"input_sha256"`
	InputReferences  []InputReference `json:"input_references,omitempty"`
	Algorithm        string           `json:"algorithm"`
	AlgorithmVersion string           `json:"algorithm_version"`
	CalculatedAt     time.Time        `json:"calculated_at"`
	Values           []Value          `json:"values"`
	Unknowns         []string         `json:"unknowns,omitempty"`
}

func NewFrozenDataset(id, instrument, frequency, adjustment, source string, frozenAt time.Time, bars []Bar) (FrozenDataset, error) {
	dataset := FrozenDataset{ID: id, ContractVersion: DatasetContractV1, Instrument: instrument, Frequency: frequency, PriceAdjustmentPolicy: adjustment, SourceReference: source, VolumeSource: "UNSPECIFIED", FrozenAt: frozenAt, Bars: cloneBars(bars)}
	if err := dataset.validateWithoutDigest(); err != nil {
		return FrozenDataset{}, err
	}
	dataset.ContentSHA256 = dataset.contentDigest()
	return dataset, nil
}

func NewFrozenDatasetWithVolumeSource(id, instrument, frequency, adjustment, source, volumeSource string, frozenAt time.Time, bars []Bar) (FrozenDataset, error) {
	dataset := FrozenDataset{ID: id, ContractVersion: DatasetContractV1, Instrument: instrument, Frequency: frequency, PriceAdjustmentPolicy: adjustment, SourceReference: source, VolumeSource: volumeSource, FrozenAt: frozenAt, Bars: cloneBars(bars)}
	if err := dataset.validateWithoutDigest(); err != nil {
		return FrozenDataset{}, err
	}
	dataset.ContentSHA256 = dataset.contentDigest()
	return dataset, nil
}

func (dataset FrozenDataset) Validate() error {
	if err := dataset.validateWithoutDigest(); err != nil {
		return err
	}
	if len(dataset.ContentSHA256) != sha256.Size*2 {
		return fmt.Errorf("dataset content SHA-256 is required")
	}
	if dataset.ContentSHA256 != dataset.contentDigest() {
		return fmt.Errorf("dataset content SHA-256 does not match frozen bars")
	}
	return nil
}

func (dataset FrozenDataset) validateWithoutDigest() error {
	if strings.TrimSpace(dataset.ID) == "" || strings.TrimSpace(dataset.Instrument) == "" || strings.TrimSpace(dataset.SourceReference) == "" {
		return fmt.Errorf("dataset ID, instrument, and source reference are required")
	}
	if dataset.VolumeSource != "UNSPECIFIED" && dataset.VolumeSource != "UNKNOWN" && dataset.VolumeSource != "SIP_CONSOLIDATED" && dataset.VolumeSource != "VENUE_SPECIFIC" && dataset.VolumeSource != "OTHER" {
		return fmt.Errorf("volume source must be explicit")
	}
	if dataset.ContractVersion != DatasetContractV1 || strings.TrimSpace(dataset.Frequency) == "" {
		return fmt.Errorf("dataset contract version and frequency are required")
	}
	if dataset.PriceAdjustmentPolicy != "AS_PROVIDED" && dataset.PriceAdjustmentPolicy != "SPLIT_ADJUSTED" {
		return fmt.Errorf("price adjustment policy must be explicit")
	}
	if dataset.FrozenAt.IsZero() || dataset.FrozenAt.Location() != time.UTC {
		return fmt.Errorf("dataset frozen_at must be UTC")
	}
	if len(dataset.Bars) == 0 {
		return fmt.Errorf("dataset requires at least one bar")
	}
	previous := time.Time{}
	for index, bar := range dataset.Bars {
		if bar.At.IsZero() || bar.At.Location() != time.UTC {
			return fmt.Errorf("bar %d timestamp must be UTC", index)
		}
		if !previous.IsZero() && !bar.At.After(previous) {
			return fmt.Errorf("bars must be strictly increasing without duplicate timestamps")
		}
		previous = bar.At
		if !finitePositive(bar.Open) || !finitePositive(bar.High) || !finitePositive(bar.Low) || !finitePositive(bar.Close) {
			return fmt.Errorf("bar %d prices must be finite and positive", index)
		}
		if bar.High < bar.Low || bar.High < bar.Open || bar.High < bar.Close || bar.Low > bar.Open || bar.Low > bar.Close {
			return fmt.Errorf("bar %d violates OHLC bounds", index)
		}
		if !finiteNonNegative(bar.Volume) {
			return fmt.Errorf("bar %d volume must be finite and non-negative", index)
		}
	}
	return nil
}

func (dataset FrozenDataset) contentDigest() string {
	canonical := struct {
		ContractVersion       string    `json:"contract_version"`
		Instrument            string    `json:"instrument"`
		Frequency             string    `json:"frequency"`
		PriceAdjustmentPolicy string    `json:"price_adjustment_policy"`
		SourceReference       string    `json:"source_reference"`
		VolumeSource          string    `json:"volume_source"`
		FrozenAt              time.Time `json:"frozen_at"`
		Bars                  []Bar     `json:"bars"`
	}{dataset.ContractVersion, dataset.Instrument, dataset.Frequency, dataset.PriceAdjustmentPolicy, dataset.SourceReference, dataset.VolumeSource, dataset.FrozenAt.UTC(), dataset.Bars}
	bytes, _ := json.Marshal(canonical)
	digest := sha256.Sum256(bytes)
	return hex.EncodeToString(digest[:])
}

func NewRequest(requestID string, dataset FrozenDataset, algorithm, algorithmVersion string, parameters []Parameter) (Request, error) {
	return NewRequestWithInputs(requestID, dataset, nil, algorithm, algorithmVersion, parameters)
}

func NewRequestWithInputs(requestID string, dataset FrozenDataset, inputs []InputReference, algorithm, algorithmVersion string, parameters []Parameter) (Request, error) {
	request := Request{ContractVersion: ServiceContractV1, RequestID: requestID, Dataset: dataset, Algorithm: algorithm, AlgorithmVersion: algorithmVersion, Parameters: append([]Parameter(nil), parameters...)}
	request.InputReferences = append([]InputReference(nil), inputs...)
	sort.Slice(request.Parameters, func(left, right int) bool { return request.Parameters[left].Name < request.Parameters[right].Name })
	sort.Slice(request.InputReferences, func(left, right int) bool {
		return request.InputReferences[left].ID < request.InputReferences[right].ID
	})
	if err := request.Validate(); err != nil {
		return Request{}, err
	}
	return request, nil
}

func (request Request) Validate() error {
	if request.ContractVersion != ServiceContractV1 || strings.TrimSpace(request.RequestID) == "" || strings.TrimSpace(request.Algorithm) == "" || strings.TrimSpace(request.AlgorithmVersion) == "" {
		return fmt.Errorf("request contract, ID, algorithm, and algorithm version are required")
	}
	if err := request.Dataset.Validate(); err != nil {
		return fmt.Errorf("request dataset: %w", err)
	}
	seenInputs := map[string]struct{}{request.Dataset.ID: {}}
	for _, input := range request.InputReferences {
		if input.ID == "" || len(input.ContentSHA256) != sha256.Size*2 {
			return fmt.Errorf("input references require an ID and SHA-256")
		}
		if _, exists := seenInputs[input.ID]; exists {
			return fmt.Errorf("input reference %q is duplicated", input.ID)
		}
		seenInputs[input.ID] = struct{}{}
	}
	seen := map[string]struct{}{}
	for _, parameter := range request.Parameters {
		if strings.TrimSpace(parameter.Name) == "" || strings.TrimSpace(parameter.Value) == "" {
			return fmt.Errorf("request parameters require non-empty names and values")
		}
		if _, exists := seen[parameter.Name]; exists {
			return fmt.Errorf("request parameter %q is duplicated", parameter.Name)
		}
		seen[parameter.Name] = struct{}{}
	}
	return nil
}

func NewResponse(request Request, calculatedAt time.Time, values []Value, unknowns []string) (Response, error) {
	response := Response{ContractVersion: ResultContractV1, RequestID: request.RequestID, DatasetID: request.Dataset.ID, InputSHA256: request.Dataset.ContentSHA256, InputReferences: append([]InputReference(nil), request.InputReferences...), Algorithm: request.Algorithm, AlgorithmVersion: request.AlgorithmVersion, CalculatedAt: calculatedAt, Values: append([]Value(nil), values...), Unknowns: sortedUnique(unknowns)}
	if err := response.Validate(request); err != nil {
		return Response{}, err
	}
	return response, nil
}

func (response Response) Validate(request Request) error {
	if response.ContractVersion != ResultContractV1 || response.RequestID != request.RequestID || response.DatasetID != request.Dataset.ID || response.InputSHA256 != request.Dataset.ContentSHA256 || response.Algorithm != request.Algorithm || response.AlgorithmVersion != request.AlgorithmVersion || !sameInputReferences(response.InputReferences, request.InputReferences) {
		return fmt.Errorf("response identity does not match request")
	}
	if response.CalculatedAt.IsZero() || response.CalculatedAt.Location() != time.UTC {
		return fmt.Errorf("response calculated_at must be UTC")
	}
	if len(response.Values) == 0 {
		return fmt.Errorf("response requires at least one value")
	}
	seen := map[string]struct{}{}
	for _, value := range response.Values {
		if strings.TrimSpace(value.Metric) == "" || strings.TrimSpace(value.Unit) == "" || !finite(value.Value) {
			return fmt.Errorf("response values require finite values, metric names, and units")
		}
		if value.ObservedAt != nil && value.ObservedAt.Location() != time.UTC {
			return fmt.Errorf("response observed_at must be UTC")
		}
		identity := value.Metric
		if value.ObservedAt != nil {
			identity += "\x00" + value.ObservedAt.Format(time.RFC3339Nano)
		}
		if _, exists := seen[identity]; exists {
			return fmt.Errorf("response metric identity %q is duplicated", identity)
		}
		seen[value.Metric] = struct{}{}
	}
	return nil
}

func finite(value float64) bool            { return !math.IsNaN(value) && !math.IsInf(value, 0) }
func finitePositive(value float64) bool    { return finite(value) && value > 0 }
func finiteNonNegative(value float64) bool { return finite(value) && value >= 0 }

func cloneBars(bars []Bar) []Bar { return append([]Bar(nil), bars...) }

func sortedUnique(values []string) []string {
	copyValues := append([]string(nil), values...)
	sort.Strings(copyValues)
	result := make([]string, 0, len(copyValues))
	for _, value := range copyValues {
		if value != "" && (len(result) == 0 || result[len(result)-1] != value) {
			result = append(result, value)
		}
	}
	return result
}

func sameInputReferences(left, right []InputReference) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
