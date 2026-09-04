package treasury

import (
	"encoding/xml"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"jax-trading-assistant/libs/contracts/canonical"
	providercontract "jax-trading-assistant/libs/contracts/provider"
	"jax-trading-assistant/libs/macroevidence"
)

type feed struct {
	Updated string  `xml:"updated"`
	Entries []entry `xml:"entry"`
}

type entry struct {
	Properties properties `xml:"content>properties"`
}

type properties struct {
	Date       string `xml:"NEW_DATE"`
	OneMonth   string `xml:"BC_1MONTH"`
	One15Month string `xml:"BC_1_5MONTH"`
	TwoMonth   string `xml:"BC_2MONTH"`
	ThreeMonth string `xml:"BC_3MONTH"`
	FourMonth  string `xml:"BC_4MONTH"`
	SixMonth   string `xml:"BC_6MONTH"`
	OneYear    string `xml:"BC_1YEAR"`
	TwoYear    string `xml:"BC_2YEAR"`
	ThreeYear  string `xml:"BC_3YEAR"`
	FiveYear   string `xml:"BC_5YEAR"`
	SevenYear  string `xml:"BC_7YEAR"`
	TenYear    string `xml:"BC_10YEAR"`
	TwentyYear string `xml:"BC_20YEAR"`
	ThirtyYear string `xml:"BC_30YEAR"`
}

func parseYieldCurve(raw []byte, ref providercontract.RawPayloadRef, requestedYear int) ([]YieldObservation, error) {
	var document feed
	if err := xml.Unmarshal(raw, &document); err != nil {
		return nil, fmt.Errorf("Treasury XML parse failed: %w", err)
	}
	if len(document.Entries) == 0 {
		return nil, errorsf("Treasury yield curve contains no entries")
	}
	updated, err := parseUpdated(document.Updated)
	if err != nil {
		return nil, err
	}
	values := map[string]func(properties) string{
		"1 Mo": func(p properties) string { return p.OneMonth }, "1.5 Mo": func(p properties) string { return p.One15Month },
		"2 Mo": func(p properties) string { return p.TwoMonth }, "3 Mo": func(p properties) string { return p.ThreeMonth },
		"4 Mo": func(p properties) string { return p.FourMonth }, "6 Mo": func(p properties) string { return p.SixMonth },
		"1 Yr": func(p properties) string { return p.OneYear }, "2 Yr": func(p properties) string { return p.TwoYear },
		"3 Yr": func(p properties) string { return p.ThreeYear }, "5 Yr": func(p properties) string { return p.FiveYear },
		"7 Yr": func(p properties) string { return p.SevenYear }, "10 Yr": func(p properties) string { return p.TenYear },
		"20 Yr": func(p properties) string { return p.TwentyYear }, "30 Yr": func(p properties) string { return p.ThirtyYear },
	}
	seriesByTenor := make(map[string]macroevidence.MacroSeries, len(tenors))
	rows := make([]properties, 0, len(document.Entries))
	var previous macroevidence.Date
	for _, item := range document.Entries {
		date, err := parseTreasuryDate(item.Properties.Date)
		if err != nil {
			return nil, err
		}
		if string(date[:4]) != strconv.Itoa(requestedYear) {
			return nil, fmt.Errorf("Treasury response contains a row outside requested year %d", requestedYear)
		}
		if previous != "" && date <= previous {
			return nil, fmt.Errorf("Treasury yield curve dates are not strictly ascending or contain duplicates")
		}
		previous = date
		rows = append(rows, item.Properties)
	}
	for _, item := range tenors {
		seriesID := macroevidence.MacroSeriesID("mser_" + canonical.DigestBytes([]byte("treasury-series\x00" + item.Name)).Value[:24])
		provenance, err := makeProvenance(ref, "series\x00"+item.Name)
		if err != nil {
			return nil, err
		}
		series := macroevidence.MacroSeries{ID: seriesID, ProviderSeriesID: "daily_treasury_par_yield_curve:" + item.XML, Title: "Daily Treasury Par Yield Curve Rates - " + item.Name, ObservationStart: mustDate(rows[0].Date), ObservationEnd: mustDate(rows[len(rows)-1].Date), Frequency: "Daily", FrequencyCode: "D", Units: "percent", UnitsShort: "%", LastUpdated: updated, RequestedInformation: macroevidence.InformationState{Mode: macroevidence.InformationStateCurrent}, SourcePayload: ref, Provenance: provenance}
		if err := series.Validate(); err != nil {
			return nil, err
		}
		seriesByTenor[item.Name] = series
	}
	out := make([]YieldObservation, 0, len(rows)*len(tenors))
	for _, row := range rows {
		date, _ := parseTreasuryDate(row.Date)
		for _, item := range tenors {
			series := seriesByTenor[item.Name]
			sourceValue := strings.TrimSpace(values[item.Name](row))
			value, err := parseTreasuryValue(sourceValue)
			if err != nil {
				return nil, fmt.Errorf("Treasury %s %s value: %w", date, item.Name, err)
			}
			idSeed := strings.Join([]string{"treasury-observation", item.Name, string(date), sourceValue}, "\x00")
			observationID := "mobs_" + canonical.DigestBytes([]byte(idSeed)).Value[:24]
			provenance, err := makeProvenance(ref, "observation\x00"+idSeed)
			if err != nil {
				return nil, err
			}
			observation := macroevidence.MacroObservation{ID: observationID, Series: series.ID, ProviderSeriesID: series.ProviderSeriesID, ObservationDate: date, Value: value, RequestedInformation: series.RequestedInformation, AcquiredAt: ref.ReceivedAt, SourcePayload: ref, Provenance: provenance}
			if err := observation.Validate(); err != nil {
				return nil, err
			}
			out = append(out, YieldObservation{Series: series, Observation: observation, Tenor: item.Name})
		}
	}
	return out, nil
}

func parseTreasuryDate(raw string) (macroevidence.Date, error) {
	raw = strings.TrimSpace(raw)
	if len(raw) < 10 {
		return "", fmt.Errorf("Treasury date is malformed")
	}
	date := macroevidence.Date(raw[:10])
	if err := date.Validate(); err != nil {
		return "", fmt.Errorf("Treasury date: %w", err)
	}
	return date, nil
}

func mustDate(raw string) macroevidence.Date {
	date, _ := parseTreasuryDate(raw)
	return date
}

func parseTreasuryValue(raw string) (macroevidence.MacroValue, error) {
	if raw == "" {
		return macroevidence.MacroValue{Present: false, SourceValue: ""}, nil
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

func parseUpdated(raw string) (*time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return nil, fmt.Errorf("Treasury feed updated timestamp is malformed")
	}
	value := parsed.UTC()
	return &value, nil
}

func makeProvenance(ref providercontract.RawPayloadRef, salt string) (canonical.Provenance, error) {
	evidenceID := canonical.EvidenceID("evd_" + canonical.DigestBytes([]byte("treasury-evidence\x00" + string(ref.ID))).Value[:24])
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
	return canonical.Provenance{ContractVersion: canonical.ProvenanceContractV1, ID: "pvn_" + canonical.DigestBytes([]byte("treasury-provenance\x00" + salt)).Value[:24], Inputs: inputs, InputFingerprint: fingerprint, Producer: canonical.ComponentIdentity{ID: NormalizerID, Kind: canonical.ComponentKindNormalizer, Name: "Treasury par yield deterministic normalizer", Version: canonical.VersionIdentity{Namespace: "jax.treasury.normalizer", Value: NormalizerVersion}, Provider: &provider}}, nil
}

func errorsf(format string, args ...any) error { return fmt.Errorf(format, args...) }
