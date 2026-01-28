package strategies

import (
	"fmt"
	"sync"
)

// Registry manages available trading strategies
type Registry struct {
	mu         sync.RWMutex
	strategies map[string]Strategy
	metadata   map[string]StrategyMetadata
}

// NewRegistry creates a new strategy registry
func NewRegistry() *Registry {
	return &Registry{
		strategies: make(map[string]Strategy),
		metadata:   make(map[string]StrategyMetadata),
	}
}

// Register adds a strategy to the registry
func (r *Registry) Register(strategy Strategy, metadata StrategyMetadata) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if strategy == nil {
		return fmt.Errorf("cannot register nil strategy")
	}

	id := strategy.ID()
	if id == "" {
		return fmt.Errorf("strategy ID cannot be empty")
	}

	if _, exists := r.strategies[id]; exists {
		return fmt.Errorf("strategy %s already registered", id)
	}

	r.strategies[id] = strategy
	r.metadata[id] = metadata
	return nil
}

// Get retrieves a strategy by ID
func (r *Registry) Get(id string) (Strategy, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	strategy, exists := r.strategies[id]
	if !exists {
		return nil, fmt.Errorf("strategy %s not found", id)
	}

	return strategy, nil
}

// List returns all registered strategy IDs
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	ids := make([]string, 0, len(r.strategies))
	for id := range r.strategies {
		ids = append(ids, id)
	}
	return ids
}

// GetMetadata returns metadata for a strategy
func (r *Registry) GetMetadata(id string) (StrategyMetadata, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	metadata, exists := r.metadata[id]
	if !exists {
		return StrategyMetadata{}, fmt.Errorf("metadata for strategy %s not found", id)
	}

	return metadata, nil
}

// ListAll returns all strategies with their metadata
func (r *Registry) ListAll() map[string]StrategyMetadata {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[string]StrategyMetadata, len(r.metadata))
	for id, meta := range r.metadata {
		result[id] = meta
	}
	return result
}
