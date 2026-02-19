package app

import "context"

type correlationIDKey struct{}

func WithCorrelationID(ctx context.Context, id string) context.Context {
	if id == "" {
		return ctx
	}
	return context.WithValue(ctx, correlationIDKey{}, id)
}

func CorrelationIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if v := ctx.Value(correlationIDKey{}); v != nil {
		if id, ok := v.(string); ok {
			return id
		}
	}
	return ""
}

func EnsureCorrelationID(ctx context.Context) context.Context {
	if CorrelationIDFromContext(ctx) != "" {
		return ctx
	}
	return WithCorrelationID(ctx, NewCorrelationID())
}
