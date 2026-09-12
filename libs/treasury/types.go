// Package treasury provides a bounded raw-first adapter for the official
// U.S. Treasury Daily Treasury Par Yield Curve Rates XML feed.
package treasury

import (
	"fmt"
	"net/url"
	"strings"

	"jax-trading-assistant/libs/contracts/canonical"
	providercontract "jax-trading-assistant/libs/contracts/provider"
	"jax-trading-assistant/libs/macroevidence"
)

const (
	ProviderID         = "pvd_treasury"
	ProviderNamespace  = "treasury.interest_rates"
	SourceID           = "src_treasury_daily_par_yield_curve"
	RawSchemaNamespace = "treasury.daily_interest_rate_xml"
	RawSchemaValue     = "v1"
	AdapterVersion     = "1.0.0"
	NormalizerVersion  = "1.0.0"
	NormalizerID       = "cmp_treasury_par_yield_normalizer"
	DefaultBaseURL     = "https://home.treasury.gov"
	DefaultMaxResponse = 8 << 20
)

var (
	ProviderIdentity = canonical.ProviderIdentity{ID: ProviderID, Namespace: ProviderNamespace, ExternalID: &canonical.ExternalID{Namespace: "provider.slug", Value: "treasury"}}
	SourceIdentity   = canonical.SourceIdentity{ID: SourceID, Kind: canonical.SourceKindPublisher}
)

type Config struct {
	BaseURL          string
	MaxResponseBytes int64
}

func (config Config) Validate() error {
	parsed, err := url.Parse(strings.TrimRight(strings.TrimSpace(config.BaseURL), "/"))
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return fmt.Errorf("treasury base URL must be an HTTPS origin without credentials, query, or fragment")
	}
	if config.MaxResponseBytes <= 0 || config.MaxResponseBytes > 64<<20 {
		return fmt.Errorf("treasury maximum response size is invalid")
	}
	return nil
}

type Dependencies struct {
	Registry *providercontract.Registry
	Executor *providercontract.OperationalExecutor
	Store    providercontract.RawPayloadStore
}

type YearRequest struct {
	Year      int
	PayloadID providercontract.RawPayloadID
	Retention providercontract.RawPayloadRetentionPolicy
}

type YieldObservation struct {
	Series      macroevidence.MacroSeries
	Observation macroevidence.MacroObservation
	Tenor       string
}

type YearResult struct {
	Execution     providercontract.ExecutionResult
	Raw           providercontract.RawPayloadDescriptor
	Observations  []YieldObservation
	Completeness  macroevidence.CompletenessState
	RequestedYear int
}

type tenor struct {
	Name string
	XML  string
}

var tenors = []tenor{
	{Name: "1 Mo", XML: "BC_1MONTH"},
	{Name: "1.5 Mo", XML: "BC_1_5MONTH"},
	{Name: "2 Mo", XML: "BC_2MONTH"},
	{Name: "3 Mo", XML: "BC_3MONTH"},
	{Name: "4 Mo", XML: "BC_4MONTH"},
	{Name: "6 Mo", XML: "BC_6MONTH"},
	{Name: "1 Yr", XML: "BC_1YEAR"},
	{Name: "2 Yr", XML: "BC_2YEAR"},
	{Name: "3 Yr", XML: "BC_3YEAR"},
	{Name: "5 Yr", XML: "BC_5YEAR"},
	{Name: "7 Yr", XML: "BC_7YEAR"},
	{Name: "10 Yr", XML: "BC_10YEAR"},
	{Name: "20 Yr", XML: "BC_20YEAR"},
	{Name: "30 Yr", XML: "BC_30YEAR"},
}

func DefaultConfig() Config {
	return Config{BaseURL: DefaultBaseURL, MaxResponseBytes: DefaultMaxResponse}
}

func rawRepresentation() providercontract.RawRepresentation {
	return providercontract.RawRepresentation{
		Boundary:  providercontract.RawBoundaryProvider,
		Format:    providercontract.RawFormatStructuredMessage,
		Schema:    canonical.VersionIdentity{Namespace: RawSchemaNamespace, Value: RawSchemaValue},
		MediaType: "text/xml",
	}
}
