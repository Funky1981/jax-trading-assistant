// Package bls provides a raw-first adapter for the official BLS national
// office release calendar. It supplies schedule evidence only.
package bls

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"jax-trading-assistant/libs/contracts/canonical"
	providercontract "jax-trading-assistant/libs/contracts/provider"
	"jax-trading-assistant/libs/releaseevidence"
)

const (
	ProviderID         = "pvd_bls"
	ProviderNamespace  = "bls.release_calendar"
	SourceID           = "src_bls_release_calendar"
	DefaultCalendarURL = "https://www.bls.gov/schedule/news_release/bls.ics"
	RawSchemaNamespace = "bls.calendar.ics"
	RawSchemaValue     = "v1"
	AdapterVersion     = "1.0.0"
	NormalizerVersion  = "1.0.0"
	NormalizerID       = "cmp_bls_release_normalizer"
	DefaultMaxResponse = 8 << 20
	BLSReleaseTimezone = "America/New_York"
)

var (
	ProviderIdentity = canonical.ProviderIdentity{ID: ProviderID, Namespace: ProviderNamespace, ExternalID: &canonical.ExternalID{Namespace: "provider.slug", Value: "bls"}}
	SourceIdentity   = canonical.SourceIdentity{ID: SourceID, Kind: canonical.SourceKindPublisher}
)

type Config struct {
	CalendarURL      string
	MaxResponseBytes int64
}

func (config Config) Validate() error {
	parsed, err := url.Parse(strings.TrimSpace(config.CalendarURL))
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return fmt.Errorf("BLS calendar URL must be an HTTPS origin without credentials, query, or fragment")
	}
	if config.MaxResponseBytes <= 0 || config.MaxResponseBytes > 64<<20 {
		return fmt.Errorf("BLS maximum response size is invalid")
	}
	return nil
}

type Dependencies struct {
	Registry *providercontract.Registry
	Executor *providercontract.OperationalExecutor
	Store    providercontract.RawPayloadStore
}

type CalendarRequest struct {
	PayloadID providercontract.RawPayloadID
	Retention providercontract.RawPayloadRetentionPolicy
}

type CalendarResult struct {
	Execution providercontract.ExecutionResult
	Raw       providercontract.RawPayloadDescriptor
	Releases  []releaseevidence.EconomicRelease
}

func dateOnly(value string) (releaseevidence.Date, error) {
	date := releaseevidence.Date(strings.TrimSpace(value))
	if err := date.Validate(); err != nil {
		return "", err
	}
	return date, nil
}

func utc(value time.Time) *time.Time {
	value = value.UTC()
	return &value
}
