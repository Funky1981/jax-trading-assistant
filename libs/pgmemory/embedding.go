// Package pgmemory provides embedding providers used by the pgmemory Store.
package pgmemory

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io"
	"math"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// EmbeddingProvider generates a dense vector embedding for a text string.
// The returned slice length must match the configured vector dimension (1536).
type EmbeddingProvider interface {
	Embed(ctx context.Context, text string) ([]float32, error)
}

const EmbeddingDimensions = 1536

type EmbeddingProviderName string

const (
	EmbeddingProviderLocal  EmbeddingProviderName = "local"
	EmbeddingProviderOpenAI EmbeddingProviderName = "openai"
)

// EmbedderConfig selects and configures the runtime embedding provider.
type EmbedderConfig struct {
	Provider EmbeddingProviderName
	BaseURL  string
	APIKey   string
	Model    string
}

// OpenAIEmbedderConfig holds configuration for an OpenAI-compatible embedder.
type OpenAIEmbedderConfig struct {
	// BaseURL is the API root, e.g. "https://api.openai.com" or an Azure endpoint.
	// Defaults to "https://api.openai.com".
	BaseURL string
	// APIKey is the Bearer token sent in the Authorization header.
	APIKey string
	// Model is the embedding model identifier.
	// Defaults to "text-embedding-3-small".
	Model string
}

type localEmbedder struct{}

type openAIEmbedder struct {
	baseURL    string
	apiKey     string
	model      string
	httpClient *http.Client
}

// NormalizeProvider resolves the configured embedding provider, defaulting to local.
func NormalizeProvider(raw string) EmbeddingProviderName {
	switch EmbeddingProviderName(strings.ToLower(strings.TrimSpace(raw))) {
	case "", EmbeddingProviderLocal:
		return EmbeddingProviderLocal
	case EmbeddingProviderOpenAI:
		return EmbeddingProviderOpenAI
	default:
		return EmbeddingProviderName(strings.ToLower(strings.TrimSpace(raw)))
	}
}

// ValidateEmbedderConfig ensures runtime embedding configuration is valid for
// the selected provider before the service starts accepting traffic.
func ValidateEmbedderConfig(cfg EmbedderConfig) error {
	switch provider := NormalizeProvider(string(cfg.Provider)); provider {
	case EmbeddingProviderLocal:
		return nil
	case EmbeddingProviderOpenAI:
		return ValidateOpenAIEmbedderConfig(OpenAIEmbedderConfig{
			BaseURL: cfg.BaseURL,
			APIKey:  cfg.APIKey,
			Model:   cfg.Model,
		})
	default:
		return fmt.Errorf("EMBEDDING_PROVIDER must be one of: local, openai")
	}
}

// NewEmbedder returns an EmbeddingProvider for the selected runtime mode.
func NewEmbedder(cfg EmbedderConfig) (EmbeddingProvider, error) {
	if err := ValidateEmbedderConfig(cfg); err != nil {
		return nil, err
	}

	switch NormalizeProvider(string(cfg.Provider)) {
	case EmbeddingProviderLocal:
		return NewLocalEmbedder(), nil
	case EmbeddingProviderOpenAI:
		return NewOpenAIEmbedder(OpenAIEmbedderConfig{
			BaseURL: cfg.BaseURL,
			APIKey:  cfg.APIKey,
			Model:   cfg.Model,
		}), nil
	default:
		return nil, fmt.Errorf("EMBEDDING_PROVIDER must be one of: local, openai")
	}
}

// NewLocalEmbedder returns an in-process deterministic embedder for local dev/test.
func NewLocalEmbedder() EmbeddingProvider {
	return &localEmbedder{}
}

// NewOpenAIEmbedder returns an EmbeddingProvider backed by an OpenAI-compatible API.
func NewOpenAIEmbedder(cfg OpenAIEmbedderConfig) EmbeddingProvider {
	if strings.TrimSpace(cfg.BaseURL) == "" {
		cfg.BaseURL = "https://api.openai.com"
	}
	if strings.TrimSpace(cfg.Model) == "" {
		cfg.Model = "text-embedding-3-small"
	}
	return &openAIEmbedder{
		baseURL:    strings.TrimRight(cfg.BaseURL, "/"),
		apiKey:     cfg.APIKey,
		model:      cfg.Model,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (e *localEmbedder) Embed(_ context.Context, text string) ([]float32, error) {
	vec := make([]float32, EmbeddingDimensions)
	tokens := tokenize(text)
	if len(tokens) == 0 {
		return vec, nil
	}

	for _, token := range tokens {
		addLocalFeature(vec, token, 1.0)
	}
	for i := 0; i < len(tokens)-1; i++ {
		addLocalFeature(vec, tokens[i]+"_"+tokens[i+1], 0.5)
	}

	var norm float64
	for _, value := range vec {
		norm += float64(value * value)
	}
	if norm == 0 {
		return vec, nil
	}
	scale := float32(1.0 / math.Sqrt(norm))
	for i := range vec {
		vec[i] *= scale
	}
	return vec, nil
}

// ValidateOpenAIEmbedderConfig ensures runtime embedding configuration is present
// before the service starts accepting traffic.
func ValidateOpenAIEmbedderConfig(cfg OpenAIEmbedderConfig) error {
	if strings.TrimSpace(cfg.APIKey) == "" {
		return fmt.Errorf("OPENAI_API_KEY is required for memory embeddings")
	}
	baseURL := strings.TrimSpace(cfg.BaseURL)
	if baseURL == "" {
		baseURL = "https://api.openai.com"
	}
	parsed, err := url.Parse(baseURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return fmt.Errorf("OPENAI_BASE_URL must be a valid absolute URL")
	}
	return nil
}

func tokenize(text string) []string {
	cleaned := strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z':
			return r
		case r >= 'A' && r <= 'Z':
			return r + ('a' - 'A')
		case r >= '0' && r <= '9':
			return r
		default:
			return ' '
		}
	}, text)
	return strings.Fields(cleaned)
}

func addLocalFeature(vec []float32, feature string, weight float32) {
	hasher := fnv.New64a()
	_, _ = hasher.Write([]byte(feature))
	sum := hasher.Sum64()

	primary := int(sum % uint64(EmbeddingDimensions))
	secondary := int((sum >> 21) % uint64(EmbeddingDimensions))
	sign := float32(1.0)
	if sum&1 == 1 {
		sign = -1.0
	}

	vec[primary] += sign * weight
	if secondary != primary {
		vec[secondary] += sign * (weight * 0.5)
	}
}

type embeddingRequestBody struct {
	Input string `json:"input"`
	Model string `json:"model"`
}

type embeddingResponseBody struct {
	Data []struct {
		Embedding []float32 `json:"embedding"`
	} `json:"data"`
}

// Embed sends text to the embeddings endpoint and returns the first embedding vector.
func (e *openAIEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	body, err := json.Marshal(embeddingRequestBody{Input: text, Model: e.model})
	if err != nil {
		return nil, fmt.Errorf("pgmemory embed: marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, e.baseURL+"/v1/embeddings", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("pgmemory embed: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if e.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+e.apiKey)
	}

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("pgmemory embed: http: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("pgmemory embed: status %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}

	var out embeddingResponseBody
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("pgmemory embed: decode response: %w", err)
	}
	if len(out.Data) == 0 || len(out.Data[0].Embedding) == 0 {
		return nil, fmt.Errorf("pgmemory embed: empty embedding in response")
	}
	if len(out.Data[0].Embedding) != EmbeddingDimensions {
		return nil, fmt.Errorf("pgmemory embed: unexpected embedding dimension %d", len(out.Data[0].Embedding))
	}
	return out.Data[0].Embedding, nil
}
