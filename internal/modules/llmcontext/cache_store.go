package llmcontext

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"
)

type PromptCache interface {
	Get(ctx context.Context, pkg PromptPackage) (LLMResult, bool, error)
	Put(ctx context.Context, pkg PromptPackage, result LLMResult, ttl time.Duration) error
}

func ExactCacheKey(pkg PromptPackage) string {
	payload := strings.Join([]string{
		string(pkg.TaskType),
		pkg.Provider,
		pkg.Model,
		pkg.CacheablePrefix,
		pkg.RetrievedMemory,
		pkg.DynamicContext,
		pkg.ResponseSchema,
	}, "\x00")
	sum := sha256.Sum256([]byte(payload))
	return hex.EncodeToString(sum[:])
}

type memoryCacheEntry struct {
	result    LLMResult
	expiresAt time.Time
}

type MemoryPromptCache struct {
	policy  CachePolicy
	mu      sync.Mutex
	entries map[string]memoryCacheEntry
}

func NewMemoryPromptCache(policy CachePolicy) *MemoryPromptCache {
	return &MemoryPromptCache{policy: policy, entries: map[string]memoryCacheEntry{}}
}

func (c *MemoryPromptCache) Get(_ context.Context, pkg PromptPackage) (LLMResult, bool, error) {
	if !c.policy.Allows(pkg.TaskType) {
		return LLMResult{}, false, nil
	}
	key := ExactCacheKey(pkg)
	c.mu.Lock()
	defer c.mu.Unlock()
	entry, ok := c.entries[key]
	if !ok || time.Now().UTC().After(entry.expiresAt) {
		delete(c.entries, key)
		return LLMResult{}, false, nil
	}
	result := entry.result
	result.CachedTokens = result.InputTokens
	return result, true, nil
}

func (c *MemoryPromptCache) Put(_ context.Context, pkg PromptPackage, result LLMResult, ttl time.Duration) error {
	if !c.policy.Allows(pkg.TaskType) {
		return fmt.Errorf("task type %s is not cache eligible", pkg.TaskType)
	}
	if ttl <= 0 {
		return fmt.Errorf("cache ttl must be positive")
	}
	key := ExactCacheKey(pkg)
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[key] = memoryCacheEntry{result: result, expiresAt: time.Now().UTC().Add(ttl)}
	return nil
}

type PostgresPromptCache struct {
	db     SQLExecutor
	policy CachePolicy
}

func NewPostgresPromptCache(db SQLExecutor, policy CachePolicy) PostgresPromptCache {
	return PostgresPromptCache{db: db, policy: policy}
}

func (c PostgresPromptCache) Get(_ context.Context, _ PromptPackage) (LLMResult, bool, error) {
	return LLMResult{}, false, fmt.Errorf("postgres prompt cache get requires query-capable database handle")
}

func (c PostgresPromptCache) Put(ctx context.Context, pkg PromptPackage, result LLMResult, ttl time.Duration) error {
	if c.db == nil {
		return fmt.Errorf("postgres prompt cache requires database handle")
	}
	if !c.policy.Allows(pkg.TaskType) {
		return fmt.Errorf("task type %s is not cache eligible", pkg.TaskType)
	}
	if ttl <= 0 {
		return fmt.Errorf("cache ttl must be positive")
	}
	sourceHash := sourceHash(pkg)
	responseBytes, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("marshal cache result: %w", err)
	}
	_, err = c.db.ExecContext(ctx, `
		INSERT INTO llm_prompt_cache (
			cache_key, task_type, provider, model, correlation_id,
			source_hash, response_text, input_tokens, output_tokens,
			response_json, expires_at
		) VALUES (
			$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11
		)
		ON CONFLICT (cache_key) DO UPDATE SET
			correlation_id = EXCLUDED.correlation_id,
			response_text = EXCLUDED.response_text,
			input_tokens = EXCLUDED.input_tokens,
			output_tokens = EXCLUDED.output_tokens,
			response_json = EXCLUDED.response_json,
			expires_at = EXCLUDED.expires_at,
			updated_at = NOW()
	`, ExactCacheKey(pkg), pkg.TaskType, pkg.Provider, pkg.Model, pkg.CorrelationID,
		sourceHash, result.Text, result.InputTokens, result.OutputTokens, responseBytes, time.Now().UTC().Add(ttl))
	if err != nil {
		return fmt.Errorf("llmcontext.PostgresPromptCache.Put: %w", err)
	}
	return nil
}

func sourceHash(pkg PromptPackage) string {
	sum := sha256.Sum256([]byte(strings.Join([]string{pkg.CacheablePrefix, pkg.RetrievedMemory, pkg.DynamicContext, pkg.ResponseSchema}, "\x00")))
	return hex.EncodeToString(sum[:])
}

var _ SQLExecutor = (*sql.DB)(nil)
