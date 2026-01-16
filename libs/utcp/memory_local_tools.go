package utcp

import (
	"context"
	"fmt"
	"log"
	"strings"

	"jax-trading-assistant/libs/contracts"
)

func RegisterMemoryTools(registry *LocalRegistry, store contracts.MemoryStore) error {
	if registry == nil {
		return fmt.Errorf("register memory tools: registry is nil")
	}
	if store == nil {
		return fmt.Errorf("register memory tools: store is nil")
	}

	if err := registry.Register(MemoryProviderID, ToolMemoryRetain, retainMemoryTool(store)); err != nil {
		return err
	}
	if err := registry.Register(MemoryProviderID, ToolMemoryRecall, recallMemoryTool(store)); err != nil {
		return err
	}
	if err := registry.Register(MemoryProviderID, ToolMemoryReflect, reflectMemoryTool(store)); err != nil {
		return err
	}

	return nil
}

func retainMemoryTool(store contracts.MemoryStore) LocalTool {
	return func(ctx context.Context, input any, output any) error {
		var in contracts.MemoryRetainRequest
		if err := decodeJSONLike(input, &in); err != nil {
			return fmt.Errorf("memory.retain: %w", err)
		}
		if strings.TrimSpace(in.Bank) == "" {
			return fmt.Errorf("memory.retain: bank is required")
		}
		in.Item.Tags = contracts.NormalizeMemoryTags(in.Item.Tags)
		if err := contracts.ValidateMemoryItem(in.Item); err != nil {
			return fmt.Errorf("memory.retain: %w", err)
		}

		log.Printf("memory.retain bank=%s type=%s", in.Bank, in.Item.Type)

		id, err := store.Retain(ctx, in.Bank, in.Item)
		if err != nil {
			return err
		}
		if output == nil {
			return nil
		}
		typed, ok := output.(*contracts.MemoryRetainResponse)
		if !ok {
			return fmt.Errorf("memory.retain: output must be *contracts.MemoryRetainResponse")
		}
		*typed = contracts.MemoryRetainResponse{ID: id}
		return nil
	}
}

func recallMemoryTool(store contracts.MemoryStore) LocalTool {
	return func(ctx context.Context, input any, output any) error {
		var in contracts.MemoryRecallRequest
		if err := decodeJSONLike(input, &in); err != nil {
			return fmt.Errorf("memory.recall: %w", err)
		}
		if strings.TrimSpace(in.Bank) == "" {
			return fmt.Errorf("memory.recall: bank is required")
		}
		in.Query.Tags = contracts.NormalizeMemoryTags(in.Query.Tags)

		log.Printf("memory.recall bank=%s q=%s", in.Bank, in.Query.Q)

		items, err := store.Recall(ctx, in.Bank, in.Query)
		if err != nil {
			return err
		}
		if output == nil {
			return nil
		}
		typed, ok := output.(*contracts.MemoryRecallResponse)
		if !ok {
			return fmt.Errorf("memory.recall: output must be *contracts.MemoryRecallResponse")
		}
		*typed = contracts.MemoryRecallResponse{Items: items}
		return nil
	}
}

func reflectMemoryTool(store contracts.MemoryStore) LocalTool {
	return func(ctx context.Context, input any, output any) error {
		var in contracts.MemoryReflectRequest
		if err := decodeJSONLike(input, &in); err != nil {
			return fmt.Errorf("memory.reflect: %w", err)
		}
		if strings.TrimSpace(in.Bank) == "" {
			return fmt.Errorf("memory.reflect: bank is required")
		}
		if strings.TrimSpace(in.Params.Query) == "" {
			return fmt.Errorf("memory.reflect: params.query is required")
		}

		log.Printf("memory.reflect bank=%s window_days=%d", in.Bank, in.Params.WindowDays)

		items, err := store.Reflect(ctx, in.Bank, in.Params)
		if err != nil {
			return err
		}
		if output == nil {
			return nil
		}
		typed, ok := output.(*contracts.MemoryReflectResponse)
		if !ok {
			return fmt.Errorf("memory.reflect: output must be *contracts.MemoryReflectResponse")
		}
		*typed = contracts.MemoryReflectResponse{Items: items}
		return nil
	}
}
