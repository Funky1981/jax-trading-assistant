package harness

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

type Model interface {
	Complete(ctx context.Context, messages []Message) (string, []ToolCall, error)
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type Validator interface {
	ValidateAnswer(policy Policy, bundle EvidenceBundle, answer string) ValidationResult
	Must(result ValidationResult) error
}

type Service struct {
	policy   Policy
	registry *Registry
	prompts  *PromptBuilder
	validate Validator
	traces   TraceSink
	model    Model
}

func NewService(policy Policy, registry *Registry, prompts *PromptBuilder, validate Validator, traces TraceSink, model Model) *Service {
	return &Service{
		policy:   policy,
		registry: registry,
		prompts:  prompts,
		validate: validate,
		traces:   traces,
		model:    model,
	}
}

func (s *Service) AnswerWithEvidence(ctx context.Context, sessionID, question string, history []Message, requestedTool *ToolCall) (string, EvidenceBundle, Trace, error) {
	trace := Trace{
		TraceID:   fmt.Sprintf("trace-%d", time.Now().UnixNano()),
		SessionID: sessionID,
		Question:  question,
		CreatedAt: time.Now().UTC(),
	}
	bundle := EvidenceBundle{}

	if s.model == nil {
		return "", bundle, trace, fmt.Errorf("harness model not configured")
	}

	conversation := make([]Message, 0, len(history)+4)
	conversation = append(conversation, Message{
		Role:    "system",
		Content: s.prompts.SystemPrompt(s.policy, s.toolNames()),
	})
	conversation = append(conversation, history...)
	conversation = append(conversation, Message{Role: "user", Content: question})

	if requestedTool != nil {
		run, evidence, err := s.executeTool(ctx, *requestedTool)
		if err != nil {
			trace.ToolRuns = append(trace.ToolRuns, run)
			trace.ToolNames = append(trace.ToolNames, requestedTool.Name)
			return "", bundle, trace, err
		}
		trace.ToolRuns = append(trace.ToolRuns, run)
		trace.ToolNames = append(trace.ToolNames, requestedTool.Name)
		bundle.Add(evidence)
		conversation = append(conversation, Message{Role: "tool", Content: toolResultMessage(run)})
	}

	answer, err := s.runAdvisoryPass(ctx, conversation, &bundle, &trace)
	if err != nil {
		return "", bundle, trace, err
	}

	finalAnswer := strings.TrimSpace(answer)
	if s.validate != nil {
		result := s.validate.ValidateAnswer(s.policy, bundle, finalAnswer)
		trace.ValidationAttempts = append(trace.ValidationAttempts, ValidationAttempt{
			Attempt:  1,
			Answer:   finalAnswer,
			Accepted: result.OK,
			Reasons:  append([]string(nil), result.Reasons...),
		})
		trace.ValidatorNotes = append(trace.ValidatorNotes, result.Reasons...)
		if !result.OK {
			conversation = append(conversation,
				Message{Role: "assistant", Content: finalAnswer},
				Message{Role: "user", Content: validationFeedback(result)},
			)
			retryAnswer, retryErr := s.runAdvisoryPass(ctx, conversation, &bundle, &trace)
			if retryErr != nil {
				finalAnswer = s.safeRefusal()
				trace.ValidatorNotes = append(trace.ValidatorNotes, "validator retry failed: "+retryErr.Error())
				trace.ValidationAttempts = append(trace.ValidationAttempts, ValidationAttempt{
					Attempt:  2,
					Answer:   finalAnswer,
					Accepted: false,
					Reasons:  []string{"validator retry failed: " + retryErr.Error()},
				})
			} else {
				finalAnswer = strings.TrimSpace(retryAnswer)
				retryResult := s.validate.ValidateAnswer(s.policy, bundle, finalAnswer)
				trace.ValidationAttempts = append(trace.ValidationAttempts, ValidationAttempt{
					Attempt:  2,
					Answer:   finalAnswer,
					Accepted: retryResult.OK,
					Reasons:  append([]string(nil), retryResult.Reasons...),
				})
				trace.ValidatorNotes = append(trace.ValidatorNotes, retryResult.Reasons...)
				if !retryResult.OK {
					finalAnswer = s.safeRefusal()
				}
			}
		}
	}

	trace.FinalAnswer = finalAnswer
	if s.traces != nil {
		_ = s.traces.WriteTrace(trace)
	}
	return finalAnswer, bundle, trace, nil
}

func (s *Service) runAdvisoryPass(ctx context.Context, conversation []Message, bundle *EvidenceBundle, trace *Trace) (string, error) {
	var finalAnswer string
	err := RunBoundedAdvisoryLoop(ctx, s.policy.MaxSteps, func(loopCtx context.Context, step int) (bool, error) {
		answer, calls, err := s.model.Complete(loopCtx, conversation)
		if err != nil {
			return false, err
		}

		trimmedAnswer := strings.TrimSpace(answer)
		if trimmedAnswer != "" {
			conversation = append(conversation, Message{Role: "assistant", Content: trimmedAnswer})
		}

		if len(calls) == 0 {
			if trimmedAnswer == "" {
				return false, fmt.Errorf("model returned neither answer nor tool calls")
			}
			finalAnswer = trimmedAnswer
			return true, nil
		}

		if len(calls) > s.policy.MaxToolCalls {
			return false, fmt.Errorf("tool call limit exceeded: got %d, max %d", len(calls), s.policy.MaxToolCalls)
		}

		for _, call := range calls {
			run, evidence, err := s.executeTool(loopCtx, call)
			trace.ToolRuns = append(trace.ToolRuns, run)
			trace.ToolNames = append(trace.ToolNames, call.Name)
			if err != nil {
				return false, err
			}
			bundle.Add(evidence)
			conversation = append(conversation, Message{Role: "tool", Content: toolResultMessage(run)})
		}

		return false, nil
	})
	if err != nil {
		return "", err
	}
	return finalAnswer, nil
}

func (s *Service) executeTool(ctx context.Context, call ToolCall) (ToolRun, EvidenceItem, error) {
	run := ToolRun{Call: call}
	def, ok := s.registry.Get(call.Name)
	if !ok {
		run.Error = "tool not found: " + call.Name
		return run, EvidenceItem{}, fmt.Errorf("%s", run.Error)
	}
	if err := s.policy.CheckToolAllowed(def); err != nil {
		run.Error = err.Error()
		return run, EvidenceItem{}, err
	}

	toolCtx := ctx
	cancel := func() {}
	if s.policy.ToolTimeout > 0 {
		toolCtx, cancel = context.WithTimeout(ctx, s.policy.ToolTimeout)
	}
	defer cancel()

	raw, err := def.Handler(toolCtx, call.Args)
	if err != nil {
		run.Error = err.Error()
		if toolCtx.Err() != nil {
			run.Error = fmt.Sprintf("tool %s timed out after %s", def.Name, s.policy.ToolTimeout)
			err = errors.New(run.Error)
		}
		return run, EvidenceItem{}, err
	}
	run.Result = raw

	return run, EvidenceItem{
		SourceTool:    def.Name,
		EvidenceLevel: def.EvidenceLevel,
		Freshness:     def.FreshnessExpectation,
		Summary:       "tool result captured",
		Raw:           raw,
	}, nil
}

func (s *Service) toolNames() []string {
	tools := s.registry.AllTools()
	names := make([]string, 0, len(tools))
	for _, tool := range tools {
		names = append(names, tool.Name)
	}
	return names
}

func (s *Service) safeRefusal() string {
	return "I can't provide a supported trading conclusion from the evidence available. I'm advisory only, and the current evidence is too weak or unsupported for a confident claim."
}

func validationFeedback(result ValidationResult) string {
	return "Your previous answer did not satisfy the safety policy. Revise it to stay advisory-only and fix these issues: " + strings.Join(result.Reasons, "; ")
}

func toolResultMessage(run ToolRun) string {
	if run.Error != "" {
		return fmt.Sprintf("tool %s failed: %s", run.Call.Name, run.Error)
	}
	if len(run.Result) == 0 {
		return fmt.Sprintf("tool %s returned no result", run.Call.Name)
	}
	return fmt.Sprintf("tool %s result: %s", run.Call.Name, string(run.Result))
}
