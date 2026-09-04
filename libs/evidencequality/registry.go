package evidencequality

import "jax-trading-assistant/libs/contracts/canonical"

const (
	VIXMappingID            = "eqm_cboe_vix_fred_vixcls"
	VIXMappingVersion       = "v1"
	VIXMeasurement          = "VIX_DAILY_CLOSE"
	VIXObservationSemantics = "CLOSE_VALUE_BY_OBSERVATION_DATE"
	VIXReleaseSemantics     = "NO_RELEASE_INSTANT_ASSERTED"
)

// ApprovedVIXMapping is the sole Phase-03 cross-source equivalence mapping.
// FRED VIXCLS is explicitly treated as a redistribution of Cboe-originated
// VIX data, not as an independent corroborating origin.
func ApprovedVIXMapping() Mapping {
	policy := ComparisonPolicy{
		ContractVersion:  ComparisonPolicyContractV1,
		ID:               "eqp_vix_exact_decimal",
		Version:          canonical.VersionIdentity{Namespace: "jax.evidencequality.policy", Value: "vix-distribution-exact-decimal-v1"},
		NumericMode:      NumericExactDecimal,
		Tolerance:        "0",
		RequireDate:      true,
		RequireInfoState: true,
		RequireRealtime:  true,
	}
	return Mapping{
		ContractVersion: MappingContractV1,
		ID:              VIXMappingID,
		Version:         canonical.VersionIdentity{Namespace: "jax.evidencequality.mapping", Value: VIXMappingVersion},
		LeftSeries: SourceSeriesIdentity{
			Provider:         canonical.ProviderIdentity{ID: "pvd_cboe", Namespace: "cboe.index_data", ExternalID: &canonical.ExternalID{Namespace: "provider.slug", Value: "cboe"}},
			Source:           canonical.SourceIdentity{ID: "src_cboe_vix_daily_history", Kind: canonical.SourceKindExchange},
			ProviderSeriesID: "VIX",
		},
		RightSeries: SourceSeriesIdentity{
			Provider:         canonical.ProviderIdentity{ID: "pvd_fred", Namespace: "fred.alfred.data_api", ExternalID: &canonical.ExternalID{Namespace: "provider.slug", Value: "fred-alfred"}},
			Source:           canonical.SourceIdentity{ID: "src_fred_economic_data", Kind: canonical.SourceKindDataset},
			ProviderSeriesID: "VIXCLS",
		},
		LeftSemantics: SeriesSemantics{
			MeasurementType: VIXMeasurement, Units: "index points", FrequencyCode: "D", ObservationSemantics: VIXObservationSemantics, ReleaseSemantics: VIXReleaseSemantics,
		},
		RightSemantics: SeriesSemantics{
			MeasurementType: VIXMeasurement, Units: "Index", FrequencyCode: "D", ObservationSemantics: VIXObservationSemantics, ReleaseSemantics: VIXReleaseSemantics,
		},
		MeasurementIdentity: VIXMeasurement,
		CanonicalUnits:      "index points",
		Lineage:             LineageSameOriginRedistributed,
		Policy:              policy,
		AuthorityNotes:      "FRED VIXCLS identifies Chicago Board Options Exchange as the source; this mapping checks distribution consistency and does not count independent origins.",
	}
}

func NewApprovedRegistry() (*Registry, error) {
	return NewRegistry([]Mapping{ApprovedVIXMapping()})
}
