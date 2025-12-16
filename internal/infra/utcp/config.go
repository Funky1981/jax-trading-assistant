package utcp

type ProviderConfig struct {
	ID        string `json:"id"`
	Transport string `json:"transport"`
	Endpoint  string `json:"endpoint,omitempty"`
}

type ProvidersConfig struct {
	Providers []ProviderConfig `json:"providers"`
}
