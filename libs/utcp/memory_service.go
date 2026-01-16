package utcp

import (
	"context"

	"jax-trading-assistant/libs/contracts"
)

type MemoryService struct {
	client Client
}

func NewMemoryService(c Client) *MemoryService {
	return &MemoryService{client: c}
}

func (s *MemoryService) Retain(ctx context.Context, in contracts.MemoryRetainRequest) (contracts.MemoryRetainResponse, error) {
	var out contracts.MemoryRetainResponse
	if err := s.client.CallTool(ctx, MemoryProviderID, ToolMemoryRetain, in, &out); err != nil {
		return contracts.MemoryRetainResponse{}, err
	}
	return out, nil
}

func (s *MemoryService) Recall(ctx context.Context, in contracts.MemoryRecallRequest) (contracts.MemoryRecallResponse, error) {
	var out contracts.MemoryRecallResponse
	if err := s.client.CallTool(ctx, MemoryProviderID, ToolMemoryRecall, in, &out); err != nil {
		return contracts.MemoryRecallResponse{}, err
	}
	return out, nil
}

func (s *MemoryService) Reflect(ctx context.Context, in contracts.MemoryReflectRequest) (contracts.MemoryReflectResponse, error) {
	var out contracts.MemoryReflectResponse
	if err := s.client.CallTool(ctx, MemoryProviderID, ToolMemoryReflect, in, &out); err != nil {
		return contracts.MemoryReflectResponse{}, err
	}
	return out, nil
}
