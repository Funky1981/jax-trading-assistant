// Package cboe provides a bounded raw-first adapter for the official Cboe
// historical VIX daily closing-value CSV.
package cboe

import (
	"fmt"
	"net/url"
	"strings"

	"jax-trading-assistant/libs/contracts/canonical"
	providercontract "jax-trading-assistant/libs/contracts/provider"
	"jax-trading-assistant/libs/macroevidence"
)

const (
	ProviderID         = "pvd_cboe"
	ProviderNamespace  = "cboe.index_data"
	SourceID           = "src_cboe_vix_daily_history"
	RawSchemaNamespace = "cboe.vix_history_csv"
	RawSchemaValue     = "v1"
	AdapterVersion     = "1.0.0"
	NormalizerVersion  = "1.0.0"
	NormalizerID       = "cmp_cboe_vix_history_normalizer"
	DefaultHistoryURL  = "https://cdn.cboe.com/api/global/us_indices/daily_prices/VIX_History.csv"
	DefaultMaxResponse = 16 << 20
)

var (
	ProviderIdentity = canonical.ProviderIdentity{ID: ProviderID, Namespace: ProviderNamespace, ExternalID: &canonical.ExternalID{Namespace: "provider.slug", Value: "cboe"}}
	SourceIdentity   = canonical.SourceIdentity{ID: SourceID, Kind: canonical.SourceKindExchange}
)

type Config struct {
	HistoryURL       string
	MaxResponseBytes int64
}

func (config Config) Validate() error {
	parsed, err := url.Parse(strings.TrimSpace(config.HistoryURL))
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return fmt.Errorf("Cboe VIX history URL must be HTTPS without credentials, query, or fragment")
	}
	if config.MaxResponseBytes <= 0 || config.MaxResponseBytes > 64<<20 {
		return fmt.Errorf("Cboe maximum response size is invalid")
	}
	return nil
}

type Dependencies struct {
	Registry *providercontract.Registry
	Executor *providercontract.OperationalExecutor
	Store    providercontract.RawPayloadStore
}

type HistoryRequest struct {
	PayloadID providercontract.RawPayloadID
	Retention providercontract.RawPayloadRetentionPolicy
}

type HistoryResult struct {
	Execution    providercontract.ExecutionResult
	Raw          providercontract.RawPayloadDescriptor
	Series       macroevidence.MacroSeries
	Observations []macroevidence.MacroObservation
	Completeness macroevidence.CompletenessState
}

func DefaultConfig() Config {
	return Config{HistoryURL: DefaultHistoryURL, MaxResponseBytes: DefaultMaxResponse}
}

func rawRepresentation() providercontract.RawRepresentation {
	return providercontract.RawRepresentation{Boundary: providercontract.RawBoundaryProvider, Format: providercontract.RawFormatTabular, Schema: canonical.VersionIdentity{Namespace: RawSchemaNamespace, Value: RawSchemaValue}, MediaType: "text/csv"}
}
