package cboe

import (
	"encoding/csv"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"
	"time"

	"jax-trading-assistant/libs/contracts/canonical"
	providercontract "jax-trading-assistant/libs/contracts/provider"
	"jax-trading-assistant/libs/macroevidence"
)

func parseHistory(raw []byte, ref providercontract.RawPayloadRef) (macroevidence.MacroSeries, []macroevidence.MacroObservation, error) {
	reader := csv.NewReader(strings.NewReader(string(raw)))
	reader.FieldsPerRecord = 5
	header, err := reader.Read()
	if err != nil {
		return macroevidence.MacroSeries{}, nil, fmt.Errorf("Cboe VIX CSV header is missing: %w", err)
	}
	expected := []string{"DATE", "OPEN", "HIGH", "LOW", "CLOSE"}
	for index := range expected {
		if strings.TrimSpace(header[index]) != expected[index] {
			return macroevidence.MacroSeries{}, nil, fmt.Errorf("Cboe VIX CSV header is not the documented schema")
		}
	}
	type row struct {
		date        macroevidence.Date
		sourceValue string
		value       macroevidence.MacroValue
	}
	rows := make([]row, 0, 10000)
	var previous macroevidence.Date
	for {
		fields, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return macroevidence.MacroSeries{}, nil, fmt.Errorf("Cboe VIX CSV row is malformed: %w", err)
		}
		date, err := parseDate(fields[0])
		if err != nil {
			return macroevidence.MacroSeries{}, nil, err
		}
		if previous != "" && date <= previous {
			return macroevidence.MacroSeries{}, nil, fmt.Errorf("Cboe VIX dates are not strictly ascending or contain duplicates")
		}
		previous = date
		sourceValue := strings.TrimSpace(fields[4])
		value, err := parseValue(sourceValue)
		if err != nil {
			return macroevidence.MacroSeries{}, nil, fmt.Errorf("Cboe VIX %s close: %w", date, err)
		}
		rows = append(rows, row{date: date, sourceValue: sourceValue, value: value})
	}
	if len(rows) == 0 {
		return macroevidence.MacroSeries{}, nil, fmt.Errorf("Cboe VIX CSV contains no data rows")
	}
	seriesProvenance, err := makeProvenance(ref, "series")
	if err != nil {
		return macroevidence.MacroSeries{}, nil, err
	}
	series := macroevidence.MacroSeries{ID: macroevidence.MacroSeriesID("mser_" + canonical.DigestBytes([]byte("cboe-series\x00VIX")).Value[:24]), ProviderSeriesID: "VIX", Title: "Cboe Volatility Index (VIX) daily closing values", ObservationStart: rows[0].date, ObservationEnd: rows[len(rows)-1].date, Frequency: "Daily", FrequencyCode: "D", Units: "index points", UnitsShort: "index points", RequestedInformation: macroevidence.InformationState{Mode: macroevidence.InformationStateCurrent}, SourcePayload: ref, Provenance: seriesProvenance}
	if err := series.Validate(); err != nil {
		return macroevidence.MacroSeries{}, nil, err
	}
	observations := make([]macroevidence.MacroObservation, 0, len(rows))
	for _, item := range rows {
		seed := strings.Join([]string{"cboe-vix-observation", string(item.date), item.sourceValue}, "\x00")
		provenance, err := makeProvenance(ref, seed)
		if err != nil {
			return macroevidence.MacroSeries{}, nil, err
		}
		observation := macroevidence.MacroObservation{ID: "mobs_" + canonical.DigestBytes([]byte(seed)).Value[:24], Series: series.ID, ProviderSeriesID: "VIX", ObservationDate: item.date, Value: item.value, RequestedInformation: series.RequestedInformation, AcquiredAt: ref.ReceivedAt, SourcePayload: ref, Provenance: provenance}
		if err := observation.Validate(); err != nil {
			return macroevidence.MacroSeries{}, nil, err
		}
		observations = append(observations, observation)
	}
	return series, observations, nil
}

func parseDate(raw string) (macroevidence.Date, error) {
	parsed, err := time.Parse("01/02/2006", strings.TrimSpace(raw))
	if err != nil {
		return "", fmt.Errorf("Cboe VIX date is malformed")
	}
	return macroevidence.Date(parsed.Format("2006-01-02")), nil
}

func parseValue(raw string) (macroevidence.MacroValue, error) {
	if raw == "" {
		return macroevidence.MacroValue{}, fmt.Errorf("value is empty")
	}
	upper := strings.ToUpper(raw)
	if upper == "N/A" || upper == "NA" || upper == "NULL" {
		return macroevidence.MacroValue{Present: false, SourceValue: raw}, nil
	}
	number, err := strconv.ParseFloat(raw, 64)
	if err != nil || math.IsNaN(number) || math.IsInf(number, 0) {
		return macroevidence.MacroValue{}, fmt.Errorf("value is not a finite numeric value")
	}
	return macroevidence.MacroValue{Present: true, SourceValue: raw, Number: &number}, nil
}

func makeProvenance(ref providercontract.RawPayloadRef, salt string) (canonical.Provenance, error) {
	evidenceID := canonical.EvidenceID("evd_" + canonical.DigestBytes([]byte("cboe-evidence\x00" + string(ref.ID))).Value[:24])
	evidenceRef, err := ref.AsEvidenceRef(canonical.ContractRef{Kind: canonical.ContractKindEvidence, ID: string(evidenceID), ContractVersion: canonical.EvidenceContractV2})
	if err != nil {
		return canonical.Provenance{}, err
	}
	inputs := []canonical.LineageInput{{Kind: canonical.LineageInputKindEvidence, Evidence: &evidenceRef}}
	fingerprint, err := canonical.ComputeInputFingerprint(inputs)
	if err != nil {
		return canonical.Provenance{}, err
	}
	provider := ProviderIdentity
	return canonical.Provenance{ContractVersion: canonical.ProvenanceContractV1, ID: "pvn_" + canonical.DigestBytes([]byte("cboe-provenance\x00" + salt)).Value[:24], Inputs: inputs, InputFingerprint: fingerprint, Producer: canonical.ComponentIdentity{ID: NormalizerID, Kind: canonical.ComponentKindNormalizer, Name: "Cboe VIX deterministic normalizer", Version: canonical.VersionIdentity{Namespace: "jax.cboe.normalizer", Value: NormalizerVersion}, Provider: &provider}}, nil
}
