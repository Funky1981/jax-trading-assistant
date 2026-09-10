package llmcontext

import (
	"context"
	"fmt"
	"strings"

	"jax-trading-assistant/internal/modules/inference"
)

// InferenceProvider adapts the Jax-owned transport to the context, routing,
// budget, and usage-accounting service. It intentionally sends no tools.
type InferenceProvider struct {
	client   inference.Client
	provider string
	model    string
}

func NewInferenceProvider(client inference.Client, provider, model string) (InferenceProvider, error) {
	if client == nil {
		return InferenceProvider{}, fmt.Errorf("llmcontext: inference client required")
	}
	if strings.TrimSpace(provider) == "" || strings.TrimSpace(model) == "" {
		return InferenceProvider{}, fmt.Errorf("llmcontext: provider and model required")
	}
	return InferenceProvider{client: client, provider: provider, model: model}, nil
}

func (p InferenceProvider) Complete(ctx context.Context, pkg PromptPackage) (LLMResult, error) {
	response, err := p.client.Complete(ctx, inference.Request{
		Model: pkg.Model,
		Messages: []inference.Message{
			{Role: "system", Content: pkg.CacheablePrefix},
			{Role: "user", Content: strings.Join(nonEmpty([]string{pkg.RetrievedMemory, pkg.DynamicContext, pkg.ResponseSchema}), "\n\n")},
		},
		MaxTokens: pkg.EstimatedOutputTokens,
	})
	if err != nil {
		return LLMResult{}, err
	}
	provider := string(response.Provider)
	if provider == "" {
		provider = p.provider
	}
	model := response.Model
	if model == "" {
		model = p.model
	}
	return LLMResult{
		CorrelationID: pkg.CorrelationID,
		RequestID:     response.RequestID,
		Provider:      provider,
		Model:         model,
		Text:          response.Text,
		InputTokens:   response.Usage.InputTokens,
		OutputTokens:  response.Usage.OutputTokens,
		CachedTokens:  response.Usage.CachedInputTokens,
	}, nil
}
