package llmcontext

import "context"

type LLMProviderClient interface {
	Complete(ctx context.Context, pkg PromptPackage) (LLMResult, error)
}

type NoopProvider struct {
	ResponseText string
}

func (p NoopProvider) Complete(_ context.Context, pkg PromptPackage) (LLMResult, error) {
	text := p.ResponseText
	if text == "" {
		text = "noop"
	}
	return LLMResult{
		CorrelationID: pkg.CorrelationID,
		Text:          text,
		InputTokens:   pkg.EstimatedInputTokens,
		OutputTokens:  1,
	}, nil
}
