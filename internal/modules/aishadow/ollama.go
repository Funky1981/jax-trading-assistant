package aishadow

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type OllamaClient struct {
	baseURL     string
	model       string
	temperature float64
	seed        int64
	http        *http.Client
}

func NewOllamaClient(config Config) *OllamaClient {
	return &OllamaClient{
		baseURL: config.BaseURL, model: config.Model, temperature: config.Temperature,
		seed: config.Seed, http: &http.Client{Timeout: config.Timeout},
	}
}

func (c *OllamaClient) Complete(request ProviderRequest) (ProviderResponse, error) {
	payload := struct {
		Model    string          `json:"model"`
		Messages []ollamaMessage `json:"messages"`
		Stream   bool            `json:"stream"`
		Think    bool            `json:"think"`
		Format   map[string]any  `json:"format"`
		Options  map[string]any  `json:"options"`
	}{
		Model: c.model, Stream: false, Think: false, Format: request.Schema,
		Messages: []ollamaMessage{{Role: "system", Content: request.System}, {Role: "user", Content: request.User}},
		Options:  map[string]any{"temperature": c.temperature, "seed": c.seed},
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return ProviderResponse{}, fmt.Errorf("marshal Ollama request: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), c.http.Timeout)
	defer cancel()
	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/chat", bytes.NewReader(raw))
	if err != nil {
		return ProviderResponse{}, fmt.Errorf("create Ollama request: %w", err)
	}
	httpRequest.Header.Set("Content-Type", "application/json")
	response, err := c.http.Do(httpRequest)
	if err != nil {
		return ProviderResponse{}, fmt.Errorf("Ollama request: %w", err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, 2<<20))
	if err != nil {
		return ProviderResponse{}, fmt.Errorf("read Ollama response: %w", err)
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return ProviderResponse{}, fmt.Errorf("Ollama HTTP %d: %s", response.StatusCode, string(body))
	}
	var decoded struct {
		Model   string        `json:"model"`
		Message ollamaMessage `json:"message"`
		Error   string        `json:"error"`
	}
	if err := json.Unmarshal(body, &decoded); err != nil {
		return ProviderResponse{}, fmt.Errorf("decode Ollama response: %w", err)
	}
	if decoded.Error != "" {
		return ProviderResponse{}, fmt.Errorf("Ollama error: %s", decoded.Error)
	}
	return ProviderResponse{Content: decoded.Message.Content, ModelIdentifier: decoded.Model}, nil
}

type ollamaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

func VerifyOllama(config Config) error {
	_, err := InspectOllamaModel(config)
	return err
}

func InspectOllamaModel(config Config) (DiagnosticModelIdentity, error) {
	client := &http.Client{Timeout: minDuration(config.Timeout, 10*time.Second)}
	response, err := client.Get(config.BaseURL + "/api/tags")
	if err != nil {
		return DiagnosticModelIdentity{}, fmt.Errorf("Ollama availability: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return DiagnosticModelIdentity{}, fmt.Errorf("Ollama availability returned HTTP %d", response.StatusCode)
	}
	var tags struct {
		Models []struct {
			Name    string `json:"name"`
			Model   string `json:"model"`
			Digest  string `json:"digest"`
			Details struct {
				Format            string `json:"format"`
				Family            string `json:"family"`
				ParameterSize     string `json:"parameter_size"`
				QuantizationLevel string `json:"quantization_level"`
			} `json:"details"`
		} `json:"models"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 2<<20)).Decode(&tags); err != nil {
		return DiagnosticModelIdentity{}, fmt.Errorf("decode Ollama model list: %w", err)
	}
	for _, model := range tags.Models {
		if model.Name == config.Model || model.Model == config.Model {
			return DiagnosticModelIdentity{
				Name: config.Model, Digest: model.Digest, Format: model.Details.Format,
				Family: model.Details.Family, ParameterSize: model.Details.ParameterSize,
				QuantizationLevel: model.Details.QuantizationLevel,
			}, nil
		}
	}
	return DiagnosticModelIdentity{}, fmt.Errorf("configured Ollama model %q is not installed", config.Model)
}

func minDuration(a, b time.Duration) time.Duration {
	if a < b {
		return a
	}
	return b
}
