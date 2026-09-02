package fred

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"jax-trading-assistant/libs/contracts/canonical"
	providercontract "jax-trading-assistant/libs/contracts/provider"
	"jax-trading-assistant/libs/macroevidence"
)

const (
	ProviderID         = "pvd_fred"
	ProviderNamespace  = "fred.alfred.data_api"
	SourceID           = "src_fred_economic_data"
	RawSchemaNamespace = "fred.api.json"
	RawSchemaValue     = "documented-v1"
	AdapterVersion     = "1.0.0"
	NormalizerVersion  = "1.0.0"
	NormalizerID       = "cmp_fred_macro_normalizer"
	APIKeyEnvironment  = "FRED_API_KEY"
	DefaultBaseURL     = "https://api.stlouisfed.org/fred"
	DefaultMaxPages    = 100
	DefaultPageSize    = 1000
	DefaultMaxResponse = 8 << 20
)

var (
	ProviderIdentity = canonical.ProviderIdentity{
		ID: ProviderID, Namespace: ProviderNamespace,
		ExternalID: &canonical.ExternalID{Namespace: "provider.slug", Value: "fred-alfred"},
	}
	SourceIdentity  = canonical.SourceIdentity{ID: SourceID, Kind: canonical.SourceKindDataset}
	seriesIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$`)
)

// These aliases make the adapter API convenient while keeping research-facing
// macro contracts in a provider-neutral package.
type Date = macroevidence.Date
type RealtimePeriod = macroevidence.RealtimePeriod
type MacroSeriesID = macroevidence.MacroSeriesID
type MacroSeries = macroevidence.MacroSeries
type InformationStateMode = macroevidence.InformationStateMode
type InformationState = macroevidence.InformationState
type MacroValue = macroevidence.MacroValue
type MacroObservation = macroevidence.MacroObservation
type CompletenessState = macroevidence.CompletenessState
type PageInfo = macroevidence.PageInfo

const (
	InformationStateCurrent        = macroevidence.InformationStateCurrent
	InformationStateAsOf           = macroevidence.InformationStateAsOf
	InformationStateVintage        = macroevidence.InformationStateVintage
	InformationStateInitialRelease = macroevidence.InformationStateInitialRelease
	CompletenessComplete           = macroevidence.CompletenessComplete
	CompletenessIncomplete         = macroevidence.CompletenessIncomplete
)

func parseDate(value string) (Date, error) {
	date := Date(strings.TrimSpace(value))
	if err := date.Validate(); err != nil {
		return "", err
	}
	return date, nil
}

func validateSeriesID(value string) error {
	if !seriesIDPattern.MatchString(value) {
		return fmt.Errorf("FRED series ID is missing or malformed")
	}
	return nil
}

type Config struct {
	BaseURL          string
	APIKey           string
	MaxResponseBytes int64
	MaxPages         int
}

func (config Config) Validate() error {
	parsed, err := url.Parse(strings.TrimSpace(config.BaseURL))
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return fmt.Errorf("FRED base URL must be an HTTPS origin without credentials, query, or fragment")
	}
	if len(config.APIKey) != 32 || !regexp.MustCompile(`^[a-z0-9]{32}$`).MatchString(config.APIKey) {
		return fmt.Errorf("FRED API key must be a 32-character lowercase alphanumeric value")
	}
	if config.MaxResponseBytes <= 0 || config.MaxResponseBytes > 64<<20 {
		return fmt.Errorf("FRED maximum response size is invalid")
	}
	if config.MaxPages < 1 || config.MaxPages > 1000 {
		return fmt.Errorf("FRED maximum page count is invalid")
	}
	return nil
}

type Dependencies struct {
	Registry *providercontract.Registry
	Executor *providercontract.OperationalExecutor
	Store    providercontract.RawPayloadStore
}

type SeriesRequest struct {
	SeriesID         string
	InformationState InformationState
	PayloadID        providercontract.RawPayloadID
	Retention        providercontract.RawPayloadRetentionPolicy
}

type SeriesResult struct {
	Execution providercontract.ExecutionResult
	Raw       providercontract.RawPayloadDescriptor
	Series    MacroSeries
}

type VintageDatesRequest struct {
	SeriesID      string
	PayloadID     providercontract.RawPayloadID
	Retention     providercontract.RawPayloadRetentionPolicy
	RealtimeStart *Date
	RealtimeEnd   *Date
	PageSize      int
	MaxPages      int
}

type VintageDatesResult struct {
	Executions             []providercontract.ExecutionResult
	RawPayloads            []providercontract.RawPayloadDescriptor
	VintageDates           []Date
	ProviderRealtimePeriod *RealtimePeriod
	Page                   PageInfo
	Completeness           CompletenessState
}

type ObservationsRequest struct {
	Series           MacroSeries
	ObservationStart Date
	ObservationEnd   Date
	InformationState InformationState
	PayloadID        providercontract.RawPayloadID
	Retention        providercontract.RawPayloadRetentionPolicy
	PageSize         int
	MaxPages         int
}

type ObservationsResult struct {
	Executions             []providercontract.ExecutionResult
	RawPayloads            []providercontract.RawPayloadDescriptor
	Observations           []MacroObservation
	ProviderRealtimePeriod *RealtimePeriod
	Page                   PageInfo
	Completeness           CompletenessState
}

type AcquisitionRequest struct {
	SeriesID             string
	SeriesPayloadID      providercontract.RawPayloadID
	ObservationPayloadID providercontract.RawPayloadID
	VintagePayloadID     providercontract.RawPayloadID
	ObservationStart     Date
	ObservationEnd       Date
	InformationState     InformationState
	Retention            providercontract.RawPayloadRetentionPolicy
	IncludeVintageDates  bool
	PageSize             int
	MaxPages             int
}

type AcquisitionResult struct {
	Series       MacroSeries
	Observations ObservationsResult
	VintageDates VintageDatesResult
	RawPayloads  []providercontract.RawPayloadDescriptor
}
