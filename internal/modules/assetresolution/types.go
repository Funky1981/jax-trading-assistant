package assetresolution

import "time"

const (
	StatusResolved   = "resolved"
	StatusAmbiguous  = "ambiguous"
	StatusUnresolved = "unresolved"
	StatusRejected   = "rejected"
)

type AliasRule struct {
	CanonicalEntity string   `json:"canonical_entity"`
	Aliases         []string `json:"aliases"`
	Symbol          string   `json:"symbol"`
	AssetClass      string   `json:"asset_class"`
	Exchange        string   `json:"exchange"`
	EffectiveFrom   string   `json:"effective_from"`
	EffectiveTo     string   `json:"effective_to,omitempty"`
	Rationale       string   `json:"rationale"`
	Provenance      string   `json:"provenance"`
	RequiresContext bool     `json:"requires_context,omitempty"`
}

type ProxyRule struct {
	Key           string   `json:"key"`
	Symbol        string   `json:"symbol"`
	Benchmark     string   `json:"benchmark"`
	AssetClass    string   `json:"asset_class"`
	MappingType   string   `json:"mapping_type"`
	Reason        string   `json:"reason"`
	SourceNames   []string `json:"source_names,omitempty"`
	SourceHosts   []string `json:"source_hosts,omitempty"`
	EventTypes    []string `json:"event_types,omitempty"`
	HeadlineTerms []string `json:"headline_terms,omitempty"`
}

type Ruleset struct {
	Version               string      `json:"version"`
	Aliases               []AliasRule `json:"aliases"`
	FinancialContextTerms []string    `json:"financial_context_terms"`
	MaterialEventTerms    []string    `json:"material_event_terms"`
	Proxies               []ProxyRule `json:"proxies"`
}

type Input struct {
	EventID         string
	NormalizedID    string
	Headline        string
	Summary         string
	SourceName      string
	SourceURL       string
	EventType       string
	PublicationAt   time.Time
	ReceiptAt       time.Time
	ExplicitSymbols []string
	ExplicitReason  string
	ExplicitMethods []string
}

// IssuerInput is deliberately narrower than Input. Issuer resolution uses the
// model-recognized issuer identity plus receipt-time anchors and never falls
// through to event-text proxy rules.
type IssuerInput struct {
	IssuerName    string
	PublicationAt time.Time
	ReceiptAt     time.Time
}

type Result struct {
	Status                      string         `json:"status"`
	Symbol                      string         `json:"symbol,omitempty"`
	Benchmark                   string         `json:"benchmark,omitempty"`
	MappingType                 string         `json:"mappingType"`
	Relationship                string         `json:"relationship"`
	ConfidenceClass             string         `json:"confidenceClass"`
	Reason                      string         `json:"reason"`
	SourceFields                []string       `json:"sourceFields"`
	SourceValues                map[string]any `json:"sourceValues"`
	RulesetVersion              string         `json:"rulesetVersion"`
	CanonicalEntity             string         `json:"canonicalEntity,omitempty"`
	MatchedAlias                string         `json:"matchedAlias,omitempty"`
	AssetClass                  string         `json:"assetClass,omitempty"`
	Exchange                    string         `json:"exchange,omitempty"`
	EffectiveFrom               *time.Time     `json:"effectiveFrom,omitempty"`
	EffectiveTo                 *time.Time     `json:"effectiveTo,omitempty"`
	AmbiguityReason             string         `json:"ambiguityReason,omitempty"`
	RejectionReason             string         `json:"rejectionReason,omitempty"`
	KnowableAtOperationalAnchor bool           `json:"knowableAtOperationalAnchor"`
	MaterialEvent               bool           `json:"materialEvent"`
}
