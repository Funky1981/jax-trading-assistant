package canonical

import (
	"fmt"
	"time"
)

type QuantResultType string

const (
	QuantResultTypeScalar       QuantResultType = "scalar"
	QuantResultTypeSeries       QuantResultType = "series"
	QuantResultTypeDistribution QuantResultType = "distribution"
	QuantResultTypeRanking      QuantResultType = "ranking"
	QuantResultTypeStatistics   QuantResultType = "statistics"
)

// QuantDimension is a typed label on a quantitative value. A slice is used
// instead of a map so serialization order is explicit and stable.
type QuantDimension struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// QuantValue is a named numeric value with explicit unit and optional series
// time/rank semantics. New algorithms extend metric names and dimensions
// without introducing arbitrary JSON.
type QuantValue struct {
	Metric     string           `json:"metric"`
	Value      float64          `json:"value"`
	Unit       string           `json:"unit"`
	ObservedAt *time.Time       `json:"observed_at,omitempty"`
	Rank       *int             `json:"rank,omitempty"`
	Dimensions []QuantDimension `json:"dimensions,omitempty"`
}

// QuantResult is a deterministic/statistical result produced by a bounded
// ResearchRun from identified canonical inputs.
type QuantResult struct {
	ContractVersion ContractVersion `json:"contract_version"`
	ID              QuantResultID   `json:"id"`
	Type            QuantResultType `json:"type"`
	ResearchRunID   ResearchRunID   `json:"research_run_id"`
	Method          ComponentRef    `json:"method"`
	InputRefs       []ContractRef   `json:"input_refs"`
	Values          []QuantValue    `json:"values"`
	CalculatedAt    time.Time       `json:"calculated_at"`
}

func (result QuantResult) Validate() error {
	const contract = "quant_result"
	if err := validateVersion(contract, result.ContractVersion, QuantResultContractV1); err != nil {
		return err
	}
	if err := validateCanonicalID(contract, "id", string(result.ID), "qnt_"); err != nil {
		return err
	}
	switch result.Type {
	case QuantResultTypeScalar, QuantResultTypeSeries, QuantResultTypeDistribution,
		QuantResultTypeRanking, QuantResultTypeStatistics:
	default:
		return invalid(contract, "type", "is not supported")
	}
	if err := validateCanonicalID(contract, "research_run_id", string(result.ResearchRunID), "run_"); err != nil {
		return err
	}
	if err := validateComponent(contract, "method", result.Method); err != nil {
		return err
	}
	if len(result.InputRefs) == 0 {
		return invalid(contract, "input_refs", "requires at least one identified input")
	}
	if err := validateContractRefs(contract, "input_refs", result.InputRefs, nil); err != nil {
		return err
	}
	for _, ref := range result.InputRefs {
		if ref.Kind == ContractKindQuantResult && ref.ID == string(result.ID) {
			return invalid(contract, "input_refs", "must not contain the result itself")
		}
	}
	if len(result.Values) == 0 {
		return invalid(contract, "values", "requires at least one typed quantitative value")
	}
	seenValues := make(map[string]struct{}, len(result.Values))
	for i, value := range result.Values {
		field := fmt.Sprintf("values[%d]", i)
		if err := validateCode(contract, field+".metric", value.Metric); err != nil {
			return err
		}
		if err := validateFinite(contract, field+".value", value.Value); err != nil {
			return err
		}
		if err := validateRequiredText(contract, field+".unit", value.Unit, maxShortText); err != nil {
			return err
		}
		if err := validateOptionalUTC(contract, field+".observed_at", value.ObservedAt); err != nil {
			return err
		}
		if result.Type == QuantResultTypeSeries && value.ObservedAt == nil {
			return invalid(contract, field+".observed_at", "is required for series results")
		}
		if result.Type == QuantResultTypeRanking {
			if value.Rank == nil || *value.Rank <= 0 {
				return invalid(contract, field+".rank", "must be positive for ranking results")
			}
		} else if value.Rank != nil {
			return invalid(contract, field+".rank", "is only valid for ranking results")
		}
		dimensions := make(map[string]struct{}, len(value.Dimensions))
		identity := value.Metric
		if value.ObservedAt != nil {
			identity += "\x00" + value.ObservedAt.Format(time.RFC3339Nano)
		}
		for j, dimension := range value.Dimensions {
			dimensionField := fmt.Sprintf("%s.dimensions[%d]", field, j)
			if err := validateCode(contract, dimensionField+".name", dimension.Name); err != nil {
				return err
			}
			if err := validateRequiredText(contract, dimensionField+".value", dimension.Value, maxShortText); err != nil {
				return err
			}
			if _, ok := dimensions[dimension.Name]; ok {
				return invalid(contract, dimensionField+".name", "duplicates a dimension name in this value")
			}
			dimensions[dimension.Name] = struct{}{}
			identity += "\x00" + dimension.Name + "=" + dimension.Value
		}
		if _, ok := seenValues[identity]; ok {
			return invalid(contract, field, "duplicates an earlier quantitative value identity")
		}
		seenValues[identity] = struct{}{}
	}
	return validateRequiredUTC(contract, "calculated_at", result.CalculatedAt)
}
