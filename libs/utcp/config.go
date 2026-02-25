package utcp

type ProviderConfig struct {
	ID              string `json:"id"`
	Transport       string `json:"transport"`
	Endpoint        string `json:"endpoint,omitempty"`
	DataSourceType  string `json:"data_source_type,omitempty"` // real | synthetic | unknown
	SourceProvider  string `json:"source_provider,omitempty"`
	IsSynthetic     bool   `json:"is_synthetic,omitempty"`
	SyntheticReason string `json:"synthetic_reason,omitempty"`
}

type ProvidersConfig struct {
	Providers []ProviderConfig `json:"providers"`
}
