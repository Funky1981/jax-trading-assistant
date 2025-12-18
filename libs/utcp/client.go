package utcp

import "context"

type Client interface {
	CallTool(ctx context.Context, providerID, toolName string, input any, output any) error
}
