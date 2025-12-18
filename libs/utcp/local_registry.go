package utcp

import (
	"context"
	"fmt"
	"sync"
)

type LocalTool func(ctx context.Context, input any, output any) error

type LocalRegistry struct {
	mu    sync.RWMutex
	tools map[string]map[string]LocalTool
}

func NewLocalRegistry() *LocalRegistry {
	return &LocalRegistry{tools: make(map[string]map[string]LocalTool)}
}

func (r *LocalRegistry) Register(providerID, toolName string, tool LocalTool) error {
	if providerID == "" {
		return fmt.Errorf("register local tool: providerID is required")
	}
	if toolName == "" {
		return fmt.Errorf("register local tool: toolName is required")
	}
	if tool == nil {
		return fmt.Errorf("register local tool: tool is required")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	providerTools, ok := r.tools[providerID]
	if !ok {
		providerTools = make(map[string]LocalTool)
		r.tools[providerID] = providerTools
	}

	if _, exists := providerTools[toolName]; exists {
		return fmt.Errorf("register local tool: %s/%s already registered", providerID, toolName)
	}

	providerTools[toolName] = tool
	return nil
}

func (r *LocalRegistry) Call(ctx context.Context, providerID, toolName string, input any, output any) error {
	r.mu.RLock()
	providerTools, ok := r.tools[providerID]
	if !ok {
		r.mu.RUnlock()
		return fmt.Errorf("local provider not found: %s", providerID)
	}
	tool, ok := providerTools[toolName]
	r.mu.RUnlock()
	if !ok {
		return fmt.Errorf("local tool not found: %s/%s", providerID, toolName)
	}

	return tool(ctx, input, output)
}
