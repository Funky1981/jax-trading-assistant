package harness

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5/pgxpool"

	"jax-trading-assistant/libs/chattools"
)

func RegisterDefaultTools(reg *Registry, pool *pgxpool.Pool) error {
	for _, tool := range chattools.DefaultTools() {
		spec := tool
		if err := reg.Register(ToolDefinition{
			Name:                 spec.Name,
			Description:          spec.Description,
			ReadOnly:             spec.ReadOnly,
			InputSchemaHint:      spec.ArgKey,
			OutputKind:           "json",
			EvidenceLevel:        evidenceLevelFromString(spec.EvidenceLevel),
			FreshnessExpectation: spec.FreshnessExpectation,
			Handler: func(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
				return spec.Handler(ctx, pool, args)
			},
		}); err != nil {
			return err
		}
	}
	return nil
}
