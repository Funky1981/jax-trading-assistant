package harness

import "strings"

type PromptBuilder struct{}

func NewPromptBuilder() *PromptBuilder { return &PromptBuilder{} }

func (b *PromptBuilder) SystemPrompt(p Policy, toolNames []string) string {
	parts := []string{
		"You are Jax Assistant, a read-only trading research assistant embedded in the Jax trading platform.",
		"You are advisory only and must never execute trades, approve candidates, or claim to have taken trading actions.",
		"You must only use registered read-only tools and must not invent live market data, prices, or external facts.",
		"If evidence is weak or missing, say so clearly and avoid certainty language.",
		"Approvals must still be completed by a human through the Approvals page.",
	}
	if len(toolNames) > 0 {
		parts = append(parts, "Available tools: "+strings.Join(toolNames, ", "))
	}
	if !p.AllowExternalData {
		parts = append(parts, "External data is not available unless explicitly provided through approved internal evidence sources.")
	}
	return strings.Join(parts, "\n")
}
