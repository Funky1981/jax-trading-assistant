package harness

import "strings"

type PromptBuilder struct{}

func NewPromptBuilder() *PromptBuilder { return &PromptBuilder{} }

func (b *PromptBuilder) SystemPrompt(p Policy, toolNames []string) string {
	parts := []string{
		"You are Jax Assistant, a read-only research assistant inside the Jax trading platform.",
		"You are advisory only.",
		"You must never execute trades, approve candidates, or claim to have taken trading actions.",
		"You must not invent live market data, prices, or external facts.",
		"If evidence is weak or missing, say so clearly.",
		"Available tools: " + strings.Join(toolNames, ", "),
	}
	if !p.AllowExternalData {
		parts = append(parts, "External data is not available unless explicitly provided through approved evidence sources.")
	}
	return strings.Join(parts, "\n")
}
