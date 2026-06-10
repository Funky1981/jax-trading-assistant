package harness

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

type Model interface {
	Complete(ctx context.Context, messages []Message) (string, error)
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type Service struct {
	policy   Policy
	registry *Registry
	prompts  *PromptBuilder
	validate *Validator
	traces   TraceSink
	model    Model
}

func NewService(policy Policy, registry *Registry, prompts *PromptBuilder, validate *Validator, traces TraceSink, model Model) *Service {
	return &Service{
		policy:   policy,
		registry: registry,
		prompts:  prompts,
		validate: validate,
		traces:   traces,
		model:    model,
	}
}

func (s *Service) AnswerWithEvidence(ctx context.Context, sessionID, question string, requestedTool string, args json.RawMessage) (string, error) {
	bundle := EvidenceBundle{}
	toolNames := []string{}
	system := s.prompts.SystemPrompt(s.policy, toolNames)

	if requestedTool != "" {
		def, ok := s.registry.Get(requestedTool)
		if !ok {
			return "", fmt.Errorf("tool not found: %s", requestedTool)
		}
		if err := s.policy.CheckToolAllowed(def); err != nil {
			return "", err
		}
		raw, err := def.Handler(ctx, args)
		if err != nil {
			return "", err
		}
		toolNames = append(toolNames, def.Name)
		bundle.Add(EvidenceItem{
			SourceTool:    def.Name,
			EvidenceLevel: def.EvidenceLevel,
			Freshness:     def.FreshnessExpectation,
			Summary:       "tool result captured",
			Raw:           raw,
		})
	}

	messages := []Message{
		{Role: "system", Content: system},
		{Role: "user", Content: question},
	}

	answer, err := s.model.Complete(ctx, messages)
	if err != nil {
		return "", err
	}

	vr := s.validate.ValidateAnswer(s.policy, bundle, answer)
	if s.traces != nil {
		_ = s.traces.WriteTrace(Trace{
			TraceID:        fmt.Sprintf("trace-%d", time.Now().UnixNano()),
			SessionID:      sessionID,
			Question:       question,
			ToolNames:      toolNames,
			ValidatorNotes: vr.Reasons,
			CreatedAt:      time.Now().UTC(),
		})
	}
	if err := s.validate.Must(vr); err != nil {
		return "", err
	}
	return answer, nil
}
