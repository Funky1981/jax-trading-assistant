// Package pgmemory provides an EmbeddingProvider interface and an
// OpenAI-compatible implementation used by the pgmemory Store.
package pgmemory

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// EmbeddingProvider generates a dense vector embedding for a text string.
// The returned slice length must match the configured vector dimension (1536).
type EmbeddingProvider interface {
	Embed(ctx context.Context, text string) ([]float32, error)
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

type openAIEmbedder struct {
	baseURL    string
	apiKey     string
	model      string
	httpClient *http.Client
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
	return out.Data[0].Embedding, nil
}
